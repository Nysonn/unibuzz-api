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
	cacheKey := "feed:v2:latest"

	if val, err := h.redis.Get(c, cacheKey).Result(); err == nil {
		var cached any
		json.Unmarshal([]byte(val), &cached)
		c.JSON(http.StatusOK, cached)
		return
	}

	rows, err := h.db.Query(c, `
		SELECT v.id, v.user_id, v.caption, v.video_url, v.thumbnail_url, v.created_at,
		       u.username, u.university_name, u.year_of_study,
		       COALESCE(array_agg(DISTINCT h.tag) FILTER (WHERE h.tag IS NOT NULL), '{}') AS hashtags
		FROM videos v
		LEFT JOIN users u ON u.id = v.user_id
		LEFT JOIN video_hashtags vh ON vh.video_id = v.id
		LEFT JOIN hashtags h ON h.id = vh.hashtag_id
		WHERE v.deleted_at IS NULL
		  AND v.status = 'processed'
		GROUP BY v.id, v.user_id, v.caption, v.video_url, v.thumbnail_url, v.created_at,
		         u.username, u.university_name, u.year_of_study
		ORDER BY v.created_at DESC
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
		var username *string
		var universityName *string
		var yearOfStudy *int32
		var hashtags []string
		rows.Scan(&id, &userID, &caption, &videoURL, &thumbnailURL, &createdAt,
			&username, &universityName, &yearOfStudy, &hashtags)
		if hashtags == nil {
			hashtags = []string{}
		}
		videos = append(videos, gin.H{
			"id":              uuid.UUID(id).String(),
			"user_id":         uuid.UUID(userID).String(),
			"username":        username,
			"university_name": universityName,
			"year_of_study":   yearOfStudy,
			"caption":         caption,
			"hashtags":        hashtags,
			"video_url":       videoURL,
			"thumbnail_url":   thumbnailURL,
			"created_at":      createdAt,
		})
	}

	if videos == nil {
		videos = []gin.H{}
	}

	data, _ := json.Marshal(videos)
	h.redis.Set(c, cacheKey, data, 30*time.Second)

	c.JSON(http.StatusOK, videos)
}

// cloudinaryThumbnailURL derives a Cloudinary thumbnail image URL from a Cloudinary video URL
// by inserting the so_1,f_jpg transformation segment.
// Returns an empty string if the URL is not hosted on Cloudinary.
func cloudinaryThumbnailURL(videoURL string) string {
	if !strings.Contains(videoURL, "cloudinary.com") {
		return ""
	}
	thumb := strings.Replace(videoURL, "/video/upload/", "/video/upload/so_1,f_jpg/", 1)
	// Swap the file extension to .jpg
	if dot := strings.LastIndex(thumb, "."); dot > strings.LastIndex(thumb, "/") {
		thumb = thumb[:dot] + ".jpg"
	}
	return thumb
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

	// Only Mbarara University students (@std.must.ac.ug) may upload videos.
	var userEmail string
	err = h.db.QueryRow(c, `SELECT email FROM users WHERE id = $1 AND deleted_at IS NULL`, userUUID).Scan(&userEmail)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "could not verify account"})
		return
	}
	if !strings.HasSuffix(strings.ToLower(userEmail), "@std.must.ac.ug") {
		c.JSON(http.StatusForbidden, gin.H{"error": "Video uploads are available to Mbarara University students only. Please sign in with your university email (@std.must.ac.ug) to upload."})
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

	// 2. If the video is already hosted on Cloudinary (client uploads directly), skip the
	// worker queue — derive the thumbnail via URL transformation and mark as processed
	// immediately so the video appears in the feed right away.
	if thumbnailURL := cloudinaryThumbnailURL(inputURL); thumbnailURL != "" {
		_, err = h.db.Exec(c, `
			UPDATE videos
			SET video_url = $1, thumbnail_url = $2, status = 'processed', updated_at = NOW()
			WHERE id = $3
		`, inputURL, thumbnailURL, videoID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update video record"})
			return
		}
		// Bust the 30-second feed cache so the new video is visible immediately.
		h.redis.Del(c, "feed:v2:latest")
		c.JSON(http.StatusAccepted, gin.H{
			"message":  "video accepted and processed",
			"video_id": videoID,
			"status":   "processed",
			"tags":     tags,
		})
		return
	}

	// Non-Cloudinary URL: enqueue the processing job for the FFmpeg worker.
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

// GET /api/videos/:id/status — lets the client poll until status becomes "processed" or "failed".
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

// GET /api/me/videos — returns all videos owned by the authenticated user,
// including pending ones, with inline vote and comment counts.
func (h *VideoHandler) GetMyVideos(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	rows, err := h.db.Query(c, `
		SELECT
			v.id, v.user_id, v.caption, v.video_url, v.thumbnail_url,
			v.created_at, v.status,
			COALESCE(array_agg(DISTINCT h.tag) FILTER (WHERE h.tag IS NOT NULL), '{}') AS hashtags,
			COALESCE((SELECT COUNT(*) FROM votes    WHERE video_id = v.id AND vote_type =  1), 0) AS upvotes,
			COALESCE((SELECT COUNT(*) FROM votes    WHERE video_id = v.id AND vote_type = -1), 0) AS downvotes,
			COALESCE((SELECT COUNT(*) FROM comments WHERE video_id = v.id AND deleted_at IS NULL), 0) AS comment_count
		FROM videos v
		LEFT JOIN video_hashtags vh ON vh.video_id = v.id
		LEFT JOIN hashtags h ON h.id = vh.hashtag_id
		WHERE v.user_id = $1 AND v.deleted_at IS NULL
		GROUP BY v.id, v.user_id, v.caption, v.video_url, v.thumbnail_url, v.created_at, v.status
		ORDER BY v.created_at DESC
		LIMIT 100
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch videos"})
		return
	}
	defer rows.Close()

	var videos []gin.H
	for rows.Next() {
		var id, ownerID [16]byte
		var caption, videoURL, thumbnailURL, createdAt any
		var status string
		var hashtags []string
		var upvotes, downvotes, commentCount int64

		if err := rows.Scan(
			&id, &ownerID, &caption, &videoURL, &thumbnailURL,
			&createdAt, &status, &hashtags,
			&upvotes, &downvotes, &commentCount,
		); err != nil {
			continue
		}
		if hashtags == nil {
			hashtags = []string{}
		}
		videos = append(videos, gin.H{
			"id":            uuid.UUID(id).String(),
			"user_id":       uuid.UUID(ownerID).String(),
			"caption":       caption,
			"video_url":     videoURL,
			"thumbnail_url": thumbnailURL,
			"created_at":    createdAt,
			"status":        status,
			"hashtags":      hashtags,
			"upvotes":       upvotes,
			"downvotes":     downvotes,
			"comments":      commentCount,
		})
	}

	if videos == nil {
		videos = []gin.H{}
	}

	c.JSON(http.StatusOK, videos)
}
