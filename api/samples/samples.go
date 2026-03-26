package samples

import (
	"database/sql"
	"math"
	"net/http"
	"reesource-tracker/api/samples/sample_mods"
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
	SampleResponse
	Mods []sample_mods.SampleModResponse `json:"mods"`
}

type SampleResponse struct {
	ID             []byte         `json:"ID"`
	LocationID     *[]byte        `json:"LocationID,omitempty"`
	ProductID      *[]byte        `json:"ProductID,omitempty"`
	TimeRegistered string         `json:"TimeRegistered"`
	LastUpdate     string         `json:"LastUpdate"`
	State          string         `json:"State"`
	OwnerID        *[]byte        `json:"OwnerID,omitempty"`
	ProductIssue   sql.NullString `json:"ProductIssue"`
}

const MAX_PROVISIONED_SAMPLES = 1000

func Routes(route *gin.RouterGroup) {
	route.GET("/samples", getSamples)
	route.GET("/sample/:sample_id", getSample)
	route.POST("/sample/:sample_id", updateSample)
	route.GET("/generate_samples", generateUniqueSamples)
	sample_mods.Routes(route.Group("/sample/:sample_id/mods"))
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

	sampleResponse := SampleResponse{
		ID:    res.ID,
		State: res.State,
	}
	if res.LocationID.Valid {
		sampleResponse.LocationID = &res.LocationID.V
	}
	if res.ProductID.Valid {
		sampleResponse.ProductID = &res.ProductID.V
	}
	if res.TimeRegistered.Valid {
		timeStr := res.TimeRegistered.Time.Format(time.RFC3339)
		sampleResponse.TimeRegistered = timeStr
	}
	if res.LastUpdate.Valid {
		timeStr := res.LastUpdate.Time.Format(time.RFC3339)
		sampleResponse.LastUpdate = timeStr
	}
	if res.OwnerID.Valid {
		sampleResponse.OwnerID = &res.OwnerID.V
	}
	sampleResponse.ProductIssue = res.ProductIssue

	c.JSON(http.StatusOK, gin.H{"sample": sampleResponse, "mods": mod_data})
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

	type updateSampleRequest struct {
		LocationID   string `form:"location_id"`
		ProductID    string `form:"product_id"`
		OwnerID      string `form:"owner_id"`
		ProductIssue string `form:"product_issue"`
		State        string `form:"state"`
	}
	var req updateSampleRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	locationBinary, locErrMsg, locOK := id_helper.MustParseAndMarshalUUID(req.LocationID)
	if !locOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": locErrMsg})
		return
	}
	productBinary, prodErrMsg, prodOK := id_helper.MustParseAndMarshalUUID(req.ProductID)
	if !prodOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": prodErrMsg})
		return
	}
	ownerBinary, ownerErrMsg, ownerOK := id_helper.MustParseAndMarshalUUID(req.OwnerID)
	if !ownerOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": ownerErrMsg})
		return
	}
	current_time := time.Now()
	res, err := database.Connection.UpdateOrCreateSample(c, database.UpdateOrCreateSampleParams{
		ID:             RawSampleID,
		LocationID:     sql.Null[[]byte]{V: locationBinary, Valid: locationBinary != nil},
		ProductID:      sql.Null[[]byte]{V: productBinary, Valid: productBinary != nil},
		OwnerID:        sql.Null[[]byte]{V: ownerBinary, Valid: ownerBinary != nil},
		ProductIssue:   sql.NullString{String: req.ProductIssue, Valid: req.ProductIssue != ""},
		TimeRegistered: sql.NullTime{Time: current_time, Valid: true},
		LastUpdate:     sql.NullTime{Time: current_time, Valid: true},
		State:          req.State,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	sync.BroadcastEvent("samples_updated", gin.H{})

	var locationID *[]byte
	if res.LocationID.Valid {
		v := res.LocationID.V
		locationID = &v
	}

	var productID *[]byte
	if res.ProductID.Valid {
		v := res.ProductID.V
		productID = &v
	}

	var ownerID *[]byte
	if res.OwnerID.Valid {
		v := res.OwnerID.V
		ownerID = &v
	}

	var timeRegistered string
	if res.TimeRegistered.Valid {
		timeRegistered = res.TimeRegistered.Time.Format(time.RFC3339)
	}

	var lastUpdate string
	if res.LastUpdate.Valid {
		lastUpdate = res.LastUpdate.Time.Format(time.RFC3339)
	}

	c.JSON(http.StatusOK, SampleResponse{
		LocationID:     locationID,
		ProductID:      productID,
		OwnerID:        ownerID,
		ID:             res.ID,
		TimeRegistered: timeRegistered,
		LastUpdate:     lastUpdate,
		State:          res.State,
		ProductIssue:   res.ProductIssue,
	})
}

func getSamples(c *gin.Context) {
	res, err := database.Connection.ListSamples(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var samples []SampleData = []SampleData{}
	for _, sample := range res {
		mod_data, err := database.Connection.ListSampleMods(c, sample.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		sampleResponse := SampleResponse{
			ID:           sample.ID,
			State:        sample.State,
			ProductIssue: sample.ProductIssue,
		}
		if sample.LocationID.Valid {
			sampleResponse.LocationID = &sample.LocationID.V
		}
		if sample.ProductID.Valid {
			sampleResponse.ProductID = &sample.ProductID.V
		}
		if sample.TimeRegistered.Valid {
			timeStr := sample.TimeRegistered.Time.Format(time.RFC3339)
			sampleResponse.TimeRegistered = timeStr
		}
		if sample.LastUpdate.Valid {
			timeStr := sample.LastUpdate.Time.Format(time.RFC3339)
			sampleResponse.LastUpdate = timeStr
		}
		if sample.OwnerID.Valid {
			sampleResponse.OwnerID = &sample.OwnerID.V
		}
		var modResponses []sample_mods.SampleModResponse
		for _, mod := range mod_data {
			modResponse := sample_mods.SampleModResponse{
				ID:        mod.ID,
				SampleID:  mod.SampleID,
				Name:      mod.Name,
				TimeAdded: mod.TimeAdded.Format(time.RFC3339),
			}
			if mod.TimeRemoved.Valid {
				timeStr := mod.TimeRemoved.Time.Format(time.RFC3339)
				modResponse.TimeRemoved = timeStr
			}
			modResponses = append(modResponses, modResponse)
		}
		samples = append(samples, SampleData{sampleResponse, modResponses})
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
