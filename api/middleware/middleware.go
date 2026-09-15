package middleware

import (
	"database/sql"
	"net/http"
	"os"
	"time"

	libauth "reesource-tracker/lib/auth"
	liboidc "reesource-tracker/lib/oidc"

	"github.com/gin-gonic/gin"
)

const (
	sessionCookieName = "auth_session"
)

// AuthBypassed returns true in test mode or when AUTH_DISABLED=1, allowing
// all auth checks to pass without a real session.
func AuthBypassed() bool {
	return gin.Mode() == gin.TestMode || os.Getenv("AUTH_DISABLED") == "1"
}

func setSessionCookie(c *gin.Context, id string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     sessionCookieName,
		Value:    id,
		Path:     "/",
		MaxAge:   int(libauth.SessionDuration.Seconds()),
		HttpOnly: true,
		Secure:   isSecure(c),
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearSessionCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecure(c),
		SameSite: http.SameSiteLaxMode,
	})
}

func SetSessionCookie(c *gin.Context, sessionID string) {
	setSessionCookie(c, sessionID)
}

// RequireAuthenticated is a Gin middleware that validates the session cookie
// and hydrates auth_user_id + auth_roles into the context. If the access token
// has expired it transparently refreshes it; if the refresh fails the session
// is deleted and the request is rejected with 401.
func RequireAuthenticated() gin.HandlerFunc {
	return func(c *gin.Context) {
		if AuthBypassed() {
			c.Next()
			return
		}
		if !hydrateSession(c) {
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireRole is a Gin middleware that additionally requires the caller to hold
// the given role (or admin, which satisfies any role).
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if AuthBypassed() {
			c.Next()
			return
		}
		roles, ok := currentRoles(c)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			c.Abort()
			return
		}
		if !hasRequiredRole(roles, role) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// EnsureAuthenticated performs an inline auth check inside a handler.
// Returns true if authenticated (or bypassed), false + writes 401 otherwise.
func EnsureAuthenticated(c *gin.Context) bool {
	if AuthBypassed() {
		return true
	}
	if _, ok := c.Get("auth_user_id"); ok {
		return true
	}
	if !hydrateSession(c) {
		return false
	}
	return true
}

// EnsureRole performs an inline role check inside a handler.
func EnsureRole(c *gin.Context, role string) bool {
	if AuthBypassed() {
		return true
	}
	if !EnsureAuthenticated(c) {
		return false
	}
	roles, _ := currentRoles(c)
	if !hasRequiredRole(roles, role) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		c.Abort()
		return false
	}
	return true
}

func CurrentUserID(c *gin.Context) ([]byte, bool) {
	v, ok := c.Get("auth_user_id")
	if !ok {
		return nil, false
	}
	b, ok := v.([]byte)
	return b, ok
}

func CurrentUserRoles(c *gin.Context) []string {
	roles, _ := currentRoles(c)
	return roles
}

// hydrateSession reads the session cookie, looks it up in the DB, optionally
// refreshes the access token, and stores user ID + roles in the Gin context.
// Returns false and writes the HTTP error response on failure.
func hydrateSession(c *gin.Context) bool {
	sessionID, err := c.Cookie(sessionCookieName)
	if err != nil || sessionID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return false
	}

	session, err := libauth.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			ClearSessionCookie(c)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "session expired"})
			return false
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "session lookup failed"})
		return false
	}

	// Transparently refresh the access token when it has expired.
	if time.Now().After(session.AccessTokenExpiresAt) {
		oidcClient := liboidc.Get()
		if oidcClient == nil {
			// OIDC not initialised; treat as unauthenticated.
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return false
		}
		newTokens, refreshErr := oidcClient.Refresh(c.Request.Context(), session.RefreshToken)
		if refreshErr != nil {
			// Refresh token expired or revoked — clean up and force re-login.
			_ = libauth.DeleteSession(c.Request.Context(), sessionID)
			ClearSessionCookie(c)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "session expired"})
			return false
		}
		_ = libauth.UpdateSessionTokens(c.Request.Context(), sessionID,
			newTokens.RefreshToken, newTokens.AccessTokenExpiry)
	}

	c.Set("auth_user_id", session.UserID)
	c.Set("auth_roles", session.Roles)
	return true
}

func currentRoles(c *gin.Context) ([]string, bool) {
	v, ok := c.Get("auth_roles")
	if !ok {
		return nil, false
	}
	roles, ok := v.([]string)
	return roles, ok
}

// hasRequiredRole implements role hierarchy: admin satisfies any role requirement.
func hasRequiredRole(roles []string, required string) bool {
	if libauth.HasRole(roles, libauth.RoleAdmin) {
		return true
	}
	return libauth.HasRole(roles, required)
}

func isSecure(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	for _, proto := range splitHeader(c.GetHeader("X-Forwarded-Proto")) {
		if proto == "https" {
			return true
		}
	}
	return false
}

func splitHeader(h string) []string {
	if h == "" {
		return nil
	}
	var parts []string
	start := 0
	for i := 0; i < len(h); i++ {
		if h[i] == ',' {
			parts = append(parts, trimSpace(h[start:i]))
			start = i + 1
		}
	}
	parts = append(parts, trimSpace(h[start:]))
	return parts
}

func trimSpace(s string) string {
	for len(s) > 0 && s[0] == ' ' {
		s = s[1:]
	}
	for len(s) > 0 && s[len(s)-1] == ' ' {
		s = s[:len(s)-1]
	}
	return s
}
