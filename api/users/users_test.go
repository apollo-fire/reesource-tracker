package users_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"reesource-tracker/api/users"
	"reesource-tracker/lib/database"
	"reesource-tracker/lib/test_helpers/mock_db"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	group := r.Group("/api")
	users.Routes(group)
	return r
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

func TestUpdateUser_NotFound(t *testing.T) {
	mock_db.ResetMockDB()
	database.Connection = mock_db.MockConnection
	r := setupRouter()

	body := map[string]string{"name": "Updated User"}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/user/123e4567-e89b-12d3-a456-426614174000", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, 404, w.Code)
	assert.Contains(t, w.Body.String(), sql.ErrNoRows.Error())
}

func TestUpdateUser_Success(t *testing.T) {
	mock_db.ResetMockDB()
	database.Connection = mock_db.MockConnection
	r := setupRouter()

	userID := []byte{
		0x12, 0x3e, 0x45, 0x67, 0xe8, 0x9b, 0x12, 0xd3,
		0xa4, 0x56, 0x42, 0x66, 0x14, 0x17, 0x40, 0x00,
	}
	_, err := database.Connection.UpsertUserByOIDCSub(context.Background(), database.UpsertUserByOIDCSubParams{
		ID:      userID,
		Name:    "Original User",
		OidcSub: sql.NullString{String: "seed-user", Valid: true},
	})
	require.NoError(t, err)

	body := map[string]string{"name": "Updated User"}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/user/123e4567-e89b-12d3-a456-426614174000", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, 200, w.Code)

	user, err := database.Connection.GetUserByID(context.Background(), userID)
	require.NoError(t, err)
	assert.Equal(t, "Updated User", user.Name)
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

func newBinaryUUID(t *testing.T) ([]byte, string) {
	id := uuid.New()
	b, err := id.MarshalBinary()
	assert.NoError(t, err)
	return b, id.String()
}

func mergeRequest(t *testing.T, legacyID, targetID string) *httptest.ResponseRecorder {
	r := setupRouter()
	database.Connection = mock_db.MockConnection
	body := map[string]string{"target_user_id": targetID}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/user/"+legacyID+"/merge", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestMergeUser_MissingTargetID(t *testing.T) {
	w := mergeRequest(t, "123e4567-e89b-12d3-a456-426614174000", "")
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "target_user_id")
}

func TestMergeUser_InvalidLegacyID(t *testing.T) {
	w := mergeRequest(t, "not-a-uuid", "123e4567-e89b-12d3-a456-426614174000")
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

func TestMergeUser_LegacyNotFound(t *testing.T) {
	w := mergeRequest(t, "00000000-0000-0000-0000-000000000000", "123e4567-e89b-12d3-a456-426614174000")
	assert.Equal(t, 404, w.Code)
	assert.Contains(t, w.Body.String(), "legacy user not found")
}

func TestMergeUser_LegacyAlreadyHasOIDC(t *testing.T) {
	database.Connection = mock_db.MockConnection
	oidcID, oidcIDStr := newBinaryUUID(t)
	_, err := database.Connection.UpsertUserByOIDCSub(t.Context(), database.UpsertUserByOIDCSubParams{
		ID:      oidcID,
		Name:    "Already Linked",
		OidcSub: sql.NullString{String: "sub-" + oidcIDStr, Valid: true},
	})
	assert.NoError(t, err)

	targetID, targetIDStr := newBinaryUUID(t)
	err = database.Connection.UpsertUser(t.Context(), database.UpsertUserParams{ID: targetID, Name: "Target"})
	assert.NoError(t, err)

	w := mergeRequest(t, oidcIDStr, targetIDStr)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "already has an OIDC account linked")
}

func TestMergeUser_TargetNotFound(t *testing.T) {
	legacyID, legacyIDStr := newBinaryUUID(t)
	err := database.Connection.UpsertUser(t.Context(), database.UpsertUserParams{ID: legacyID, Name: "Legacy"})
	assert.NoError(t, err)

	w := mergeRequest(t, legacyIDStr, "00000000-0000-0000-0000-000000000000")
	assert.Equal(t, 404, w.Code)
	assert.Contains(t, w.Body.String(), "target user not found")
}

func TestMergeUser_TargetNotOIDC(t *testing.T) {
	legacyID, legacyIDStr := newBinaryUUID(t)
	err := database.Connection.UpsertUser(t.Context(), database.UpsertUserParams{ID: legacyID, Name: "Legacy"})
	assert.NoError(t, err)

	targetID, targetIDStr := newBinaryUUID(t)
	err = database.Connection.UpsertUser(t.Context(), database.UpsertUserParams{ID: targetID, Name: "Also Legacy"})
	assert.NoError(t, err)

	w := mergeRequest(t, legacyIDStr, targetIDStr)
	assert.Equal(t, 400, w.Code)
	assert.Contains(t, w.Body.String(), "target user has no OIDC account linked")
}

func TestMergeUser_Success(t *testing.T) {
	database.Connection = mock_db.MockConnection

	legacyID, legacyIDStr := newBinaryUUID(t)
	err := database.Connection.UpsertUser(t.Context(), database.UpsertUserParams{ID: legacyID, Name: "Legacy"})
	assert.NoError(t, err)

	targetID, targetIDStr := newBinaryUUID(t)
	_, err = database.Connection.UpsertUserByOIDCSub(t.Context(), database.UpsertUserByOIDCSubParams{
		ID:      targetID,
		Name:    "Target",
		OidcSub: sql.NullString{String: "sub-" + targetIDStr, Valid: true},
	})
	assert.NoError(t, err)

	sampleID, _ := newBinaryUUID(t)
	_, err = database.Connection.UpdateOrCreateSample(t.Context(), database.UpdateOrCreateSampleParams{
		ID:      sampleID,
		State:   "unassigned",
		OwnerID: sql.Null[[]byte]{V: legacyID, Valid: true},
	})
	assert.NoError(t, err)

	w := mergeRequest(t, legacyIDStr, targetIDStr)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, w.Body.String(), "merged")

	_, err = database.Connection.GetUserByID(t.Context(), legacyID)
	assert.Error(t, err)

	sample, err := database.Connection.GetSampleById(t.Context(), sampleID)
	assert.NoError(t, err)
	assert.Equal(t, targetID, sample.OwnerID.V)
}
