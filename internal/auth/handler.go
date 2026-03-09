package auth

import (
	"net/http"

	db "github.com/Nysonn/unibuzz-api/internal/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service}
}

func (h *Handler) Register(c *gin.Context) {

	var req struct {
		FullName    string `json:"full_name"`
		Username    string `json:"username"`
		Email       string `json:"email"`
		Password    string `json:"password"`
		University  string `json:"university_name"`
		Course      string `json:"course"`
		YearOfStudy int32  `json:"year_of_study"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}

	user, err := h.service.Register(c, db.CreateUserParams{
		FullName:       req.FullName,
		Username:       req.Username,
		Email:          req.Email,
		PasswordHash:   req.Password,
		UniversityName: pgtype.Text{String: req.University, Valid: req.University != ""},
		Course:         pgtype.Text{String: req.Course, Valid: req.Course != ""},
		YearOfStudy:    pgtype.Int4{Int32: req.YearOfStudy, Valid: true},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":                user.ID,
		"full_name":         user.FullName,
		"username":          user.Username,
		"email":             user.Email,
		"university_name":   user.UniversityName,
		"course":            user.Course,
		"year_of_study":     user.YearOfStudy,
		"profile_photo_url": user.ProfilePhotoUrl,
		"is_admin":          user.IsAdmin,
		"is_suspended":      user.IsSuspended,
		"created_at":        user.CreatedAt,
		"updated_at":        user.UpdatedAt,
	})
}

func (h *Handler) Login(c *gin.Context) {

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}

	access, refresh, err := h.service.Login(c, req.Email, req.Password)

	if err != nil {
		c.JSON(http.StatusUnauthorized, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  access,
		"refresh_token": refresh,
	})
}
