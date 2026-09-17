package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"reesource-tracker/api/middleware"
	libauth "reesource-tracker/lib/auth"
	"reesource-tracker/lib/database"
	liboidc "reesource-tracker/lib/oidc"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const stateCookieName = "oidc_state"

var configuredAppBaseURL string

func Initialize(ctx context.Context, issuerURL, clientID, clientSecret, baseURL, roleClaimPath string, roleMap map[string]string) error {
	normalizedBaseURL, err := normalizeBaseURL(baseURL)
	if err != nil {
		return err
	}
	configuredAppBaseURL = normalizedBaseURL
	redirectURL := configuredAppBaseURL + "/api/auth/callback"
	return liboidc.Initialize(ctx, issuerURL, clientID, clientSecret, redirectURL, roleClaimPath, roleMap)
}

func Routes(route *gin.RouterGroup) {
	auth := route.Group("/auth")
	auth.GET("/login", login)
	auth.GET("/callback", callback)
	auth.GET("/logout", logout)
	auth.GET("/session", getSession)
	auth.POST("/backchannel-logout", backchannelLogout)
	auth.GET("/frontchannel-logout", frontchannelLogout)
}

// GET /api/auth/login — redirects the browser to the OIDC provider.
func login(c *gin.Context) {
	state, err := randomHex(16)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate state"})
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     stateCookieName,
		Value:    state,
		Path:     "/",
		MaxAge:   300, // 5 minutes
		HttpOnly: true,
		Secure:   isSecure(c),
		SameSite: http.SameSiteLaxMode,
	})
	c.Redirect(http.StatusFound, liboidc.Get().AuthCodeURL(state))
}

// GET /api/auth/callback — handles the OIDC redirect back from the provider.
func callback(c *gin.Context) {
	// Verify state to prevent CSRF.
	stateCookie, err := c.Cookie(stateCookieName)
	if err != nil || stateCookie == "" || stateCookie != c.Query("state") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name: stateCookieName, Value: "", Path: "/", MaxAge: -1,
	})

	oidcClient := liboidc.Get()
	tokens, err := oidcClient.Exchange(c.Request.Context(), c.Query("code"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token exchange failed"})
		return
	}

	roles := extractRolesFromContext(c, tokens)

	// JIT provision: upsert user by OIDC subject.
	newID, _ := uuid.New().MarshalBinary()
	user, err := database.Connection.UpsertUserByOIDCSub(c.Request.Context(), database.UpsertUserByOIDCSubParams{
		ID:      newID,
		Name:    tokens.Name,
		OidcSub: sql.NullString{String: tokens.Subject, Valid: true},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user provisioning failed"})
		return
	}

	sessionID, err := libauth.NewSessionID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create session"})
		return
	}

	expiry := time.Now().Add(libauth.SessionDuration)
	err = libauth.CreateSession(c.Request.Context(), libauth.Session{
		ID:                   sessionID,
		UserID:               user.ID,
		Roles:                roles,
		OIDCSid:              tokens.OIDCSid,
		IDToken:              tokens.RawIDToken,
		RefreshToken:         tokens.RefreshToken,
		AccessTokenExpiresAt: tokens.AccessTokenExpiry,
		ExpiresAt:            expiry,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not persist session"})
		return
	}

	middleware.SetSessionCookie(c, sessionID)
	c.Redirect(http.StatusFound, "/app")
}

// POST /api/auth/logout — ends the local session and redirects to the OIDC
// end_session_endpoint so the provider session is also terminated.
func logout(c *gin.Context) {
	sessionID, _ := c.Cookie("auth_session")
	var idToken string
	if sessionID != "" {
		if s, err := libauth.GetSession(c.Request.Context(), sessionID); err == nil {
			idToken = s.IDToken
		}
		_ = libauth.DeleteSession(c.Request.Context(), sessionID)
	}
	middleware.ClearSessionCookie(c)
	postLogout := appBaseURL() + "/app"
	c.Redirect(http.StatusFound, liboidc.Get().EndSessionURL(idToken, postLogout))
}

// GET /api/auth/session — returns the authenticated user's identity and roles.
func getSession(c *gin.Context) {
	if !middleware.EnsureAuthenticated(c) {
		return
	}
	userID, _ := middleware.CurrentUserID(c)
	user, err := database.Connection.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user lookup failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":    user.ID,
		"name":  user.Name,
		"roles": middleware.CurrentUserRoles(c),
	})
}

// POST /api/auth/backchannel-logout — provider-to-server logout notification
// (signed JWT in logout_token form field, per OIDC Back-Channel Logout spec).
func backchannelLogout(c *gin.Context) {
	tokenString := c.PostForm("logout_token")
	if tokenString == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	sub, sid, err := liboidc.Get().VerifyLogoutToken(c.Request.Context(), tokenString)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	if sid != "" {
		_ = libauth.DeleteSessionByOIDCSID(c.Request.Context(), sid)
	} else if sub != "" {
		user, lookupErr := database.Connection.GetUserByOIDCSub(c.Request.Context(),
			sql.NullString{String: sub, Valid: true})
		if lookupErr == nil {
			_ = libauth.DeleteUserSessions(c.Request.Context(), user.ID)
		}
	}
	c.Status(http.StatusOK)
}

// GET /api/auth/frontchannel-logout — browser-iframe logout from the provider.
// Less reliable than back-channel but clears the browser cookie immediately.
func frontchannelLogout(c *gin.Context) {
	sid := c.Query("sid")
	if sid != "" {
		_ = libauth.DeleteSessionByOIDCSID(c.Request.Context(), sid)
	}
	middleware.ClearSessionCookie(c)
	c.Status(http.StatusOK)
}

// extractRolesFromContext reads roles from the token claims using the OIDC
// client's configured claim path and role map.
func extractRolesFromContext(c *gin.Context, tokens *liboidc.TokenSet) []string {
	client := liboidc.Get()
	if client == nil || tokens.RawClaims == nil {
		return nil
	}
	return liboidc.ExtractRoles(tokens.RawClaims, client.RoleClaimPath(), client.RoleMap())
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func isSecure(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}
	for _, proto := range c.Request.Header["X-Forwarded-Proto"] {
		if proto == "https" {
			return true
		}
	}
	return false
}

func appBaseURL() string {
	return configuredAppBaseURL
}

func normalizeBaseURL(baseURL string) (string, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base_url: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("invalid base_url: %q", baseURL)
	}
	return baseURL, nil
}
