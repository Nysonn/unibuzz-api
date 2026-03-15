package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SearchHandler struct {
	db *pgxpool.Pool
}

func NewSearchHandler(db *pgxpool.Pool) *SearchHandler {
	return &SearchHandler{db: db}
}

// GET /api/search?tag=<query>&username=<query>
// At least one param is required. Both can be combined. No authentication required.
// Supports partial (ILIKE) matching. Returns up to 50 processed videos, newest first.
func (h *SearchHandler) Search(c *gin.Context) {
	tag := c.Query("tag")
	username := c.Query("username")

	if tag == "" && username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one of 'tag' or 'username' query params is required"})
		return
	}

	query := `
		SELECT DISTINCT v.id, v.user_id, v.caption, v.video_url, v.thumbnail_url, v.created_at
		FROM videos v
		LEFT JOIN users u ON v.user_id = u.id
		LEFT JOIN video_hashtags vh ON v.id = vh.video_id
		LEFT JOIN hashtags ht ON vh.hashtag_id = ht.id
		WHERE v.deleted_at IS NULL
		  AND v.status = 'processed'
	`

	args := []any{}
	argIdx := 1

	if tag != "" {
		query += fmt.Sprintf(" AND ht.tag ILIKE $%d", argIdx)
		args = append(args, "%"+tag+"%")
		argIdx++
	}

	if username != "" {
		query += fmt.Sprintf(" AND u.username ILIKE $%d", argIdx)
		args = append(args, "%"+username+"%")
	}

	query += " ORDER BY v.created_at DESC LIMIT 50"

	rows, err := h.db.Query(c, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
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

	c.JSON(http.StatusOK, videos)
}
