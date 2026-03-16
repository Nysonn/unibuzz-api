package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserHandler struct {
	db *pgxpool.Pool
}

func NewUserHandler(db *pgxpool.Pool) *UserHandler {
	return &UserHandler{db: db}
}

// GET /api/me — returns the authenticated user's profile.
func (h *UserHandler) GetMe(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var id [16]byte
	var fullName, username, email string
	var universityName, course, profilePhotoURL *string
	var yearOfStudy *int32
	var isAdmin, isSuspended *bool
	var createdAt, updatedAt any

	err := h.db.QueryRow(c, `
		SELECT id, full_name, username, email, university_name, course, year_of_study,
		       profile_photo_url, is_admin, is_suspended, created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`, userID).Scan(&id, &fullName, &username, &email,
		&universityName, &course, &yearOfStudy,
		&profilePhotoURL, &isAdmin, &isSuspended,
		&createdAt, &updatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                uuid.UUID(id).String(),
		"full_name":         fullName,
		"username":          username,
		"email":             email,
		"university_name":   universityName,
		"course":            course,
		"year_of_study":     yearOfStudy,
		"profile_photo_url": profilePhotoURL,
		"is_admin":          isAdmin,
		"is_suspended":      isSuspended,
		"created_at":        createdAt,
		"updated_at":        updatedAt,
	})
}

type updateMeRequest struct {
	FullName        *string `json:"full_name"`
	Username        *string `json:"username"`
	UniversityName  *string `json:"university_name"`
	Course          *string `json:"course"`
	YearOfStudy     *int32  `json:"year_of_study"`
	ProfilePhotoURL *string `json:"profile_photo_url"`
}

// PATCH /api/me — updates the authenticated user's profile (only provided fields are changed).
func (h *UserHandler) UpdateMe(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req updateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	_, err := h.db.Exec(c, `
		UPDATE users SET
			full_name         = COALESCE($2, full_name),
			username          = COALESCE($3, username),
			university_name   = COALESCE($4, university_name),
			course            = COALESCE($5, course),
			year_of_study     = COALESCE($6, year_of_study),
			profile_photo_url = COALESCE($7, profile_photo_url),
			updated_at        = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`, userID, req.FullName, req.Username, req.UniversityName,
		req.Course, req.YearOfStudy, req.ProfilePhotoURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "profile updated"})
}
