package sample_mods

import (
	"database/sql"
	"net/http"
	"reesource-tracker/api/sync"
	"reesource-tracker/lib/database"
	"reesource-tracker/lib/id_helper"
	sampleid "reesource-tracker/lib/sample_id"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func Routes(route *gin.RouterGroup) {
	route.POST("/", addMod)
	route.DELETE("/:mod_id", removeMod)
	route.GET("/", listMods)
}

func addMod(c *gin.Context) {
	sampleID := c.Param("sample_id")
	if sampleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sample ID is required"})
		return
	}
	var req struct {
		Name string `json:"name" form:"name"`
	}
	if err := c.ShouldBind(&req); err != nil || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mod name is required"})
		return
	}
	parts, err := sampleid.ParseSampleID(sampleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sample ID format"})
		return
	}
	RawSampleID := parts[:]
	modID, err := uuid.New().MarshalBinary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate mod ID"})
		return
	}
	timeNow := time.Now()
	err = database.Connection.AddSampleMod(c, database.AddSampleModParams{
		ID:        modID,
		SampleID:  RawSampleID,
		Name:      req.Name,
		TimeAdded: timeNow,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	sync.BroadcastEvent("samples_updated", gin.H{})
	c.JSON(http.StatusOK, gin.H{"message": "Mod added"})
}

func removeMod(c *gin.Context) {
	modID := c.Param("mod_id")
	if modID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mod ID is required"})
		return
	}
	modUUID, msg, ok := id_helper.MustParseAndMarshalUUID(modID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	err := database.Connection.RemoveSampleMod(c, database.RemoveSampleModParams{
		TimeRemoved: sql.NullTime{Time: time.Now(), Valid: true},
		ID:          modUUID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	sync.BroadcastEvent("samples_updated", gin.H{})
	c.JSON(http.StatusOK, gin.H{"message": "Mod removed"})
}

func listMods(c *gin.Context) {
	sampleID := c.Param("sample_id")
	if sampleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sample ID is required"})
		return
	}
	parts, err := sampleid.ParseSampleID(sampleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sample ID format"})
		return
	}
	RawSampleID := parts[:]

	mods, err := database.Connection.ListSampleMods(c, RawSampleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"mods": mods})
}
