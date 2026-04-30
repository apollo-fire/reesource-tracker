package assignmentlinks

import (
	"database/sql"
	"errors"
	"net/http"

	"reesource-tracker/api/middleware"
	libauth "reesource-tracker/lib/auth"
	"reesource-tracker/lib/database"

	"github.com/gin-gonic/gin"
)

// SelfRoutes registers authenticated routes for managing the current user's
// assignment link under the provided /self group.
func SelfRoutes(self *gin.RouterGroup) {
	self.POST("/assignment-link", createSelfAssignmentLink)
	self.GET("/assignment-link", getSelfAssignmentLink)
	self.DELETE("/assignment-link", middleware.RequireConfirmedAction(), deleteSelfAssignmentLink)
}

func createSelfAssignmentLink(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	row, rawToken, err := libauth.CreateStandardAssignmentLinkForUser(c, userID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, libauth.AssignmentLinkMap(c.Request, row, rawToken))
}

func getSelfAssignmentLink(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	row, err := database.Connection.GetActiveStandardAssignmentLinkByUserID(c, userID)
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

func deleteSelfAssignmentLink(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	revoked, err := database.Connection.RevokeActiveStandardAssignmentLinksForUser(c, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if revoked > 0 {
		_ = libauth.AuditLog(c, &userID, "assignment_link_revoked", "user", libauth.IDOrEmpty(userID), map[string]any{"revoked_count": revoked, "scope": "self"})
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "revoked_count": revoked})
}
