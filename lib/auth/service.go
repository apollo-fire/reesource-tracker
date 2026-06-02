package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"reesource-tracker/lib/database"
	id_helper "reesource-tracker/lib/id_helper"

	"github.com/google/uuid"
)

type RuntimeConfig struct {
	AuditRetentionDays int
}

type BootstrapUserOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

const (
	NormalAssignmentLinkLifetime = time.Hour
	SessionDuration              = 24 * time.Hour
)

func EnsureBootstrapState(ctx context.Context) (bool, string, string, error) {
	hasAdmin, err := database.Connection.AnyAdminExists(ctx)
	if err != nil {
		return false, "", "", err
	}
	if hasAdmin {
		return false, "", "", nil
	}

	active, err := database.Connection.GetActiveBootstrapLink(ctx)
	if err == nil {
		uid, _ := id_helper.UnmarshalUUID(active.UserID)
		// The raw token is not recoverable from the stored hash. Return an
		// empty token; callers must initiate a fresh bootstrap flow to obtain
		// a usable assignment token.
		return true, "", uid, nil
	}
	if err != sql.ErrNoRows {
		return false, "", "", err
	}

	// No active bootstrap link yet. Let the user choose an existing account or create a new one.
	return true, "", "", nil
}

func ListBootstrapUserOptions(ctx context.Context) ([]BootstrapUserOption, error) {
	users, err := database.Connection.ListUsersWithoutAdmin(ctx)
	if err != nil {
		return nil, err
	}

	options := make([]BootstrapUserOption, 0, len(users))
	for _, u := range users {
		options = append(options, BootstrapUserOption{ID: IDOrEmpty(u.ID), Name: u.Name})
	}
	return options, nil
}

func SelectBootstrapUser(ctx context.Context, userID []byte) (string, string, error) {
	token, err := RandomHex(24)
	if err != nil {
		return "", "", err
	}
	tokenHash := HashToken(token)

	if err := database.Connection.RevokeActiveBootstrapLinks(ctx); err != nil {
		return "", "", err
	}
	row, err := database.Connection.CreateAssignmentLink(ctx, database.CreateAssignmentLinkParams{
		TokenHash:       tokenHash,
		UserID:          userID,
		CreatedByUserID: sql.Null[[]byte]{Valid: false},
		Purpose:         "bootstrap",
		ExpiresAt:       sql.NullTime{Valid: false},
	})
	if err != nil {
		return "", "", err
	}
	uid, _ := id_helper.UnmarshalUUID(row.UserID)
	_ = AuditLog(ctx, nil, "bootstrap_link_created", "user", uid, map[string]any{})
	return token, uid, nil
}

func CreateBootstrapUserAndSelect(ctx context.Context, name string) (string, string, error) {
	if name == "" {
		name = "New Admin"
	}
	newID, err := uuid.New().MarshalBinary()
	if err != nil {
		return "", "", err
	}
	if err := database.Connection.UpsertUserName(ctx, database.UpsertUserNameParams{ID: newID, Name: name}); err != nil {
		return "", "", err
	}
	_ = AuditLog(ctx, nil, "bootstrap_user_created", "user", IDOrEmpty(newID), map[string]any{"name": name})
	return SelectBootstrapUser(ctx, newID)
}

func RunAuditRetentionCleanup(ctx context.Context, cfg RuntimeConfig) {
	if cfg.AuditRetentionDays <= 0 {
		cfg.AuditRetentionDays = 90
	}

	cleanup := func() {
		cutoff := time.Now().Add(-time.Duration(cfg.AuditRetentionDays) * 24 * time.Hour)
		_ = database.Connection.DeleteAuditLogsOlderThan(ctx, cutoff)
		_ = database.Connection.DeleteExpiredAuthChallenges(ctx)
	}

	cleanup()

	go func() {
		ticker := time.NewTicker(12 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cleanup()
			}
		}
	}()
}

// CreateStandardAssignmentLinkForUser creates a new passkey-assignment link for
// targetUserID. Only a SHA-256 hash of the raw token is persisted; the raw
// token is returned as the second value and must be delivered to the recipient
// at this point — it cannot be recovered later.
func CreateStandardAssignmentLinkForUser(ctx context.Context, targetUserID []byte, createdBy []byte) (database.PasskeyAssignmentLink, string, error) {
	token, err := RandomHex(24)
	if err != nil {
		return database.PasskeyAssignmentLink{}, "", err
	}
	tokenHash := HashToken(token)

	if _, err := database.Connection.RevokeActiveStandardAssignmentLinksForUser(ctx, targetUserID); err != nil {
		return database.PasskeyAssignmentLink{}, "", err
	}

	row, err := database.Connection.CreateAssignmentLink(ctx, database.CreateAssignmentLinkParams{
		TokenHash:       tokenHash,
		UserID:          targetUserID,
		CreatedByUserID: sql.Null[[]byte]{V: createdBy, Valid: createdBy != nil},
		Purpose:         "standard",
		ExpiresAt:       sql.NullTime{Time: time.Now().Add(NormalAssignmentLinkLifetime), Valid: true},
	})
	if err != nil {
		return database.PasskeyAssignmentLink{}, "", err
	}
	_ = AuditLog(ctx, &createdBy, "assignment_link_created", "user", IDOrEmpty(targetUserID), map[string]any{"purpose": "standard"})
	return row, token, nil
}

func AuditLog(ctx context.Context, actorUserID *[]byte, action, targetType, targetID string, metadata interface{}) error {
	payload, err := json.Marshal(metadata)
	if err != nil {
		payload = []byte("{}")
	}

	actor := sql.Null[[]byte]{Valid: false}
	if actorUserID != nil {
		actor = sql.Null[[]byte]{V: *actorUserID, Valid: true}
	}

	return database.Connection.InsertAuditLog(ctx, database.InsertAuditLogParams{
		ActorUserID: actor,
		Action:      action,
		TargetType:  targetType,
		TargetID:    sql.NullString{String: targetID, Valid: targetID != ""},
		Column5:     payload,
	})
}

func IDOrEmpty(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	s, err := id_helper.UnmarshalUUID(raw)
	if err != nil {
		return ""
	}
	return s
}
