package auth

import (
	"context"
	"database/sql"
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
}

func BeginRegistration(ctx context.Context, assignmentToken string) (BeginRegistrationResult, error) {
	if assignmentToken == "" {
		return BeginRegistrationResult{}, errors.New("assignment_token is required")
	}

	link, err := database.Connection.GetActiveAssignmentLinkByTokenHash(ctx, assignmentToken)
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
	if in.ChallengeToken == "" || in.CredentialID == "" || in.PublicKey == "" {
		return nil, errors.New("challenge_token, credential_id and public_key are required")
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
		Column5:      transportBytes,
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
