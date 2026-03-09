package handlers

import (
	"net/http"

	"github.com/Nysonn/unibuzz-api/internal/dto"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReportHandler struct {
	db *pgxpool.Pool
}

func NewReportHandler(db *pgxpool.Pool) *ReportHandler {
	return &ReportHandler{db: db}
}

func (h *ReportHandler) ReportVideo(c *gin.Context) {
	videoID := c.Param("id")
	userID, _ := c.Get("user_id")

	var req dto.ReportVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	_, err := h.db.Exec(c,
		`INSERT INTO reports (reporter_id, video_id, reason) VALUES ($1,$2,$3)`,
		userID, videoID, req.Reason,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit report"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "report submitted"})
}
