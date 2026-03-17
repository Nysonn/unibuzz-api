package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

// GET /api/me/comment-filters — lists all comment keyword filters for the authenticated user.
func (h *UserHandler) GetCommentFilters(c *gin.Context) {
	userID, _ := c.Get("user_id")

	rows, err := h.db.Query(c,
		`SELECT id, keyword, created_at FROM user_comment_filters WHERE user_id=$1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch filters"})
		return
	}
	defer rows.Close()

	filters := []gin.H{}
	for rows.Next() {
		var id [16]byte
		var keyword string
		var createdAt any
		rows.Scan(&id, &keyword, &createdAt)
		filters = append(filters, gin.H{
			"id":         uuid.UUID(id).String(),
			"keyword":    keyword,
			"created_at": createdAt,
		})
	}

	c.JSON(http.StatusOK, filters)
}

type addCommentFilterRequest struct {
	Keyword string `json:"keyword" binding:"required,min=1,max=100"`
}

// POST /api/me/comment-filters — adds a keyword filter (max 50 per user).
// Comments containing this keyword (case-insensitive) will be hidden on all of the user's videos.
func (h *UserHandler) AddCommentFilter(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var req addCommentFilterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: keyword is required (1–100 characters)"})
		return
	}

	// Enforce the 50-keyword cap.
	var count int
	if err := h.db.QueryRow(c,
		`SELECT COUNT(*) FROM user_comment_filters WHERE user_id=$1`,
		userID,
	).Scan(&count); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check filter count"})
		return
	}
	if count >= 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "maximum of 50 filter keywords reached"})
		return
	}

	// Store keyword in lowercase for consistent matching.
	// ON CONFLICT DO NOTHING lets us detect duplicates via pgx.ErrNoRows.
	var filterID string
	err := h.db.QueryRow(c,
		`INSERT INTO user_comment_filters (user_id, keyword)
		 VALUES ($1, LOWER($2))
		 ON CONFLICT (user_id, keyword) DO NOTHING
		 RETURNING id`,
		userID, req.Keyword,
	).Scan(&filterID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusConflict, gin.H{"error": "keyword already exists in your filter list"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add filter"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "filter added", "filter_id": filterID})
}

// DELETE /api/me/comment-filters/:filter_id — removes a keyword filter.
func (h *UserHandler) DeleteCommentFilter(c *gin.Context) {
	filterID := c.Param("filter_id")
	userID, _ := c.Get("user_id")

	tag, err := h.db.Exec(c,
		`DELETE FROM user_comment_filters WHERE id=$1 AND user_id=$2`,
		filterID, userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete filter"})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "filter not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "filter removed"})
}
