package users

import (
	"bytes"
	"net/http"
	"reesource-tracker/api/middleware"
	"reesource-tracker/api/sync"
	libauth "reesource-tracker/lib/auth"
	"reesource-tracker/lib/database"
	id_helper "reesource-tracker/lib/id_helper"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserResponse struct {
	ID    []byte   `json:"ID"`
	Name  string   `json:"Name"`
	Roles []string `json:"Roles"`
}

func Routes(route *gin.RouterGroup) {
	route.GET("/users", getUsers)
	route.POST("/user", createUser)
	route.GET("/user/:user_id", getUser)
	route.POST("/user/:user_id", updateUser)
	route.DELETE("/user/:user_id", deleteUser)
	route.POST("/user/:user_id/roles", setRole)
	route.DELETE("/user/:user_id/roles/:role", removeRole)
}

// DELETE /user/:user_id
func deleteUser(c *gin.Context) {
	if !middleware.EnsureRole(c, libauth.RoleAdmin) || !middleware.EnsureConfirmed(c) {
		return
	}
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
	actor, _ := middleware.CurrentUserID(c)
	_ = libauth.AuditLog(c, &actor, "user_deleted", "user", userID, gin.H{})
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	sync.BroadcastEvent("users_updated", gin.H{})
}

func createUser(c *gin.Context) {
	if !middleware.EnsureRole(c, libauth.RoleAdmin) {
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	new_uid, err := uuid.New().MarshalBinary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate user ID"})
		return
	}
	params := database.UpsertUserParams{
		ID:   new_uid,
		Name: req.Name,
	}
	err = database.Connection.UpsertUser(c, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err = database.Connection.SetUserRole(c, database.SetUserRoleParams{UserID: new_uid, Role: libauth.RoleUser}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	actor, _ := middleware.CurrentUserID(c)
	link, linkErr := libauth.CreateStandardAssignmentLinkForUser(c, new_uid, actor)
	if linkErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": linkErr.Error()})
		return
	}
	_ = libauth.AuditLog(c, &actor, "user_created", "user", userIDString(new_uid), gin.H{"name": req.Name})
	c.JSON(http.StatusOK, gin.H{"status": "success", "assignment_token": link.TokenHash, "expires_at": link.ExpiresAt.Time, "link_id": link.ID})
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
	response := UserResponse{
		ID:   user.ID,
		Name: user.Name,
	}
	response.Roles, _ = database.Connection.ListUserRoles(c, user.ID)
	c.JSON(http.StatusOK, response)
}

func updateUser(c *gin.Context) {
	if !middleware.EnsureRole(c, libauth.RoleAdmin) {
		return
	}
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
	params := database.UpsertUserParams{
		ID:   binary_uuid,
		Name: req.Name,
	}
	err := database.Connection.UpsertUser(c, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	actor, _ := middleware.CurrentUserID(c)
	_ = libauth.AuditLog(c, &actor, "user_updated", "user", userID, gin.H{"name": req.Name})
	c.JSON(http.StatusOK, gin.H{"status": "success"})
	sync.BroadcastEvent("users_updated", gin.H{})
}

func getUsers(c *gin.Context) {
	res, err := database.Connection.GetUsersWithRoles(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	responses := make([]UserResponse, 0, len(res))
	for _, user := range res {
		responses = append(responses, UserResponse{
			ID:    user.ID,
			Name:  user.Name,
			Roles: user.Roles,
		})
	}
	c.JSON(http.StatusOK, responses)
}

func setRole(c *gin.Context) {
	if !middleware.EnsureRole(c, libauth.RoleAdmin) || !middleware.EnsureConfirmed(c) {
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !libauth.IsValidRole(req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
		return
	}

	targetID, msg, ok := id_helper.MustParseAndMarshalUUID(c.Param("user_id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	if err := database.Connection.SetUserRole(c, database.SetUserRoleParams{UserID: targetID, Role: req.Role}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	actor, _ := middleware.CurrentUserID(c)
	_ = libauth.AuditLog(c, &actor, "role_set", "user", c.Param("user_id"), gin.H{"role": req.Role})
	sync.BroadcastEvent("users_updated", gin.H{})
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func removeRole(c *gin.Context) {
	if !middleware.EnsureRole(c, libauth.RoleAdmin) || !middleware.EnsureConfirmed(c) {
		return
	}

	targetID, msg, ok := id_helper.MustParseAndMarshalUUID(c.Param("user_id"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	role := c.Param("role")
	if !libauth.IsValidRole(role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
		return
	}

	actor, _ := middleware.CurrentUserID(c)
	if role == libauth.RoleAdmin {
		adminCount, err := database.Connection.CountAdmins(c)
		if err == nil && adminCount <= 1 && bytes.Equal(actor, targetID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sole admin cannot self-demote"})
			return
		}
	}

	if err := database.Connection.RemoveUserRole(c, database.RemoveUserRoleParams{UserID: targetID, Role: role}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_ = libauth.AuditLog(c, &actor, "role_removed", "user", c.Param("user_id"), gin.H{"role": role})
	sync.BroadcastEvent("users_updated", gin.H{})
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func userIDString(raw []byte) string {
	s, err := id_helper.UnmarshalUUID(raw)
	if err != nil {
		return ""
	}
	return s
}

