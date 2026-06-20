package auth

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"reesource-tracker/lib/database"
	id_helper "reesource-tracker/lib/id_helper"
)

type registrationChallengePayload struct {
	ChallengeValue   string `json:"challenge_value"`
	AssignmentLinkID int64  `json:"assignment_link_id,omitempty"`
	UserID           string `json:"user_id,omitempty"`
	Flow             string `json:"flow"`
}

type BeginRegistrationResult struct {
	ChallengeToken string
	Challenge      string
	UserID         string
	UserName       string
}

type FinishRegistrationInput struct {
	ChallengeToken string
	Challenge      string
	CredentialID   string
	PublicKey      string
	Label          string
	Transports     []string
	// ClientDataJSON is the base64url-encoded raw clientDataJSON bytes from
	// the authenticator. Required for server-side challenge verification.
	ClientDataJSON string
}

type BeginLoginResult struct {
	ChallengeToken string
	Challenge      string
}

type FinishLoginInput struct {
	ChallengeToken string
	Challenge      string
	CredentialID   string
	SignCounter    int64
	// WebAuthn assertion fields required for server-side verification.
	ClientDataJSON    string
	AuthenticatorData string
	Signature         string
}

func BeginRegistration(ctx context.Context, assignmentToken string) (BeginRegistrationResult, error) {
	if assignmentToken == "" {
		return BeginRegistrationResult{}, errors.New("assignment_token is required")
	}

	link, err := database.Connection.GetActiveAssignmentLinkByTokenHash(ctx, HashToken(assignmentToken))
	if err != nil {
		return BeginRegistrationResult{}, errors.New("invalid or expired assignment link")
	}

	challengeToken, err := RandomHex(24)
	if err != nil {
		return BeginRegistrationResult{}, err
	}
	challengeValue, err := RandomHex(32)
	if err != nil {
		return BeginRegistrationResult{}, err
	}

	userIDString, _ := id_helper.UnmarshalUUID(link.UserID)
	userName := userIDString
	if user, userErr := database.Connection.GetUserByID(ctx, link.UserID); userErr == nil && user.Name != "" {
		userName = user.Name
	}
	payload := registrationChallengePayload{
		ChallengeValue:   challengeValue,
		AssignmentLinkID: link.ID,
		UserID:           userIDString,
		Flow:             "register",
	}
	payloadBytes, _ := json.Marshal(payload)

	err = database.Connection.InsertAuthChallenge(ctx, database.InsertAuthChallengeParams{
		ChallengeToken: challengeToken,
		ChallengeBytes: payloadBytes,
		UserID:         sql.Null[[]byte]{V: link.UserID, Valid: true},
		FlowType:       "register",
		ExpiresAt:      time.Now().Add(10 * time.Minute),
	})
	if err != nil {
		return BeginRegistrationResult{}, err
	}

	return BeginRegistrationResult{
		ChallengeToken: challengeToken,
		Challenge:      challengeValue,
		UserID:         userIDString,
		UserName:       userName,
	}, nil
}

func FinishRegistration(ctx context.Context, in FinishRegistrationInput) ([]byte, error) {
	if in.ChallengeToken == "" || in.CredentialID == "" || in.PublicKey == "" || in.ClientDataJSON == "" {
		return nil, errors.New("challenge_token, credential_id, public_key and client_data_json are required")
	}

	challenge, err := database.Connection.GetActiveAuthChallenge(ctx, in.ChallengeToken)
	if err != nil {
		return nil, errors.New("invalid challenge")
	}

	var payload registrationChallengePayload
	if err := json.Unmarshal(challenge.ChallengeBytes, &payload); err != nil {
		return nil, errors.New("invalid challenge payload")
	}
	if payload.Flow != "register" || payload.ChallengeValue != in.Challenge {
		return nil, errors.New("challenge mismatch")
	}

	// Verify the authenticator used the server-issued challenge.
	if err := verifyClientDataJSON(in.ClientDataJSON, "webauthn.create", payload.ChallengeValue); err != nil {
		return nil, err
	}

	link, err := database.Connection.GetAssignmentLinkByID(ctx, payload.AssignmentLinkID)
	if err != nil {
		return nil, errors.New("assignment link not found")
	}
	if link.ConsumedAt.Valid || link.RevokedAt.Valid || (link.ExpiresAt.Valid && link.ExpiresAt.Time.Before(time.Now())) {
		return nil, errors.New("assignment link inactive")
	}

	credentialID, err := DecodeBase64(in.CredentialID)
	if err != nil {
		return nil, errors.New("invalid credential_id")
	}
	publicKey, err := DecodeBase64(in.PublicKey)
	if err != nil {
		return nil, errors.New("invalid public_key")
	}
	transportBytes, _ := json.Marshal(in.Transports)
	label := in.Label
	if label == "" {
		label = "Sign-in passkey"
	}

	if err := database.Connection.InsertPasskey(ctx, database.InsertPasskeyParams{
		CredentialID: credentialID,
		UserID:       challenge.UserID.V,
		PublicKey:    publicKey,
		SignCounter:  0,
		Transports:   transportBytes,
		Label:        sql.NullString{String: label, Valid: true},
	}); err != nil {
		return nil, err
	}

	_ = database.Connection.ConsumeAssignmentLink(ctx, link.ID)
	if link.Purpose == "bootstrap" {
		_ = database.Connection.SetUserRole(ctx, database.SetUserRoleParams{UserID: challenge.UserID.V, Role: RoleAdmin})
		_ = AuditLog(ctx, nil, "bootstrap_assigned_admin", "user", IDOrEmpty(challenge.UserID.V), map[string]any{})
	}

	_ = database.Connection.MarkAuthChallengeUsed(ctx, in.ChallengeToken)
	_ = AuditLog(ctx, &challenge.UserID.V, "passkey_registered", "user", IDOrEmpty(challenge.UserID.V), map[string]any{"label": in.Label})

	return challenge.UserID.V, nil
}

func BeginLogin(ctx context.Context, userIDText string) (BeginLoginResult, error) {
	var userIDNull sql.Null[[]byte]
	if userIDText != "" {
		userID, msg, ok := id_helper.MustParseAndMarshalUUID(userIDText)
		if !ok {
			return BeginLoginResult{}, errors.New(msg)
		}
		userIDNull = sql.Null[[]byte]{V: userID, Valid: true}
	}

	challengeToken, err := RandomHex(24)
	if err != nil {
		return BeginLoginResult{}, err
	}
	challengeValue, err := RandomHex(32)
	if err != nil {
		return BeginLoginResult{}, err
	}

	payload := registrationChallengePayload{ChallengeValue: challengeValue, UserID: userIDText, Flow: "login"}
	payloadBytes, _ := json.Marshal(payload)
	if err := database.Connection.InsertAuthChallenge(ctx, database.InsertAuthChallengeParams{
		ChallengeToken: challengeToken,
		ChallengeBytes: payloadBytes,
		UserID:         userIDNull,
		FlowType:       "login",
		ExpiresAt:      time.Now().Add(10 * time.Minute),
	}); err != nil {
		return BeginLoginResult{}, err
	}

	return BeginLoginResult{ChallengeToken: challengeToken, Challenge: challengeValue}, nil
}

func FinishLogin(ctx context.Context, in FinishLoginInput) ([]byte, error) {
	if in.ChallengeToken == "" || in.CredentialID == "" {
		return nil, errors.New("challenge_token and credential_id are required")
	}
	if in.ClientDataJSON == "" || in.AuthenticatorData == "" || in.Signature == "" {
		return nil, errors.New("client_data_json, authenticator_data and signature are required")
	}

	challenge, err := database.Connection.GetActiveAuthChallenge(ctx, in.ChallengeToken)
	if err != nil {
		return nil, errors.New("invalid challenge")
	}

	var payload registrationChallengePayload
	if err := json.Unmarshal(challenge.ChallengeBytes, &payload); err != nil || payload.ChallengeValue != in.Challenge || payload.Flow != "login" {
		return nil, errors.New("challenge mismatch")
	}

	credentialID, err := DecodeBase64(in.CredentialID)
	if err != nil {
		return nil, errors.New("invalid credential_id")
	}

	passkey, err := database.Connection.GetPasskeyByCredentialID(ctx, credentialID)
	if err != nil || passkey.RevokedAt.Valid {
		return nil, errors.New("credential not allowed")
	}
	if challenge.UserID.Valid && !BytesEqual(passkey.UserID, challenge.UserID.V) {
		return nil, errors.New("credential does not match user")
	}

	// Verify the WebAuthn assertion: challenge binding and signature.
	if err := verifyPasskeyAssertion(in.ClientDataJSON, in.AuthenticatorData, in.Signature, passkey.PublicKey, payload.ChallengeValue); err != nil {
		return nil, err
	}

	if in.SignCounter > passkey.SignCounter {
		_ = database.Connection.UpdatePasskeySignCounter(ctx, database.UpdatePasskeySignCounterParams{
			CredentialID: credentialID,
			SignCounter:  in.SignCounter,
		})
	}

	_ = database.Connection.MarkAuthChallengeUsed(ctx, in.ChallengeToken)
	_ = AuditLog(ctx, &passkey.UserID, "passkey_login", "user", IDOrEmpty(passkey.UserID), map[string]any{})

	return passkey.UserID, nil
}

// webAuthnClientData is the subset of the clientDataJSON object that we need
// for server-side WebAuthn verification.
type webAuthnClientData struct {
	Type      string `json:"type"`
	Challenge string `json:"challenge"`
}

// verifyClientDataJSON decodes the base64url-encoded clientDataJSON, checks
// that the embedded type matches expectedType, and verifies that the challenge
// bytes equal the SHA-256-hex challenge stored on the server.
func verifyClientDataJSON(clientDataJSONB64, expectedType, expectedChallengeHex string) error {
	raw, err := DecodeBase64(clientDataJSONB64)
	if err != nil {
		return errors.New("invalid client_data_json encoding")
	}
	var cd webAuthnClientData
	if err := json.Unmarshal(raw, &cd); err != nil {
		return errors.New("invalid client_data_json format")
	}
	if cd.Type != expectedType {
		return errors.New("unexpected webauthn type in client_data_json")
	}
	expectedBytes, err := hex.DecodeString(expectedChallengeHex)
	if err != nil {
		return errors.New("internal: invalid challenge hex")
	}
	receivedBytes, err := DecodeBase64(cd.Challenge)
	if err != nil {
		return errors.New("invalid challenge encoding in client_data_json")
	}
	if !bytes.Equal(expectedBytes, receivedBytes) {
		return errors.New("challenge mismatch in client_data_json")
	}
	return nil
}

// verifyPasskeyAssertion validates a WebAuthn authentication assertion. It
// verifies the challenge embedded in clientDataJSON and then checks the ECDSA
// or RSA signature over the standard WebAuthn signed-data:
//
//	authenticatorData || SHA-256(clientDataJSON)
func verifyPasskeyAssertion(clientDataJSONB64, authenticatorDataB64, signatureB64 string, storedPublicKey []byte, expectedChallengeHex string) error {
	raw, err := DecodeBase64(clientDataJSONB64)
	if err != nil {
		return errors.New("invalid client_data_json encoding")
	}
	var cd webAuthnClientData
	if err := json.Unmarshal(raw, &cd); err != nil {
		return errors.New("invalid client_data_json format")
	}
	if cd.Type != "webauthn.get" {
		return errors.New("unexpected webauthn type in client_data_json")
	}
	expectedBytes, err := hex.DecodeString(expectedChallengeHex)
	if err != nil {
		return errors.New("internal: invalid challenge hex")
	}
	receivedBytes, err := DecodeBase64(cd.Challenge)
	if err != nil {
		return errors.New("invalid challenge encoding in client_data_json")
	}
	if !bytes.Equal(expectedBytes, receivedBytes) {
		return errors.New("challenge mismatch in client_data_json")
	}

	authData, err := DecodeBase64(authenticatorDataB64)
	if err != nil {
		return errors.New("invalid authenticator_data encoding")
	}
	if len(authData) < 37 {
		return errors.New("authenticator_data too short")
	}

	sig, err := DecodeBase64(signatureB64)
	if err != nil {
		return errors.New("invalid signature encoding")
	}

	pub, err := x509.ParsePKIXPublicKey(storedPublicKey)
	if err != nil {
		return errors.New("cannot parse stored public key")
	}

	// signedData = authenticatorData || SHA-256(clientDataJSON)
	clientDataHash := sha256.Sum256(raw)
	signedData := make([]byte, len(authData)+sha256.Size)
	copy(signedData, authData)
	copy(signedData[len(authData):], clientDataHash[:])
	msgHash := sha256.Sum256(signedData)

	switch key := pub.(type) {
	case *ecdsa.PublicKey:
		if !ecdsa.VerifyASN1(key, msgHash[:], sig) {
			return errors.New("assertion signature verification failed")
		}
	case *rsa.PublicKey:
		if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, msgHash[:], sig); err != nil {
			return errors.New("assertion signature verification failed")
		}
	default:
		return errors.New("unsupported public key algorithm")
	}
	return nil
}
