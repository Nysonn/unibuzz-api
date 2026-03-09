package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VoteHandler struct {
	db *pgxpool.Pool
}

func NewVoteHandler(db *pgxpool.Pool) *VoteHandler {
	return &VoteHandler{db: db}
}

type voteRequest struct {
	VoteType int `json:"vote_type" binding:"required"`
}

func (h *VoteHandler) VoteVideo(c *gin.Context) {
	videoID := c.Param("id")
	userID, _ := c.Get("user_id")

	var req voteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if req.VoteType != 1 && req.VoteType != -1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "vote_type must be 1 or -1"})
		return
	}

	_, err := h.db.Exec(c, `
	INSERT INTO votes (user_id, video_id, vote_type)
	VALUES ($1,$2,$3)
	ON CONFLICT (user_id, video_id)
	DO UPDATE SET vote_type = EXCLUDED.vote_type
	`, userID, videoID, req.VoteType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "vote failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "vote recorded"})
}

func (h *VoteHandler) GetVideoVotes(c *gin.Context) {
	videoID := c.Param("id")

	var upvotes, downvotes int64
	err := h.db.QueryRow(c, `
	SELECT
		COUNT(*) FILTER (WHERE vote_type = 1),
		COUNT(*) FILTER (WHERE vote_type = -1)
	FROM votes
	WHERE video_id=$1
	`, videoID).Scan(&upvotes, &downvotes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not fetch votes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"upvotes":   upvotes,
		"downvotes": downvotes,
	})
}
