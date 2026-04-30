package passkeys

import (
	"net/http"

	"reesource-tracker/api/middleware"
	libauth "reesource-tracker/lib/auth"
	"reesource-tracker/lib/database"
	id_helper "reesource-tracker/lib/id_helper"

	"github.com/gin-gonic/gin"
)

// AdminRoutes registers admin-only passkey endpoints under the provided /admin group.
func AdminRoutes(admin *gin.RouterGroup) {
	admin.GET("/users/:user_id/passkeys", adminListUserPasskeys)
	admin.POST("/users/:user_id/passkeys/revoke-all", middleware.RequireConfirmedAction(), adminRevokeAllPasskeys)
	admin.POST("/passkeys/:credential_id/revoke", middleware.RequireConfirmedAction(), adminRevokePasskey)
}

func adminRevokeAllPasskeys(c *gin.Context) {
	uid, msg, ok := id_helper.MustParseAndMarshalUUID(c.Param("user_id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	if err := database.Connection.RevokeAllPasskeysForUser(c, uid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	actor, _ := middleware.CurrentUserID(c)
	_ = libauth.AuditLog(c, &actor, "passkeys_revoked_all", "user", c.Param("user_id"), map[string]any{})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func adminRevokePasskey(c *gin.Context) {
	credentialID, credentialIDHex, err := libauth.DecodeCredentialID(c.Param("credential_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credential_id"})
		return
	}
	if err := database.Connection.RevokePasskey(c, credentialID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	actor, _ := middleware.CurrentUserID(c)
	_ = libauth.AuditLog(c, &actor, "passkey_revoked", "passkey", credentialIDHex, map[string]any{"scope": "admin"})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func adminListUserPasskeys(c *gin.Context) {
	uid, msg, ok := id_helper.MustParseAndMarshalUUID(c.Param("user_id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	rows, err := database.Connection.ListPasskeysByUser(c, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, libauth.PasskeyListMap(rows, nil))
}


