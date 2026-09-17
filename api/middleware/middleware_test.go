package middleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	libauth "reesource-tracker/lib/auth"
	"reesource-tracker/lib/database"
	liboidc "reesource-tracker/lib/oidc"
	"reesource-tracker/lib/test_helpers/mock_db"

	"github.com/stretchr/testify/require"
)

func TestLockSessionRefreshSerializesSameSession(t *testing.T) {
	unlock := lockSessionRefresh("shared-session")

	acquired := make(chan struct{})
	go func() {
		release := lockSessionRefresh("shared-session")
		close(acquired)
		release()
	}()

	select {
	case <-acquired:
		t.Fatal("second refresh lock acquisition should block for the same session")
	case <-time.After(50 * time.Millisecond):
	}

	unlock()

	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("second refresh lock acquisition should resume after the first unlocks")
	}
}

func TestLockSessionRefreshAllowsDifferentSessions(t *testing.T) {
	unlock := lockSessionRefresh("session-a")
	defer unlock()

	acquired := make(chan struct{})
	go func() {
		release := lockSessionRefresh("session-b")
		close(acquired)
		release()
	}()

	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("different sessions should not block each other while refreshing")
	}
}

func TestApplyRefreshedSessionPersistsRefreshedRoles(t *testing.T) {
	mock_db.ResetMockDB()
	database.Connection = mock_db.MockConnection

	ctx := context.Background()
	sessionID := "session-1"
	userID := []byte{
		0x12, 0x3e, 0x45, 0x67, 0xe8, 0x9b, 0x12, 0xd3,
		0xa4, 0x56, 0x42, 0x66, 0x14, 0x17, 0x40, 0x00,
	}
	_, err := database.Connection.UpsertUserByOIDCSub(ctx, database.UpsertUserByOIDCSubParams{
		ID:      userID,
		Name:    "Test User",
		OidcSub: sql.NullString{String: "test-user", Valid: true},
	})
	require.NoError(t, err)

	err = libauth.CreateSession(ctx, libauth.Session{
		ID:                   sessionID,
		UserID:               userID,
		Roles:                []string{libauth.RoleAdmin},
		IDToken:              "old-id-token",
		RefreshToken:         "old-refresh-token",
		AccessTokenExpiresAt: time.Now().Add(-time.Minute),
		ExpiresAt:            time.Now().Add(time.Hour),
	})
	require.NoError(t, err)

	session, err := libauth.GetSession(ctx, sessionID)
	require.NoError(t, err)

	newTokens := &liboidc.TokenSet{
		RawIDToken:        "new-id-token",
		RefreshToken:      "new-refresh-token",
		AccessTokenExpiry: time.Now().Add(time.Hour),
		RawClaims:         json.RawMessage(`{"groups":["user"]}`),
	}

	err = applyRefreshedSession(ctx, sessionID, session, newTokens, "groups", nil)
	require.NoError(t, err)
	require.Equal(t, []string{libauth.RoleUser}, session.Roles)
	require.Equal(t, "new-id-token", session.IDToken)

	stored, err := libauth.GetSession(ctx, sessionID)
	require.NoError(t, err)
	require.Equal(t, []string{libauth.RoleUser}, stored.Roles)
	require.Equal(t, "new-id-token", stored.IDToken)
	require.Equal(t, "new-refresh-token", stored.RefreshToken)
	require.WithinDuration(t, newTokens.AccessTokenExpiry, stored.AccessTokenExpiresAt, time.Second)
}
