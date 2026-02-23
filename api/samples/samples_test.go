package samples_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"reesource-tracker/api/samples"
	"reesource-tracker/lib/database"
	"reesource-tracker/lib/test_helpers/mock_db"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	group := r.Group("/api")
	samples.Routes(group)
	return r
}

func TestGetSample_InvalidID(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/sample/invalid-id", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid sample ID format")
}

func TestUpdateSample_MissingID(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/sample/", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
}

func TestUpdateSample_InvalidID(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/sample/invalid-id", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid sample ID format")
}

func TestGenerateUniqueSamples_InvalidNum(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/generate_samples?num_samples=notanumber", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid number of samples")
}

func TestGenerateUniqueSamples_MissingNum(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/generate_samples", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid number of samples")
}
