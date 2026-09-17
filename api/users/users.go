package users

import (
	"database/sql"
	"net/http"
	"reesource-tracker/api/sync"
	"reesource-tracker/lib/database"
	id_helper "reesource-tracker/lib/id_helper"

	"github.com/gin-gonic/gin"
)

type UserResponse struct {
	ID      []byte `json:"ID"`
	Name    string `json:"Name"`
	HasOIDC bool   `json:"HasOIDC"`
}

func Routes(route *gin.RouterGroup) {
	route.GET("/users", getUsers)
	route.GET("/user/:user_id", getUser)
	route.POST("/user/:user_id", updateUser)
	route.POST("/user/:user_id/merge", mergeUser)
	route.DELETE("/user/:user_id", deleteUser)
}

// DELETE /user/:user_id
func deleteUser(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
		return
	}
	binary_uuid, errMsg, ok := id_helper.MustParseAndMarshalUUID(userID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}
	err := database.Connection.DeleteUserByID(c, binary_uuid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	sync.BroadcastEvent("users_updated", gin.H{})
}

func getUser(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
		return
	}
	userIDBytes, errMsg, ok := id_helper.MustParseAndMarshalUUID(userID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}
	user, err := database.Connection.GetUserByID(c, userIDBytes)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, UserResponse{ID: user.ID, Name: user.Name, HasOIDC: user.OidcSub.Valid})
}

// POST /user/:user_id/merge — reassigns all data owned by the legacy (non-OIDC)
// user at :user_id to the OIDC-linked user given in the request body, then
// deletes the legacy user.
func mergeUser(c *gin.Context) {
	legacyUserID := c.Param("user_id")
	if legacyUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
		return
	}
	var req struct {
		TargetUserID string `json:"target_user_id"`
	}
	if err := c.ShouldBind(&req); err != nil || req.TargetUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_user_id required"})
		return
	}

	legacyID, errMsg, ok := id_helper.MustParseAndMarshalUUID(legacyUserID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}
	targetID, errMsg, ok := id_helper.MustParseAndMarshalUUID(req.TargetUserID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}

	legacyUser, err := database.Connection.GetUserByID(c, legacyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "legacy user not found"})
		return
	}
	if legacyUser.OidcSub.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user already has an OIDC account linked"})
		return
	}

	targetUser, err := database.Connection.GetUserByID(c, targetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "target user not found"})
		return
	}
	if !targetUser.OidcSub.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target user has no OIDC account linked"})
		return
	}

	err = database.Connection.MergeUsers(c, database.MergeUsersParams{
		TargetID: targetUser.ID,
		LegacyID: legacyUser.ID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "merged"})
	sync.BroadcastEvent("users_updated", gin.H{})
	sync.BroadcastEvent("samples_updated", gin.H{})
}

func updateUser(c *gin.Context) {
	userID := c.Param("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	binary_uuid, errMsg, ok := id_helper.MustParseAndMarshalUUID(userID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}
	rows, err := database.Connection.UpdateUser(c, database.UpdateUserParams{
		ID:   binary_uuid,
		Name: req.Name,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": sql.ErrNoRows.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
	sync.BroadcastEvent("users_updated", gin.H{})
}

func getUsers(c *gin.Context) {
	res, err := database.Connection.GetUsers(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var responses []UserResponse
	for _, user := range res {
		responses = append(responses, UserResponse{ID: user.ID, Name: user.Name, HasOIDC: user.OidcSub.Valid})
	}
	c.JSON(http.StatusOK, responses)
}
