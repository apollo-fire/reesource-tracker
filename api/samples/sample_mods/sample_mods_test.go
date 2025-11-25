package sample_mods_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"reesource-tracker/api/samples/sample_mods"
	"reesource-tracker/lib/database"
	"reesource-tracker/lib/test_helpers/mock_db"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	group := r.Group("/api/sample/:sample_id/mods")
	sample_mods.Routes(group)
	return r
}

func TestGetSampleMod_MissingID(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/sample/123e4567-e89b-12d3-a456-426614174000/mods/", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
}

func TestUpdateSampleMod_MissingID(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	body := map[string]string{"name": "Updated Mod"}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/sample/123e4567-e89b-12d3-a456-426614174000/mods/", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
}

func TestUpdateSampleMod_InvalidID(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	body := map[string]string{"name": "Updated Mod"}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/sample/123e4567-e89b-12d3-a456-426614174000/mods/not-a-uuid", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
}

func TestDeleteSampleMod_MissingID(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/sample/123e4567-e89b-12d3-a456-426614174000/mods/", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
}

func TestDeleteSampleMod_InvalidID(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/sample/123e4567-e89b-12d3-a456-426614174000/mods/not-a-uuid", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

func TestDeleteSampleMod_Success(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/sample/123e4567-e89b-12d3-a456-426614174000/mods/123e4567-e89b-12d3-a456-426614174001", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}
