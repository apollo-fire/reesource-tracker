package assignmentlinks

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"reesource-tracker/api/middleware"
	libauth "reesource-tracker/lib/auth"
	"reesource-tracker/lib/database"
	id_helper "reesource-tracker/lib/id_helper"

	"github.com/gin-gonic/gin"
)

// AdminRoutes registers admin-only assignment link and audit log endpoints
// under the provided /admin group.
func AdminRoutes(admin *gin.RouterGroup) {
	admin.GET("/users/:user_id/assignment-link", adminGetActiveAssignmentLink)
	admin.POST("/users/:user_id/assignment-link", adminCreateAssignmentLink)
	admin.DELETE("/users/:user_id/assignment-link", middleware.RequireConfirmedAction(), adminDeleteActiveAssignmentLink)
	admin.POST("/assignment-links/:link_id/revoke", middleware.RequireConfirmedAction(), adminRevokeAssignmentLink)
	admin.GET("/audit-logs", adminListAuditLogs)
}

func adminCreateAssignmentLink(c *gin.Context) {
	uid, msg, ok := id_helper.MustParseAndMarshalUUID(c.Param("user_id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	actor, _ := middleware.CurrentUserID(c)
	row, rawToken, err := libauth.CreateStandardAssignmentLinkForUser(c, uid, actor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, libauth.AssignmentLinkMap(c.Request, row, rawToken))
}

func adminGetActiveAssignmentLink(c *gin.Context) {
	uid, msg, ok := id_helper.MustParseAndMarshalUUID(c.Param("user_id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	row, err := database.Connection.GetActiveStandardAssignmentLinkByUserID(c, uid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusOK, gin.H{"has_active_link": false})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	res := libauth.AssignmentLinkMap(c.Request, row, "")
	res["has_active_link"] = true
	c.JSON(http.StatusOK, res)
}

func adminDeleteActiveAssignmentLink(c *gin.Context) {
	uid, msg, ok := id_helper.MustParseAndMarshalUUID(c.Param("user_id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	revoked, err := database.Connection.RevokeActiveStandardAssignmentLinksForUser(c, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if revoked > 0 {
		actor, _ := middleware.CurrentUserID(c)
		_ = libauth.AuditLog(c, &actor, "assignment_link_revoked", "user", c.Param("user_id"), map[string]any{"revoked_count": revoked})
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "revoked_count": revoked})
}

func adminRevokeAssignmentLink(c *gin.Context) {
	linkID, err := strconv.ParseInt(c.Param("link_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid link_id"})
		return
	}
	if err := database.Connection.RevokeAssignmentLink(c, linkID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	actor, _ := middleware.CurrentUserID(c)
	_ = libauth.AuditLog(c, &actor, "assignment_link_revoked", "assignment_link", c.Param("link_id"), map[string]any{})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func adminListAuditLogs(c *gin.Context) {
	limit := int32(100)
	offset := int32(0)
	if s := c.Query("limit"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			limit = int32(v)
		}
	}
	if s := c.Query("offset"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v >= 0 {
			offset = int32(v)
		}
	}
	rows, err := database.Connection.ListAuditLogs(c, database.ListAuditLogsParams{Limit: limit, Offset: offset})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rows)
}
