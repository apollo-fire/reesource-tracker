package emails

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"reesource-tracker/api/middleware"
	libauth "reesource-tracker/lib/auth"
	"reesource-tracker/lib/database"
	id_helper "reesource-tracker/lib/id_helper"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers the unauthenticated email registration endpoint.
// This allows an assignment link to be used to add an email address (instead
// of, or in addition to, a passkey).
func RegisterRoutes(authRoutes *gin.RouterGroup) {
	authRoutes.POST("/email/register", registerEmail)
}

// SelfRoutes registers authenticated routes for the current user.
func SelfRoutes(self *gin.RouterGroup) {
	self.GET("/emails", listSelfEmails)
	self.POST("/emails", addSelfEmail)
	self.POST("/emails/:id/remove", middleware.RequireConfirmedAction(), removeSelfEmail)
}

// AdminRoutes registers admin-only email management routes.
func AdminRoutes(admin *gin.RouterGroup) {
	admin.GET("/users/:user_id/emails", adminListUserEmails)
	admin.POST("/users/:user_id/emails", adminAddUserEmail)
	admin.POST("/users/:user_id/emails/:id/remove", middleware.RequireConfirmedAction(), adminRemoveUserEmail)
}

// registerEmail adds an email address to the user identified by an assignment
// link, then consumes the link. It mirrors the passkey registration flow so
// that the account-setup page can offer either method from the same link.
func registerEmail(c *gin.Context) {
	var req struct {
		AssignmentToken string `json:"assignment_token"`
		Email           string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.AssignmentToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assignment_token is required"})
		return
	}
	if strings.TrimSpace(req.Email) == "" || !strings.Contains(req.Email, "@") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "a valid email is required"})
		return
	}
	email := strings.TrimSpace(req.Email)

	link, err := database.Connection.GetActiveAssignmentLinkByTokenHash(c, libauth.HashToken(req.AssignmentToken))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired assignment link"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := database.Connection.InsertUserEmail(c, database.InsertUserEmailParams{
		UserID: link.UserID,
		Email:  email,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	_ = database.Connection.ConsumeAssignmentLink(c, link.ID)

	if link.Purpose == "bootstrap" {
		_ = database.Connection.SetUserRole(c, database.SetUserRoleParams{
			UserID: link.UserID,
			Role:   libauth.RoleAdmin,
		})
	}

	userIDStr := libauth.IDOrEmpty(link.UserID)
	_ = libauth.AuditLog(c, nil, "email_registered", "user", userIDStr, map[string]any{"email": email})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func listSelfEmails(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	rows, err := database.Connection.ListUserEmails(c, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, emailListResponse(rows))
}

func addSelfEmail(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	var req struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Email) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
		return
	}
	if !strings.Contains(req.Email, "@") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email address"})
		return
	}
	email := strings.TrimSpace(req.Email)
	if err := database.Connection.InsertUserEmail(c, database.InsertUserEmailParams{UserID: userID, Email: email}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = libauth.AuditLog(c, &userID, "user_email_added", "user", libauth.IDOrEmpty(userID), map[string]any{"email": email})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func removeSelfEmail(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	// Fetch first to confirm ownership and get the email for audit log.
	rows, err := database.Connection.ListUserEmails(c, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var target string
	for _, r := range rows {
		if r.ID == id {
			target = r.Email
			break
		}
	}
	if target == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "email not found"})
		return
	}
	if err := database.Connection.DeleteUserEmail(c, database.DeleteUserEmailParams{UserID: userID, Email: target}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = libauth.AuditLog(c, &userID, "user_email_removed", "user", libauth.IDOrEmpty(userID), map[string]any{"email": target})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func adminListUserEmails(c *gin.Context) {
	uid, msg, ok := id_helper.MustParseAndMarshalUUID(c.Param("user_id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	rows, err := database.Connection.ListUserEmails(c, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, emailListResponse(rows))
}

func adminAddUserEmail(c *gin.Context) {
	uid, msg, ok := id_helper.MustParseAndMarshalUUID(c.Param("user_id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	var req struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Email) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
		return
	}
	if !strings.Contains(req.Email, "@") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email address"})
		return
	}
	email := strings.TrimSpace(req.Email)
	if err := database.Connection.InsertUserEmail(c, database.InsertUserEmailParams{UserID: uid, Email: email}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	actor, _ := middleware.CurrentUserID(c)
	_ = libauth.AuditLog(c, &actor, "user_email_added", "user", c.Param("user_id"), map[string]any{"email": email, "scope": "admin"})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func adminRemoveUserEmail(c *gin.Context) {
	uid, msg, ok := id_helper.MustParseAndMarshalUUID(c.Param("user_id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	rows, err := database.Connection.ListUserEmails(c, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var target string
	for _, r := range rows {
		if r.ID == id {
			target = r.Email
			break
		}
	}
	if target == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "email not found"})
		return
	}
	if err := database.Connection.DeleteUserEmail(c, database.DeleteUserEmailParams{UserID: uid, Email: target}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	actor, _ := middleware.CurrentUserID(c)
	_ = libauth.AuditLog(c, &actor, "user_email_removed", "user", c.Param("user_id"), map[string]any{"email": target, "scope": "admin"})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func emailListResponse(rows []database.UserEmail) []gin.H {
	res := make([]gin.H, 0, len(rows))
	for _, e := range rows {
		res = append(res, gin.H{
			"id":         e.ID,
			"email":      e.Email,
			"created_at": e.CreatedAt,
		})
	}
	return res
}
