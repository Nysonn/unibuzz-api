package handlers

import (
	"net/http"

	"github.com/Nysonn/unibuzz-api/internal/dto"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CommentHandler struct {
	db *pgxpool.Pool
}

func NewCommentHandler(db *pgxpool.Pool) *CommentHandler {
	return &CommentHandler{db: db}
}

func (h *CommentHandler) CreateComment(c *gin.Context) {
	videoID := c.Param("id")
	userID, _ := c.Get("user_id")

	var req dto.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// Block new comments if the video owner has disabled them.
	var commentsDisabled bool
	err := h.db.QueryRow(c,
		`SELECT comments_disabled FROM videos WHERE id=$1 AND deleted_at IS NULL`,
		videoID,
	).Scan(&commentsDisabled)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "video not found"})
		return
	}
	if commentsDisabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "comments are disabled for this video"})
		return
	}

	var commentID string
	err = h.db.QueryRow(c,
		`INSERT INTO comments (user_id, video_id, content) VALUES ($1,$2,$3) RETURNING id`,
		userID, videoID, req.Comment,
	).Scan(&commentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "comment failed"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "comment added", "comment_id": commentID})
}

func (h *CommentHandler) GetVideoComments(c *gin.Context) {
	videoID := c.Param("id")

	// Return early with a clear flag when comments are disabled.
	var commentsDisabled bool
	err := h.db.QueryRow(c,
		`SELECT comments_disabled FROM videos WHERE id=$1 AND deleted_at IS NULL`,
		videoID,
	).Scan(&commentsDisabled)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "video not found"})
		return
	}
	if commentsDisabled {
		c.JSON(http.StatusOK, gin.H{"comments_disabled": true, "comments": []gin.H{}})
		return
	}

	// Fetch comments, excluding any that match a keyword in the video owner's filter list.
	// Keyword matching is case-insensitive and partial (substring).
	rows, err := h.db.Query(c, `
		SELECT c.id, c.user_id, c.video_id, c.content, c.created_at, c.updated_at, u.username
		FROM comments c
		JOIN videos v ON v.id = c.video_id
		JOIN users u ON u.id = c.user_id
		WHERE c.video_id = $1
		  AND c.deleted_at IS NULL
		  AND NOT EXISTS (
		    SELECT 1
		    FROM user_comment_filters ucf
		    WHERE ucf.user_id = v.user_id
		      AND LOWER(c.content) LIKE '%' || LOWER(ucf.keyword) || '%'
		  )
		ORDER BY c.created_at DESC
		LIMIT 20
	`, videoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch comments"})
		return
	}
	defer rows.Close()

	comments := []gin.H{}
	for rows.Next() {
		var id, userID, vid [16]byte
		var content, username string
		var createdAt, updatedAt any
		rows.Scan(&id, &userID, &vid, &content, &createdAt, &updatedAt, &username)
		comments = append(comments, gin.H{
			"id":         uuid.UUID(id).String(),
			"user_id":    uuid.UUID(userID).String(),
			"video_id":   uuid.UUID(vid).String(),
			"content":    content,
			"username":   username,
			"created_at": createdAt,
			"updated_at": updatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"comments_disabled": false, "comments": comments})
}

// ToggleComments enables or disables the comment section for a video.
// Only the video owner can call this endpoint.
func (h *CommentHandler) ToggleComments(c *gin.Context) {
	videoID := c.Param("id")
	userID, _ := c.Get("user_id")
	callerID, _ := userID.(uuid.UUID)

	var ownerIDBytes [16]byte
	var currentDisabled bool
	err := h.db.QueryRow(c,
		`SELECT user_id, comments_disabled FROM videos WHERE id=$1 AND deleted_at IS NULL`,
		videoID,
	).Scan(&ownerIDBytes, &currentDisabled)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "video not found"})
		return
	}

	if uuid.UUID(ownerIDBytes) != callerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "only the video owner can toggle comments"})
		return
	}

	newState := !currentDisabled
	_, err = h.db.Exec(c,
		`UPDATE videos SET comments_disabled=$1, updated_at=NOW() WHERE id=$2`,
		newState, videoID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to toggle comments"})
		return
	}

	msg := "comments enabled"
	if newState {
		msg = "comments disabled"
	}
	c.JSON(http.StatusOK, gin.H{"message": msg, "comments_disabled": newState})
}

func (h *CommentHandler) UpdateComment(c *gin.Context) {
	commentID := c.Param("comment_id")
	userID, _ := c.Get("user_id")

	var req dto.UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	tag, err := h.db.Exec(c,
		`UPDATE comments SET content=$1, updated_at=NOW() WHERE id=$2 AND user_id=$3 AND deleted_at IS NULL`,
		req.Comment, commentID, userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "not allowed or comment not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "comment updated"})
}

func (h *CommentHandler) DeleteComment(c *gin.Context) {
	commentID := c.Param("comment_id")
	userID, _ := c.Get("user_id")

	tag, err := h.db.Exec(c,
		`UPDATE comments SET deleted_at=NOW() WHERE id=$1 AND user_id=$2 AND deleted_at IS NULL`,
		commentID, userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "not allowed or comment not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "comment deleted"})
}
