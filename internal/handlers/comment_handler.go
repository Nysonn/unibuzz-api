package handlers

import (
	"net/http"

	"github.com/Nysonn/unibuzz-api/internal/dto"
	"github.com/gin-gonic/gin"
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

	var commentID string
	err := h.db.QueryRow(c,
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

	rows, err := h.db.Query(c, `
	SELECT id, user_id, video_id, content, created_at, updated_at
	FROM comments
	WHERE video_id=$1 AND deleted_at IS NULL
	ORDER BY created_at DESC
	LIMIT 20
	`, videoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch comments"})
		return
	}
	defer rows.Close()

	var comments []gin.H
	for rows.Next() {
		var id, userID, vid, content, createdAt, updatedAt any
		rows.Scan(&id, &userID, &vid, &content, &createdAt, &updatedAt)
		comments = append(comments, gin.H{
			"id":         id,
			"user_id":    userID,
			"video_id":   vid,
			"content":    content,
			"created_at": createdAt,
			"updated_at": updatedAt,
		})
	}

	c.JSON(http.StatusOK, comments)
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
