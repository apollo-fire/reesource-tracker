package passkeys

import (
	"io"
	"net/http"

	"reesource-tracker/api/middleware"
	libauth "reesource-tracker/lib/auth"

	"github.com/gin-gonic/gin"
)

// Routes registers the unauthenticated passkey registration and login endpoints.
func Routes(authRoutes *gin.RouterGroup) {
	authRoutes.POST("/register/begin", beginRegistration)
	authRoutes.POST("/register/finish", finishRegistration)
	authRoutes.POST("/login/begin", beginLogin)
	authRoutes.POST("/login/finish", finishLogin)
}

func beginRegistration(c *gin.Context) {
	var req struct {
		AssignmentToken string `json:"assignment_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := libauth.BeginRegistration(c, req.AssignmentToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"challenge_token": res.ChallengeToken,
		"challenge":       res.Challenge,
		"user_id":         res.UserID,
		"user_name":       res.UserName,
	})
}

func finishRegistration(c *gin.Context) {
	var req struct {
		ChallengeToken string   `json:"challenge_token"`
		Challenge      string   `json:"challenge"`
		CredentialID   string   `json:"credential_id"`
		PublicKey      string   `json:"public_key"`
		Label          string   `json:"label"`
		Transports     []string `json:"transports"`
		ClientDataJSON string   `json:"client_data_json"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	credentialID, err := libauth.DecodeBase64(req.CredentialID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credential_id"})
		return
	}
	userID, err := libauth.FinishRegistration(c, libauth.FinishRegistrationInput{
		ChallengeToken: req.ChallengeToken,
		Challenge:      req.Challenge,
		CredentialID:   req.CredentialID,
		PublicKey:      req.PublicKey,
		Label:          req.Label,
		Transports:     req.Transports,
		ClientDataJSON: req.ClientDataJSON,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := middleware.SetSessionCookieWithCredential(c, userID, credentialID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func beginLogin(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil && err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := libauth.BeginLogin(c, req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"challenge_token": res.ChallengeToken, "challenge": res.Challenge})
}

func finishLogin(c *gin.Context) {
	var req struct {
		ChallengeToken    string `json:"challenge_token"`
		Challenge         string `json:"challenge"`
		CredentialID      string `json:"credential_id"`
		SignCounter       int64  `json:"sign_counter"`
		ClientDataJSON    string `json:"client_data_json"`
		AuthenticatorData string `json:"authenticator_data"`
		Signature         string `json:"signature"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	credentialID, err := libauth.DecodeBase64(req.CredentialID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credential_id"})
		return
	}
	userID, err := libauth.FinishLogin(c, libauth.FinishLoginInput{
		ChallengeToken:    req.ChallengeToken,
		Challenge:         req.Challenge,
		CredentialID:      req.CredentialID,
		SignCounter:       req.SignCounter,
		ClientDataJSON:    req.ClientDataJSON,
		AuthenticatorData: req.AuthenticatorData,
		Signature:         req.Signature,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := middleware.SetSessionCookieWithCredential(c, userID, credentialID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
