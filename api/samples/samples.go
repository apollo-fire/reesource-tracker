package samples

import (
	"database/sql"
	"math"
	"net/http"
	"reesource-tracker/api/samples/mods"
	"reesource-tracker/api/sync"
	"reesource-tracker/lib/database"
	id_helper "reesource-tracker/lib/id_helper"
	sampleid "reesource-tracker/lib/sample_id"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type SampleData struct {
	database.ListSamplesRow
	Mods []database.SampleMod `json:"mods"`
}

const MAX_PROVISIONED_SAMPLES = 1000

func Routes(route *gin.RouterGroup) {
	route.GET("/samples", getSamples)
	route.GET("/sample/:sample_id", getSample)
	route.POST("/sample/:sample_id", updateSample)
	route.GET("/generate_samples", generateUniqueSamples)
	mods.Routes(route.Group("/sample/:sample_id/mods"))
}

func getSample(c *gin.Context) {
	sampleID := c.Param("sample_id")
	if sampleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sample ID is required"})
		return
	}
	// Expecting sampleID in the format "xx-xx-xx" (6 base36 chars, 3 pairs)
	parts := strings.Split(sampleID, "-")
	if len(parts) != 3 || len(parts[0]) != 2 || len(parts[1]) != 2 || len(parts[2]) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sample ID format"})
		return
	}
	// Convert base36 pairs to bytes
	var rawID [4]byte
	for i, part := range parts {
		val, err := strconv.ParseUint(part, 36, 8)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sample ID format"})
			return
		}
		rawID[i] = byte(val)
	}
	RawSampleID := rawID[:]
	res, err := database.Connection.GetSampleById(c, RawSampleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	mod_data, err := database.Connection.ListSampleMods(c, RawSampleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sample": res, "mods": mod_data})
}

func updateSample(c *gin.Context) {
	// Update a specific sample by ID, or create a new sample with the provided uuid if it doesn't exist
	// These IDs will be provided by a QR code that opens the sample page
	sampleID := c.Param("sample_id")
	if sampleID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sample ID is required"})
		return
	}
	// Expecting sampleID in the format "xx-xx-xx" (6 base36 chars, 3 pairs)
	parts := strings.Split(strings.ToUpper(sampleID), "-")
	if len(parts) != 3 || len(parts[0]) != 2 || len(parts[1]) != 2 || len(parts[2]) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sample ID format"})
		return
	}
	// Convert base36 pairs to bytes
	var rawID [4]byte
	for i, part := range parts {
		val, err := strconv.ParseUint(part, 36, 8)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid sample ID format"})
			return
		}
		rawID[i] = byte(val)
	}
	RawSampleID := rawID[:]

	// Marshal location_id and product_id to binary ([]byte)
	locationID := c.PostForm("location_id")
	locationBinary, locErrMsg, locOK := id_helper.MustParseAndMarshalUUID(locationID)
	if !locOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": locErrMsg})
		return
	}

	productID := c.PostForm("product_id")
	productBinary, prodErrMsg, prodOK := id_helper.MustParseAndMarshalUUID(productID)
	if !prodOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": prodErrMsg})
		return
	}

	ownerID := c.PostForm("owner_id")
	ownerBinary, ownerErrMsg, ownerOK := id_helper.MustParseAndMarshalUUID(ownerID)
	if !ownerOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": ownerErrMsg})
		return
	}

	productIssue := c.PostForm("product_issue")

	current_time := time.Now()

	res, err := database.Connection.UpdateOrCreateSample(c, database.UpdateOrCreateSampleParams{
		ID:             RawSampleID,
		LocationID:     locationBinary,
		ProductID:      productBinary,
		OwnerID:        ownerBinary,
		ProductIssue:   sql.NullString{String: productIssue, Valid: true},
		TimeRegistered: sql.NullTime{Time: current_time, Valid: true},
		LastUpdate:     sql.NullTime{Time: current_time, Valid: true},
		State:          c.PostForm("state"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	sync.BroadcastEvent("samples_updated", gin.H{})
	c.JSON(http.StatusOK, res)
}

func getSamples(c *gin.Context) {
	res, err := database.Connection.ListSamples(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var samples []SampleData = []SampleData{}
	for _, sample := range res {
		// Convert time_registered to a string
		mod_data, err := database.Connection.ListSampleMods(c, sample.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		samples = append(samples, SampleData{sample, mod_data})
	}
	c.JSON(http.StatusOK, samples)
}

func generateUniqueSamples(c *gin.Context) {
	numSamplesStr := c.Query("num_samples")
	numSamples, err := strconv.Atoi(numSamplesStr)
	if err != nil || numSamples <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid number of samples"})
		return
	}
	numSamples = int(math.Min(float64(numSamples), float64(MAX_PROVISIONED_SAMPLES)))
	sample_ids := make([]string, numSamples)
	for i := 0; i < numSamples; i++ {
		new_id_string, new_id, err := sampleid.GenerateNewSampleID()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		_, err = database.Connection.UpdateOrCreateSample(c, database.UpdateOrCreateSampleParams{
			ID:    new_id[:],
			State: "unassigned",
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		sample_ids[i] = new_id_string
	}
	c.JSON(http.StatusOK, gin.H{"message": "Samples generated successfully", "sample_ids": sample_ids})
	sync.BroadcastEvent("samples_updated", gin.H{})
}
