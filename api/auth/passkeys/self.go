package passkeys

import (
	"bytes"
	"database/sql"
	"errors"
	"net/http"

	"reesource-tracker/api/middleware"
	libauth "reesource-tracker/lib/auth"
	"reesource-tracker/lib/database"

	"github.com/gin-gonic/gin"
)

// SelfRoutes registers authenticated routes for the current user's passkeys
// under the provided /self group.
func SelfRoutes(self *gin.RouterGroup) {
	self.GET("/passkeys", listSelfPasskeys)
	self.POST("/passkeys/:credential_id/revoke", middleware.RequireConfirmedAction(), revokeSelfPasskey)
}

func listSelfPasskeys(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	rows, err := database.Connection.ListPasskeysByUser(c, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	activeCredentialID, _ := middleware.CurrentCredentialID(c)
	c.JSON(http.StatusOK, libauth.PasskeyListMap(rows, activeCredentialID))
}

func revokeSelfPasskey(c *gin.Context) {
	userID, _ := middleware.CurrentUserID(c)
	credentialID, credentialIDHex, err := libauth.DecodeCredentialID(c.Param("credential_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credential_id"})
		return
	}
	passkey, err := database.Connection.GetPasskeyByCredentialID(c, credentialID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "passkey not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !bytes.Equal(passkey.UserID, userID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	activeCredentialID, ok := middleware.CurrentCredentialID(c)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "reauthentication required before removing passkeys"})
		return
	}
	if bytes.Equal(activeCredentialID, credentialID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot remove the passkey used by your current session"})
		return
	}
	if err := database.Connection.RevokePasskey(c, credentialID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = libauth.AuditLog(c, &userID, "passkey_revoked", "passkey", credentialIDHex, map[string]any{"scope": "self"})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
