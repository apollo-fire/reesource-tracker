package middleware

import (
	"net/http"
	"os"
	"strings"

	libauth "reesource-tracker/lib/auth"
	"reesource-tracker/lib/database"

	"github.com/gin-gonic/gin"
)

const (
	SessionCookieName = "auth_session"
	sessionDuration   = 24 * 60 * 60 // seconds
)

var cookieSecret []byte

// SetCookieSecret must be called once during application startup before any
// request is served.
func SetCookieSecret(secret []byte) {
	cookieSecret = secret
}

func isSecureRequest(c *gin.Context) bool {
	if c.Request != nil && c.Request.TLS != nil {
		return true
	}

	xForwardedProto := c.GetHeader("X-Forwarded-Proto")
	for _, proto := range strings.Split(xForwardedProto, ",") {
		if strings.EqualFold(strings.TrimSpace(proto), "https") {
			return true
		}
	}

	return false
}

func setSessionCookieValue(c *gin.Context, token string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   sessionDuration,
		HttpOnly: true,
		Secure:   isSecureRequest(c),
		SameSite: http.SameSiteLaxMode,
	})
}

// SetSessionCookie signs and sets the auth cookie on the response.
func SetSessionCookie(c *gin.Context, userID []byte) error {
	token, err := libauth.BuildSessionToken(cookieSecret, userID, libauth.SessionDuration)
	if err != nil {
		return err
	}
	setSessionCookieValue(c, token)
	return nil
}

// SetSessionCookieWithCredential signs and sets the auth cookie including the
// credential ID used for the current session.
func SetSessionCookieWithCredential(c *gin.Context, userID []byte, credentialID []byte) error {
	token, err := libauth.BuildSessionTokenWithCredential(cookieSecret, userID, credentialID, libauth.SessionDuration)
	if err != nil {
		return err
	}
	setSessionCookieValue(c, token)
	return nil
}

// ParseSessionCookie validates and parses the auth cookie, returning the user ID.
func ParseSessionCookie(c *gin.Context) ([]byte, error) {
	token, err := c.Cookie(SessionCookieName)
	if err != nil {
		return nil, err
	}
	return libauth.ParseSessionToken(cookieSecret, token)
}

// ParseSessionCookieWithCredential validates and parses the auth cookie,
// returning both user ID and current session credential ID (if present).
func ParseSessionCookieWithCredential(c *gin.Context) ([]byte, []byte, error) {
	token, err := c.Cookie(SessionCookieName)
	if err != nil {
		return nil, nil, err
	}
	return libauth.ParseSessionTokenWithCredential(cookieSecret, token)
}

// CurrentUserID returns the resolved user ID from context if already hydrated.
func CurrentUserID(c *gin.Context) ([]byte, bool) {
	v, ok := c.Get("auth_user_id")
	if !ok {
		return nil, false
	}
	b, ok := v.([]byte)
	return b, ok
}

// CurrentCredentialID returns the credential ID used by the current session if
// available.
func CurrentCredentialID(c *gin.Context) ([]byte, bool) {
	v, ok := c.Get("auth_credential_id")
	if !ok {
		return nil, false
	}
	b, ok := v.([]byte)
	return b, ok
}

// AuthBypassed returns true when the server is running in test mode or auth
// has been explicitly disabled via environment variable.
func AuthBypassed() bool {
	return gin.Mode() == gin.TestMode || os.Getenv("AUTH_DISABLED") == "1"
}

// EnsureAuthenticated checks authentication inline in a handler (not as Gin
// middleware). Returns false and writes the error response when not authenticated.
func EnsureAuthenticated(c *gin.Context) bool {
	if AuthBypassed() {
		return true
	}
	if _, ok := CurrentUserID(c); ok {
		return true
	}
	if err := hydrateSessionIntoContext(c); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		c.Abort()
		return false
	}
	return true
}

// EnsureRole checks authentication and role inline in a handler.
func EnsureRole(c *gin.Context, role string) bool {
	if AuthBypassed() {
		return true
	}
	if !EnsureAuthenticated(c) {
		return false
	}
	userID, _ := CurrentUserID(c)
	roles, err := database.Connection.ListUserRoles(c, userID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "role lookup failed"})
		c.Abort()
		return false
	}
	if !hasRequiredRole(roles, role) {
		c.JSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		c.Abort()
		return false
	}
	return true
}

// EnsureConfirmed checks for an explicit confirmation signal inline in a handler.
func EnsureConfirmed(c *gin.Context) bool {
	if AuthBypassed() {
		return true
	}
	if isConfirmed(c) {
		return true
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "explicit confirmation required"})
	c.Abort()
	return false
}

// RequireAuthenticated is a Gin middleware factory.
func RequireAuthenticated() gin.HandlerFunc {
	return func(c *gin.Context) {
		if AuthBypassed() {
			c.Next()
			return
		}
		if err := hydrateSessionIntoContext(c); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		c.Next()
	}
}

// RequireRole is a Gin middleware factory that enforces a minimum role.
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if AuthBypassed() {
			c.Next()
			return
		}
		if err := hydrateSessionIntoContext(c); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		userID, _ := CurrentUserID(c)
		roles, err := database.Connection.ListUserRoles(c, userID)
		if err != nil || !hasRequiredRole(roles, role) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
			return
		}
		c.Next()
	}
}

// RequireConfirmedAction is a Gin middleware factory that requires an explicit
// confirmation signal (header, query param, or form field).
func RequireConfirmedAction() gin.HandlerFunc {
	return func(c *gin.Context) {
		if AuthBypassed() {
			c.Next()
			return
		}
		if !isConfirmed(c) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "explicit confirmation required"})
			return
		}
		c.Next()
	}
}

// hydrateSessionIntoContext reads and validates the session cookie, then stores
// the user ID in the Gin context for subsequent handlers.
func hydrateSessionIntoContext(c *gin.Context) error {
	if _, exists := c.Get("auth_user_id"); exists {
		return nil
	}
	userID, credentialID, err := ParseSessionCookieWithCredential(c)
	if err != nil {
		return err
	}
	c.Set("auth_user_id", userID)
	if len(credentialID) > 0 {
		c.Set("auth_credential_id", credentialID)
	}
	return nil
}

func hasRequiredRole(roles []string, required string) bool {
	if required == libauth.RoleUser {
		return true
	}
	switch required {
	case libauth.RoleAdmin:
		return libauth.HasRole(roles, libauth.RoleAdmin)
	case libauth.RoleMaintainer:
		return libauth.HasRole(roles, libauth.RoleAdmin) || libauth.HasRole(roles, libauth.RoleMaintainer)
	}
	return false
}

func isConfirmed(c *gin.Context) bool {
	normalize := func(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
	confirmed := func(s string) bool { return s == "true" || s == "yes" || s == "confirm" }
	return confirmed(normalize(c.GetHeader("X-Confirm-Action"))) ||
		confirmed(normalize(c.Query("confirm_action"))) ||
		confirmed(normalize(c.PostForm("confirm_action")))
}
