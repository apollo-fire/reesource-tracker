package locations

import (
	"database/sql"
	"net/http"
	"reesource-tracker/api/sync"
	"reesource-tracker/lib/database"
	id_helper "reesource-tracker/lib/id_helper"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func Routes(route *gin.RouterGroup) {
	route.GET("/locations", getLocations)
	route.POST("/location", createLocation)
	route.GET("/location/:location_id", getLocation)
	route.POST("/location/:location_id", updateLocation)
	route.DELETE("/location/:location_id", deleteLocation)
}

func deleteLocation(c *gin.Context) {
	locationID := c.Param("location_id")
	if locationID == "" {
		c.JSON(400, gin.H{"error": "id required"})
		return
	}
	binary_uuid, errMsg, ok := id_helper.MustParseAndMarshalUUID(locationID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}
	err := database.Connection.DeleteLocationByID(c, binary_uuid)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "deleted"})
	sync.BroadcastEvent("locations_updated", gin.H{})
}

func createLocation(c *gin.Context) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	new_uid, err := uuid.New().MarshalBinary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate product ID"})
		return
	}
	params := database.UpsertLocationParams{
		ID:          new_uid,
		Name:        req.Name,
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
	}
	err = database.Connection.UpsertLocation(c, params)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "success"})
	sync.BroadcastEvent("locations_updated", gin.H{})
}

func getLocations(c *gin.Context) {
	res, err := database.Connection.GetLocations(c)
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.JSON(200, res)
}

func getLocation(c *gin.Context) {
	locationID := c.Param("location_id")
	if locationID == "" {
		c.JSON(400, gin.H{"error": "id required"})
		return
	}
	location, err := database.Connection.GetLocation(c, locationID)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, location)
}

func updateLocation(c *gin.Context) {
	var req struct {
		Name             string `json:"name"`
		Description      string `json:"description"`
		ParentLocationID string `json:"parent_location_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	locationID := c.Param("location_id")
	if locationID == "" {
		c.JSON(400, gin.H{"error": "id required"})
		return
	}
	binary_uuid, errMsg, ok := id_helper.MustParseAndMarshalUUID(locationID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}

	var parentBinaryUUID []byte
	if req.ParentLocationID != "" {
		parentBinaryUUID, errMsg, ok = id_helper.MustParseAndMarshalUUID(req.ParentLocationID)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
			return
		}
	}

	// UpsertLocation expects UpsertLocationParams struct
	params := database.UpsertLocationParams{
		ID:               binary_uuid,
		Name:             req.Name,
		Description:      sql.NullString{String: req.Description, Valid: req.Description != ""},
		ParentLocationID: parentBinaryUUID,
	}
	err := database.Connection.UpsertLocation(c, params)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"status": "success"})
	sync.BroadcastEvent("locations_updated", gin.H{})
}
