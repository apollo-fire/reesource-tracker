package auth

import (
	"context"
	"net/http"
	"os"

	authassignmentlinks "reesource-tracker/api/auth/assignmentlinks"
	authbootstrap "reesource-tracker/api/auth/bootstrap"
	authemails "reesource-tracker/api/auth/emails"
	authpasskeys "reesource-tracker/api/auth/passkeys"
	"reesource-tracker/api/middleware"
	libauth "reesource-tracker/lib/auth"
	"reesource-tracker/lib/database"

	"github.com/gin-gonic/gin"
)

type RuntimeConfig = libauth.RuntimeConfig

func Initialize(ctx context.Context, cfg RuntimeConfig) error {
secret := os.Getenv("SESSION_SECRET")
if secret == "" {
generated, err := libauth.RandomHex(32)
if err != nil {
return err
}
secret = generated
}
middleware.SetCookieSecret([]byte(secret))

if _, _, _, err := libauth.EnsureBootstrapState(ctx); err != nil {
return err
}
libauth.RunAuditRetentionCleanup(ctx, cfg)
return nil
}

func Routes(route *gin.RouterGroup) {
authRoutes := route.Group("/auth")

// Session + feature discovery (not method-specific).
authRoutes.GET("/session", getSession)
authRoutes.POST("/logout", logout)
authRoutes.GET("/features", getAuthFeatures)

// Bootstrap flow (pre-auth account creation).
authbootstrap.Routes(authRoutes)

// Each auth method registers its own public sign-in routes and receives
// the same /self and /admin sub-groups for credential management.
// Adding a new method: implement libauth.AuthMethod and add it here.
self := authRoutes.Group("/self")
self.Use(middleware.RequireAuthenticated())

admin := authRoutes.Group("/admin")
admin.Use(middleware.RequireAuthenticated(), middleware.RequireRole(libauth.RoleAdmin))

// Passkey auth method.
authpasskeys.Routes(authRoutes)
authpasskeys.SelfRoutes(self)
authpasskeys.AdminRoutes(admin)
	// Assignment links (shared mechanism for all auth methods).
	authassignmentlinks.SelfRoutes(self)
	authassignmentlinks.AdminRoutes(admin)
// Email (magic-link) auth method.
authemails.RegisterRoutes(authRoutes)
authemails.LoginRoutes(authRoutes)
authemails.SelfRoutes(self)
authemails.AdminRoutes(admin)
}

func getSession(c *gin.Context) {
if middleware.AuthBypassed() {
c.JSON(http.StatusOK, gin.H{"authenticated": false})
return
}
userID, err := middleware.ParseSessionCookie(c)
if err != nil {
c.JSON(http.StatusOK, gin.H{"authenticated": false})
return
}
user, err := database.Connection.GetUserByID(c, userID)
if err != nil {
c.JSON(http.StatusOK, gin.H{"authenticated": false})
return
}
roles, _ := database.Connection.ListUserRoles(c, userID)
c.JSON(http.StatusOK, gin.H{
"authenticated": true,
"user":          gin.H{"ID": user.ID, "Name": user.Name},
"roles":         roles,
})
}

func logout(c *gin.Context) {
c.SetCookie(middleware.SessionCookieName, "", -1, "/", "", false, true)
c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func getAuthFeatures(c *gin.Context) {
c.JSON(http.StatusOK, gin.H{
"magic_links_enabled": libauth.MagicLinkEnabled(),
})
}
