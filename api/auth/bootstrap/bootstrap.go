package bootstrap

import (
	"net/http"

	libauth "reesource-tracker/lib/auth"
	id_helper "reesource-tracker/lib/id_helper"

	"github.com/gin-gonic/gin"
)

func Routes(authRoutes *gin.RouterGroup) {
	authRoutes.GET("/bootstrap-status", getBootstrapStatus)
	authRoutes.GET("/bootstrap-options", getBootstrapOptions)
	authRoutes.POST("/bootstrap/select-user", bootstrapSelectUser)
	authRoutes.POST("/bootstrap/create-user", bootstrapCreateUser)
}

func getBootstrapStatus(c *gin.Context) {
	required, token, userID, err := libauth.EnsureBootstrapState(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"bootstrap_required": required,
		"assignment_token":   token,
		"user_id":            userID,
		"assignment_url":     "/app?assignment_token=" + token,
	})
}

func getBootstrapOptions(c *gin.Context) {
	required, token, userID, err := libauth.EnsureBootstrapState(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	users, err := libauth.ListBootstrapUserOptions(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"bootstrap_required": required,
		"assignment_token":   token,
		"user_id":            userID,
		"users":              users,
	})
}

func bootstrapSelectUser(c *gin.Context) {
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.UserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}
	required, _, _, err := libauth.EnsureBootstrapState(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !required {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bootstrap flow is not active"})
		return
	}
	uid, msg, ok := id_helper.MustParseAndMarshalUUID(req.UserID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	token, selectedUserID, err := libauth.SelectBootstrapUser(c, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"assignment_token": token, "user_id": selectedUserID})
}

func bootstrapCreateUser(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	_ = c.ShouldBindJSON(&req)
	required, _, _, err := libauth.EnsureBootstrapState(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !required {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bootstrap flow is not active"})
		return
	}
	token, selectedUserID, err := libauth.CreateBootstrapUserAndSelect(c, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"assignment_token": token, "user_id": selectedUserID})
}
