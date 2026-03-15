package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Nysonn/unibuzz-api/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type VideoHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewVideoHandler(db *pgxpool.Pool, redis *redis.Client) *VideoHandler {
	return &VideoHandler{db: db, redis: redis}
}

// GET /api/feed — returns the 20 most recent processed videos, cached in Redis for 30s.
func (h *VideoHandler) GetFeed(c *gin.Context) {
	cacheKey := "feed:latest"

	if val, err := h.redis.Get(c, cacheKey).Result(); err == nil {
		var cached any
		json.Unmarshal([]byte(val), &cached)
		c.JSON(http.StatusOK, cached)
		return
	}

	rows, err := h.db.Query(c, `
		SELECT id, user_id, caption, video_url, thumbnail_url, created_at
		FROM videos
		WHERE deleted_at IS NULL
		  AND status = 'processed'
		ORDER BY created_at DESC
		LIMIT 20
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "feed error"})
		return
	}
	defer rows.Close()

	var videos []gin.H
	for rows.Next() {
		var id, userID [16]byte
		var caption, videoURL, thumbnailURL, createdAt any
		rows.Scan(&id, &userID, &caption, &videoURL, &thumbnailURL, &createdAt)
		videos = append(videos, gin.H{
			"id":            uuid.UUID(id).String(),
			"user_id":       uuid.UUID(userID).String(),
			"caption":       caption,
			"video_url":     videoURL,
			"thumbnail_url": thumbnailURL,
			"created_at":    createdAt,
		})
	}

	if videos == nil {
		videos = []gin.H{}
	}

	data, _ := json.Marshal(videos)
	h.redis.Set(c, cacheKey, data, 30*time.Second)

	c.JSON(http.StatusOK, videos)
}

// resolveVideoURL rewrites a Cloudinary embed player URL to a direct video URL.
// e.g. https://player.cloudinary.com/embed/?cloud_name=X&public_id=Y
//
//	→ https://res.cloudinary.com/X/video/upload/Y.mp4
//
// Non-Cloudinary-embed URLs are returned unchanged.
func resolveVideoURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid url: %w", err)
	}
	if u.Host == "player.cloudinary.com" && u.Path == "/embed/" {
		cloudName := u.Query().Get("cloud_name")
		publicID := u.Query().Get("public_id")
		if cloudName == "" || publicID == "" {
			return "", fmt.Errorf("cloudinary embed URL missing cloud_name or public_id")
		}
		return fmt.Sprintf("https://res.cloudinary.com/%s/video/upload/%s.mp4", cloudName, publicID), nil
	}
	return rawURL, nil
}

// POST /api/videos/upload?input_url=<url>&caption=<text>
// Creates a pending video row in the DB, then enqueues it for FFmpeg + Cloudinary processing.
// Returns the new video_id immediately so the client can poll for status.
func (h *VideoHandler) UploadVideo(c *gin.Context) {
	rawInputURL := c.Query("input_url")
	if rawInputURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "input_url query param is required"})
		return
	}
	inputURL, err := resolveVideoURL(rawInputURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user id"})
		return
	}

	caption := c.DefaultQuery("caption", "")

	// Parse hashtags from comma-separated query param, normalize to lowercase, deduplicate.
	var tags []string
	if rawTags := c.Query("tags"); rawTags != "" {
		seen := map[string]bool{}
		for _, t := range strings.Split(rawTags, ",") {
			tag := strings.ToLower(strings.TrimSpace(t))
			if tag != "" && !seen[tag] {
				tags = append(tags, tag)
				seen[tag] = true
			}
		}
	}
	if len(tags) > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "maximum 10 hashtags allowed"})
		return
	}

	// 1. Insert a pending video row — worker will fill video_url + thumbnail_url later.
	var videoID string
	err = h.db.QueryRow(c, `
		INSERT INTO videos (user_id, caption, status)
		VALUES ($1, $2, 'pending')
		RETURNING id
	`, userUUID, caption).Scan(&videoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create video record"})
		return
	}

	// 1a. Upsert hashtags and link them to the video.
	for _, tag := range tags {
		var hashtagID string
		err = h.db.QueryRow(c, `
			INSERT INTO hashtags (tag) VALUES ($1)
			ON CONFLICT (tag) DO UPDATE SET tag = EXCLUDED.tag
			RETURNING id
		`, tag).Scan(&hashtagID)
		if err != nil {
			continue
		}
		h.db.Exec(c, `
			INSERT INTO video_hashtags (video_id, hashtag_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, videoID, hashtagID)
	}

	// 2. Enqueue the processing job.
	job := models.VideoJob{VideoID: videoID, InputURL: inputURL}
	data, err := json.Marshal(job)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode job"})
		return
	}
	if err := h.redis.RPush(c, "video_jobs", data).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enqueue video"})
		return
	}

	// 3. Return the video_id so the client can poll GET /api/videos/:id/status.
	c.JSON(http.StatusAccepted, gin.H{
		"message":  "video accepted and queued for processing",
		"video_id": videoID,
		"status":   "pending",
		"tags":     tags,
	})
}

// GET /api/videos/:id/status — lets the client poll until status becomes "processed".
func (h *VideoHandler) GetVideoStatus(c *gin.Context) {
	videoID := c.Param("id")

	var status string
	var videoURL, thumbnailURL *string
	err := h.db.QueryRow(c, `
		SELECT status, video_url, thumbnail_url
		FROM videos WHERE id = $1 AND deleted_at IS NULL
	`, videoID).Scan(&status, &videoURL, &thumbnailURL)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "video not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"video_id":      videoID,
		"status":        status,
		"video_url":     videoURL,
		"thumbnail_url": thumbnailURL,
	})
}
