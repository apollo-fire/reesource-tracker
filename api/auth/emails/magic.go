package emails

// Magic link sign-in is the authentication flow for the email auth method.
// A time-limited single-use token is generated and dispatched to the user's
// registered email address via the configured webhook; the user clicks the link
// which calls /auth/email/login/consume to establish a session.

import (
	"net/http"
	"net/url"
	"strings"

	"reesource-tracker/api/middleware"
	libauth "reesource-tracker/lib/auth"

	"github.com/gin-gonic/gin"
)

// LoginRoutes registers the unauthenticated magic-link sign-in endpoints.
func LoginRoutes(authRoutes *gin.RouterGroup) {
	authRoutes.POST("/email/login/request", requestMagicLink)
	authRoutes.POST("/email/login/consume", consumeMagicLink)
}

func requestMagicLink(c *gin.Context) {
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
	result, err := libauth.PrepareMagicLink(c, strings.TrimSpace(req.Email))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Always 200 – avoids leaking whether the email is registered.
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
	if result != nil {
		loginLink := buildMagicLinkURL(c, result.Token)
		go libauth.SendMagicLinkNotification(result.Email, result.UserName, loginLink)
	}
}

func consumeMagicLink(c *gin.Context) {
	var req struct {
		Token string `json:"token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token is required"})
		return
	}
	userID, err := libauth.ConsumeMagicLink(c, req.Token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := middleware.SetSessionCookie(c, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func buildMagicLinkURL(c *gin.Context, token string) string {
	scheme := c.GetHeader("X-Forwarded-Proto")
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	return scheme + "://" + host + "/app?magic_token=" + url.QueryEscape(token)
}
