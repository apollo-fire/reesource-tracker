package users_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"reesource-tracker/api/users"
	"reesource-tracker/lib/database"
	"reesource-tracker/lib/test_helpers/mock_db"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	group := r.Group("/api")
	users.Routes(group)
	return r
}

func TestCreateUser_Success(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	body := map[string]string{"name": "Test User"}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/user", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "success")
}

func TestGetUser_NotFound(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/user/00000000-0000-0000-0000-000000000000", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

func TestUpdateUser_InvalidID(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	body := map[string]string{"name": "Updated User"}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/user/not-a-uuid", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

func TestDeleteUser_MissingID(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/user/", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 404, w.Code)
}

func TestDeleteUser_InvalidID(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/user/not-a-uuid", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

func TestDeleteUser_Success(t *testing.T) {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/user/123e4567-e89b-12d3-a456-426614174000", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "deleted")
}

func TestGetUsers_ReturnsRoles(t *testing.T) {
	mock_db.ResetMockDB()
	database.Connection = mock_db.MockConnection
	r := setupRouter()

	// Create a user first so the list is non-empty.
	body := map[string]string{"name": "Role Test User"}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/user", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)

	// Now list users and verify the Roles field is present and populated.
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/users", nil)
	r.ServeHTTP(w2, req2)
	assert.Equal(t, 200, w2.Code)
	var resp []map[string]interface{}
	assert.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp)
	for _, user := range resp {
		rolesRaw, ok := user["Roles"]
		assert.True(t, ok, "Roles key should be present")
		roles, ok := rolesRaw.([]interface{})
		assert.True(t, ok, "Roles should be a JSON array")
		// createUser always assigns the 'user' role.
		assert.NotEmpty(t, roles, "user should have at least one role")
		roleStrings := make([]string, 0, len(roles))
		for _, r := range roles {
			s, isStr := r.(string)
			assert.True(t, isStr, "each role should be a string")
			roleStrings = append(roleStrings, s)
		}
		assert.Contains(t, roleStrings, "user", "newly created user should have 'user' role")
	}
}
