package auth_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"reesource-tracker/api/auth"
	"reesource-tracker/api/middleware"
	"reesource-tracker/lib/database"
	"reesource-tracker/lib/test_helpers/mock_db"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testPrivKey and testPubKeyDER are generated once and reused across all
// integration tests that need real WebAuthn key material.
var (
	testPrivKey   *ecdsa.PrivateKey
	testPubKeyDER []byte
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	middleware.SetCookieSecret([]byte("test-secret-key-for-tests"))

	var err error
	testPrivKey, err = ecdsa.GenerateKey(elliptic.P256(), cryptorand.Reader)
	if err != nil {
		panic("failed to generate test EC key: " + err.Error())
	}
	testPubKeyDER, err = x509.MarshalPKIXPublicKey(&testPrivKey.PublicKey)
	if err != nil {
		panic("failed to marshal test public key: " + err.Error())
	}

	os.Exit(m.Run())
}

func setupRouter() *gin.Engine {
	r := gin.New()
	group := r.Group("/api")
	auth.Routes(group)
	return r
}

// ── Bootstrap tests ──────────────────────────────────────────────────────────

// TestGetBootstrapStatus_BootstrapRequired verifies that an empty DB (no admin)
// causes the endpoint to report bootstrap as required.
func TestGetBootstrapStatus_BootstrapRequired(t *testing.T) {
	mock_db.ResetMockDB()
	database.Connection = mock_db.MockConnection

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/auth/bootstrap-status", nil)
	setupRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["bootstrap_required"])
}

// TestGetBootstrapOptions_ReturnsBootstrapRequired verifies that when no admin
// exists the options endpoint reports bootstrap as required and returns an empty
// users slice.
func TestGetBootstrapOptions_ReturnsBootstrapRequired(t *testing.T) {
	mock_db.ResetMockDB()
	database.Connection = mock_db.MockConnection

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/auth/bootstrap-options", nil)
	setupRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["bootstrap_required"])
	users, ok := resp["users"].([]interface{})
	assert.True(t, ok)
	assert.Empty(t, users)
}

// TestBootstrapSelectUser_MissingUserID verifies that omitting user_id returns 400.
func TestBootstrapSelectUser_MissingUserID(t *testing.T) {
	database.Connection = mock_db.MockConnection

	body, _ := json.Marshal(map[string]string{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/bootstrap/select-user", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	setupRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

// TestBootstrapSelectUser_InvalidUUID verifies that a non-UUID user_id returns 400.
func TestBootstrapSelectUser_InvalidUUID(t *testing.T) {
	database.Connection = mock_db.MockConnection

	body, _ := json.Marshal(map[string]string{"user_id": "not-a-uuid"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/bootstrap/select-user", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	setupRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

// TestBootstrapCreateUser_Success verifies that a new user can be created
// during the bootstrap flow and that an assignment token is returned.
func TestBootstrapCreateUser_Success(t *testing.T) {
	mock_db.ResetMockDB()
	database.Connection = mock_db.MockConnection

	body, _ := json.Marshal(map[string]string{"name": "Admin User"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/bootstrap/create-user", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	setupRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["assignment_token"])
	assert.NotEmpty(t, resp["user_id"])
}

// TestBootstrapSelectUser_WhenNotRequired verifies that when an admin already
// exists the select-user endpoint returns 400 with a clear message.
func TestBootstrapSelectUser_WhenNotRequired(t *testing.T) {
	mock_db.ResetMockDB()
	database.Connection = mock_db.MockConnection

	ctx := context.Background()
	uid, err := uuid.New().MarshalBinary()
	require.NoError(t, err)
	require.NoError(t, database.Connection.UpsertUserName(ctx, database.UpsertUserNameParams{ID: uid, Name: "Admin"}))
	require.NoError(t, database.Connection.SetUserRole(ctx, database.SetUserRoleParams{UserID: uid, Role: "admin"}))

	body, _ := json.Marshal(map[string]string{"user_id": "123e4567-e89b-12d3-a456-426614174000"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/bootstrap/select-user", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	setupRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "bootstrap flow is not active")
}

// TestBootstrapCreateUser_WhenNotRequired verifies that when an admin already
// exists the create-user endpoint also returns 400.
func TestBootstrapCreateUser_WhenNotRequired(t *testing.T) {
	mock_db.ResetMockDB()
	database.Connection = mock_db.MockConnection

	ctx := context.Background()
	uid, err := uuid.New().MarshalBinary()
	require.NoError(t, err)
	require.NoError(t, database.Connection.UpsertUserName(ctx, database.UpsertUserNameParams{ID: uid, Name: "Admin"}))
	require.NoError(t, database.Connection.SetUserRole(ctx, database.SetUserRoleParams{UserID: uid, Role: "admin"}))

	body, _ := json.Marshal(map[string]string{"name": "Another User"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/bootstrap/create-user", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	setupRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "bootstrap flow is not active")
}

// ── Session / Logout tests ────────────────────────────────────────────────────

// TestGetSession_AuthBypassed verifies that in test mode (gin.TestMode) the
// session endpoint reports the user as unauthenticated.
func TestGetSession_AuthBypassed(t *testing.T) {
	database.Connection = mock_db.MockConnection

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/auth/session", nil)
	setupRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, false, resp["authenticated"])
}

// TestLogout_Success verifies that calling /logout always returns 200.
func TestLogout_Success(t *testing.T) {
	database.Connection = mock_db.MockConnection

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	setupRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

// ── Registration tests ────────────────────────────────────────────────────────

// TestBeginRegistration_MissingToken verifies that an empty assignment_token
// returns 400.
func TestBeginRegistration_MissingToken(t *testing.T) {
	database.Connection = mock_db.MockConnection

	body, _ := json.Marshal(map[string]string{"assignment_token": ""})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/register/begin", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	setupRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

// TestBeginRegistration_InvalidToken verifies that a non-existent
// assignment_token returns 400.
func TestBeginRegistration_InvalidToken(t *testing.T) {
	database.Connection = mock_db.MockConnection

	body, _ := json.Marshal(map[string]string{"assignment_token": "nonexistenttoken123"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/register/begin", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	setupRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

// TestFinishRegistration_MissingFields verifies that sending an incomplete
// payload returns 400.
func TestFinishRegistration_MissingFields(t *testing.T) {
	database.Connection = mock_db.MockConnection

	body, _ := json.Marshal(map[string]string{"challenge_token": "tok"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/register/finish", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	setupRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

// TestFinishRegistration_InvalidCredentialID verifies that an invalid base64
// credential_id returns 400.
func TestFinishRegistration_InvalidCredentialID(t *testing.T) {
	database.Connection = mock_db.MockConnection

	body, _ := json.Marshal(map[string]string{
		"challenge_token": "tok",
		"challenge":       "chal",
		"credential_id":   "!!! not valid base64 !!!",
		"public_key":      base64.RawURLEncoding.EncodeToString([]byte("pk")),
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/register/finish", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	setupRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid credential_id")
}

// ── Login tests ───────────────────────────────────────────────────────────────

// TestBeginLogin_NoUserID verifies that login can be initiated without
// specifying a user, and a challenge is returned.
func TestBeginLogin_NoUserID(t *testing.T) {
	database.Connection = mock_db.MockConnection

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/login/begin", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	setupRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["challenge_token"])
	assert.NotEmpty(t, resp["challenge"])
}

// TestBeginLogin_InvalidUserID verifies that an invalid UUID user_id returns 400.
func TestBeginLogin_InvalidUserID(t *testing.T) {
	database.Connection = mock_db.MockConnection

	body, _ := json.Marshal(map[string]string{"user_id": "not-a-valid-uuid"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/login/begin", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	setupRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

// TestFinishLogin_MissingFields verifies that an empty payload returns 400.
func TestFinishLogin_MissingFields(t *testing.T) {
	database.Connection = mock_db.MockConnection

	body, _ := json.Marshal(map[string]string{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/login/finish", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	setupRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

// TestFinishLogin_InvalidCredentialID verifies that an invalid base64
// credential_id returns 400.
func TestFinishLogin_InvalidCredentialID(t *testing.T) {
	database.Connection = mock_db.MockConnection

	body, _ := json.Marshal(map[string]interface{}{
		"challenge_token": "tok",
		"challenge":       "chal",
		"credential_id":   "!!! invalid base64 !!!",
		"sign_counter":    0,
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/login/finish", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	setupRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid credential_id")
}

// TestFinishLogin_InvalidChallenge verifies that a non-existent challenge_token
// returns 400.
func TestFinishLogin_InvalidChallenge(t *testing.T) {
	database.Connection = mock_db.MockConnection

	credID := base64.RawURLEncoding.EncodeToString([]byte("some-credential"))
	body, _ := json.Marshal(map[string]interface{}{
		"challenge_token": "nonexistent-challenge-token",
		"challenge":       "some-challenge-value",
		"credential_id":   credID,
		"sign_counter":    0,
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/auth/login/finish", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	setupRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

// ── Self route tests (auth bypassed in test mode) ─────────────────────────────

// TestListSelfPasskeys_Success verifies that the self passkeys endpoint returns
// 200 (empty list when no passkeys exist for the nil user in test mode).
func TestListSelfPasskeys_Success(t *testing.T) {
	database.Connection = mock_db.MockConnection

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/auth/self/passkeys", nil)
	setupRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestGetSelfAssignmentLink_NoActiveLink verifies that the self assignment-link
// endpoint returns has_active_link=false when no link is present.
func TestGetSelfAssignmentLink_NoActiveLink(t *testing.T) {
	mock_db.ResetMockDB()
	database.Connection = mock_db.MockConnection

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/auth/self/assignment-link", nil)
	setupRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, false, resp["has_active_link"])
}

// ── Admin route tests (auth bypassed in test mode) ────────────────────────────

// TestAdminListAuditLogs_Success verifies that the admin audit-logs endpoint
// returns 200 (empty list when no events have been recorded).
func TestAdminListAuditLogs_Success(t *testing.T) {
	mock_db.ResetMockDB()
	database.Connection = mock_db.MockConnection

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/auth/admin/audit-logs", nil)
	setupRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// TestAdminGetAssignmentLink_InvalidUserID verifies that a non-UUID user_id
// path parameter returns 400.
func TestAdminGetAssignmentLink_InvalidUserID(t *testing.T) {
	database.Connection = mock_db.MockConnection

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/auth/admin/users/not-a-uuid/assignment-link", nil)
	setupRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

// TestAdminListUserPasskeys_InvalidUserID verifies that a non-UUID user_id
// path parameter returns 400.
func TestAdminListUserPasskeys_InvalidUserID(t *testing.T) {
	database.Connection = mock_db.MockConnection

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/auth/admin/users/not-a-uuid/passkeys", nil)
	setupRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "error")
}

// TestAdminListUserPasskeys_ValidUserID verifies that a valid UUID returns 200
// with a (possibly empty) list of passkeys.
func TestAdminListUserPasskeys_ValidUserID(t *testing.T) {
	database.Connection = mock_db.MockConnection

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/auth/admin/users/123e4567-e89b-12d3-a456-426614174000/passkeys", nil)
	setupRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ── Full-flow integration tests ───────────────────────────────────────────────

// TestFullRegistrationFlow tests the complete passkey registration flow:
//
//  1. bootstrap/create-user → assignment_token
//  2. register/begin        → challenge_token, challenge
//  3. register/finish       → session cookie set, 200 ok
func TestFullRegistrationFlow(t *testing.T) {
	mock_db.ResetMockDB()
	database.Connection = mock_db.MockConnection
	r := setupRouter()

	// Step 1: create bootstrap user.
	body1, _ := json.Marshal(map[string]string{"name": "Bootstrap Admin"})
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/auth/bootstrap/create-user", bytes.NewBuffer(body1))
	req1.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w1, req1)
	require.Equal(t, http.StatusOK, w1.Code, "create-user: %s", w1.Body)

	var createResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w1.Body.Bytes(), &createResp))
	assignmentToken, _ := createResp["assignment_token"].(string)
	require.NotEmpty(t, assignmentToken)

	// Step 2: begin registration.
	body2, _ := json.Marshal(map[string]string{"assignment_token": assignmentToken})
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/api/auth/register/begin", bytes.NewBuffer(body2))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code, "register/begin: %s", w2.Body)

	var beginResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &beginResp))
	challengeToken, _ := beginResp["challenge_token"].(string)
	challengeValue, _ := beginResp["challenge"].(string)
	require.NotEmpty(t, challengeToken)
	require.NotEmpty(t, challengeValue)

	// Step 3: finish registration with a real key pair and valid clientDataJSON.
	credentialID := base64.RawURLEncoding.EncodeToString([]byte("integration-test-credential-1"))
	publicKeyB64 := base64.RawURLEncoding.EncodeToString(testPubKeyDER)

	// Build a clientDataJSON matching the server's challenge.
	challengeHexBytes, err := hex.DecodeString(challengeValue)
	require.NoError(t, err)
	clientDataMap := map[string]interface{}{
		"type":      "webauthn.create",
		"challenge": base64.RawURLEncoding.EncodeToString(challengeHexBytes),
		"origin":    "http://localhost",
	}
	clientDataBytes, _ := json.Marshal(clientDataMap)
	clientDataJSONB64 := base64.RawURLEncoding.EncodeToString(clientDataBytes)

	body3, _ := json.Marshal(map[string]interface{}{
		"challenge_token":  challengeToken,
		"challenge":        challengeValue,
		"credential_id":    credentialID,
		"public_key":       publicKeyB64,
		"client_data_json": clientDataJSONB64,
		"label":            "Integration Test Passkey",
		"transports":       []string{"internal"},
	})
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest(http.MethodPost, "/api/auth/register/finish", bytes.NewBuffer(body3))
	req3.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code, "register/finish: %s", w3.Body)
	assert.Contains(t, w3.Body.String(), "ok")
}

// TestGetBootstrapStatus_NotRequired verifies that after a full registration
// (which promotes the user to admin) bootstrap is no longer required.
// This test depends on TestFullRegistrationFlow having already run.
func TestGetBootstrapStatus_NotRequired(t *testing.T) {
	mock_db.ResetMockDB()
	database.Connection = mock_db.MockConnection

	ctx := context.Background()
	uid, err := uuid.New().MarshalBinary()
	require.NoError(t, err)
	require.NoError(t, database.Connection.UpsertUserName(ctx, database.UpsertUserNameParams{ID: uid, Name: "Admin"}))
	require.NoError(t, database.Connection.SetUserRole(ctx, database.SetUserRoleParams{UserID: uid, Role: "admin"}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/auth/bootstrap-status", nil)
	setupRouter().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, false, resp["bootstrap_required"])
}

// TestFullLoginFlow tests the complete login flow using the passkey registered
// in TestFullRegistrationFlow (DB state is inherited from that test).
func TestFullLoginFlow(t *testing.T) {
	database.Connection = mock_db.MockConnection
	r := setupRouter()

	// Step 1: begin login (no user filter).
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/auth/login/begin", bytes.NewBufferString(`{}`))
	req1.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w1, req1)
	require.Equal(t, http.StatusOK, w1.Code, "login/begin: %s", w1.Body)

	var beginResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w1.Body.Bytes(), &beginResp))
	challengeToken, _ := beginResp["challenge_token"].(string)
	challengeValue, _ := beginResp["challenge"].(string)
	require.NotEmpty(t, challengeToken)

	// Step 2: finish login with the credential from TestFullRegistrationFlow,
	// using a real WebAuthn assertion signed with the test private key.
	credentialID := base64.RawURLEncoding.EncodeToString([]byte("integration-test-credential-1"))

	// Build clientDataJSON for the login challenge.
	loginChallengeHexBytes, err := hex.DecodeString(challengeValue)
	require.NoError(t, err)
	loginClientDataMap := map[string]interface{}{
		"type":      "webauthn.get",
		"challenge": base64.RawURLEncoding.EncodeToString(loginChallengeHexBytes),
		"origin":    "http://localhost",
	}
	loginClientDataBytes, _ := json.Marshal(loginClientDataMap)
	loginClientDataJSONB64 := base64.RawURLEncoding.EncodeToString(loginClientDataBytes)

	// Build a minimal 37-byte authenticatorData (rpIdHash + flags + counter).
	rpIdHash := sha256.Sum256([]byte("localhost"))
	authDataBytes := make([]byte, 37)
	copy(authDataBytes[0:32], rpIdHash[:])
	authDataBytes[32] = 0x01 // UP (user present) flag
	binary.BigEndian.PutUint32(authDataBytes[33:37], 1)

	// Sign: SHA-256(authenticatorData || SHA-256(clientDataJSON)).
	cdHash := sha256.Sum256(loginClientDataBytes)
	signedData := make([]byte, len(authDataBytes)+sha256.Size)
	copy(signedData, authDataBytes)
	copy(signedData[len(authDataBytes):], cdHash[:])
	msgHash := sha256.Sum256(signedData)
	sig, err := ecdsa.SignASN1(cryptorand.Reader, testPrivKey, msgHash[:])
	require.NoError(t, err)

	body2, _ := json.Marshal(map[string]interface{}{
		"challenge_token":    challengeToken,
		"challenge":          challengeValue,
		"credential_id":      credentialID,
		"sign_counter":       1,
		"client_data_json":   loginClientDataJSONB64,
		"authenticator_data": base64.RawURLEncoding.EncodeToString(authDataBytes),
		"signature":          base64.RawURLEncoding.EncodeToString(sig),
	})
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/api/auth/login/finish", bytes.NewBuffer(body2))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code, "login/finish: %s", w2.Body)
	assert.Contains(t, w2.Body.String(), "ok")
}
