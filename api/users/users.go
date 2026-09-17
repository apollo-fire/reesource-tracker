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
	ID   []byte `json:"ID"`
	Name string `json:"Name"`
}

func Routes(route *gin.RouterGroup) {
	route.GET("/users", getUsers)
	route.GET("/user/:user_id", getUser)
	route.POST("/user/:user_id", updateUser)
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
	c.JSON(http.StatusOK, UserResponse{ID: user.ID, Name: user.Name})
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
		responses = append(responses, UserResponse{ID: user.ID, Name: user.Name})
	}
	c.JSON(http.StatusOK, responses)
}
