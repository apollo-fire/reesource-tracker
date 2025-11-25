package locations_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"reesource-tracker/api/locations"
	"reesource-tracker/lib/database"
	"reesource-tracker/lib/test_helpers/mock_db"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	group := r.Group("/api")
	locations.Routes(group)
	return r
}

func TestCreateLocation_MissingName(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	body := map[string]string{"description": "desc"}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/location", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "success")
}

func TestCreateLocation_EmptyName(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	body := map[string]string{"name": "", "description": "desc"}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/location", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "success")
}

func TestCreateLocation_Success(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	body := map[string]string{"name": "Test Location", "description": "desc"}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/location", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "success")
}

func TestCreateLocation_InvalidParentID(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	body := map[string]string{"name": "Test Location", "parent_location_id": "not-a-uuid"}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/location/123e4567-e89b-12d3-a456-426614174000", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

func TestGetLocation_MissingID(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/location/bad_id", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
}

func TestGetLocation_NotFound(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/location/00000000-0000-0000-0000-000000000000", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

func TestDeleteLocation_MissingID(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/location/", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
}

func TestDeleteLocation_Success(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/location/123e4567-e89b-12d3-a456-426614174000", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "deleted")
}

// Add more tests for get, update, delete, and edge cases as needed.
