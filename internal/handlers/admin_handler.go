package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AdminHandler struct {
	db *pgxpool.Pool
}

func NewAdminHandler(db *pgxpool.Pool) *AdminHandler {
	return &AdminHandler{db: db}
}

func (h *AdminHandler) GetReports(c *gin.Context) {
	query := `
	SELECT id, reporter_id, video_id, reason, resolved, created_at
	FROM reports
	ORDER BY created_at DESC
	`

	rows, err := h.db.Query(c, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch reports"})
		return
	}
	defer rows.Close()

	var reports []gin.H
	for rows.Next() {
		var id, reporterID, videoID, reason, resolved, createdAt any
		rows.Scan(&id, &reporterID, &videoID, &reason, &resolved, &createdAt)
		reports = append(reports, gin.H{
			"id":          id,
			"reporter_id": reporterID,
			"video_id":    videoID,
			"reason":      reason,
			"resolved":    resolved,
			"created_at":  createdAt,
		})
	}

	c.JSON(http.StatusOK, reports)
}

func (h *AdminHandler) SuspendUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	adminID, _ := c.Get("user_id")

	_, err = h.db.Exec(c, `UPDATE users SET is_suspended=true WHERE id=$1`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to suspend user"})
		return
	}

	h.db.Exec(c,
		`INSERT INTO admin_actions (admin_id, target_user_id, action_type) VALUES ($1,$2,'suspend_user')`,
		adminID, userID,
	)

	c.JSON(http.StatusOK, gin.H{"message": "user suspended"})
}

func (h *AdminHandler) UnsuspendUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	_, err = h.db.Exec(c, `UPDATE users SET is_suspended=false WHERE id=$1`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unsuspend user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user unsuspended"})
}

func (h *AdminHandler) BanUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	_, err = h.db.Exec(c, `UPDATE users SET banned=true WHERE id=$1`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to ban user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user banned"})
}

func (h *AdminHandler) AdminGetUsers(c *gin.Context) {
	adminID, _ := c.Get("user_id")

	query := `
	SELECT id, full_name, username, email, university_name,
	course, year_of_study, is_suspended, banned, created_at
	FROM users
	WHERE deleted_at IS NULL
	  AND id != $1
	ORDER BY created_at DESC
	`

	rows, err := h.db.Query(c, query, adminID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch users"})
		return
	}
	defer rows.Close()

	var users []gin.H
	for rows.Next() {
		var id [16]byte
		var fullName, username, email, university, course, year, suspended, banned, createdAt any
		rows.Scan(&id, &fullName, &username, &email, &university, &course, &year, &suspended, &banned, &createdAt)
		users = append(users, gin.H{
			"id":            uuid.UUID(id).String(),
			"full_name":     fullName,
			"username":      username,
			"email":         email,
			"university":    university,
			"course":        course,
			"year_of_study": year,
			"is_suspended":  suspended,
			"banned":        banned,
			"created_at":    createdAt,
		})
	}

	c.JSON(http.StatusOK, users)
}

func (h *AdminHandler) AdminGetUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	row := h.db.QueryRow(c, `
	SELECT id, full_name, username, email,
	university_name, course, year_of_study,
	is_suspended, banned, created_at
	FROM users
	WHERE id=$1 AND deleted_at IS NULL
	`, id)

	var uid [16]byte
	var fullName, username, email, university, course, year, suspended, banned, createdAt any
	if err := row.Scan(&uid, &fullName, &username, &email, &university, &course, &year, &suspended, &banned, &createdAt); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":            uuid.UUID(uid).String(),
		"full_name":     fullName,
		"username":      username,
		"email":         email,
		"university":    university,
		"course":        course,
		"year_of_study": year,
		"is_suspended":  suspended,
		"banned":        banned,
		"created_at":    createdAt,
	})
}

func (h *AdminHandler) AdminDeleteUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	_, err = h.db.Exec(c, `UPDATE users SET deleted_at=NOW() WHERE id=$1`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user deleted"})
}

func (h *AdminHandler) AdminDeleteVideo(c *gin.Context) {
	videoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid video id"})
		return
	}
	adminID, _ := c.Get("user_id")

	_, err = h.db.Exec(c, `UPDATE videos SET deleted_at=NOW() WHERE id=$1`, videoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete video"})
		return
	}

	h.db.Exec(c,
		`INSERT INTO admin_actions (admin_id, target_user_id, action_type) VALUES ($1,$2,'delete_video')`,
		adminID, videoID,
	)

	c.JSON(http.StatusOK, gin.H{"message": "video removed"})
}
