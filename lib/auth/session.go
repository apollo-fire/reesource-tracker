package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"

	"reesource-tracker/lib/database"
)

// SessionDuration is the maximum lifetime of a DB session entry. The effective
// session lifetime is also bounded by the OIDC refresh token expiry.
const SessionDuration = 30 * 24 * time.Hour

type Session struct {
	ID                   string
	UserID               []byte
	Roles                []string
	OIDCSid              string
	IDToken              string
	RefreshToken         string
	AccessTokenExpiresAt time.Time
	ExpiresAt            time.Time
}

func NewSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func CreateSession(ctx context.Context, s Session) error {
	roles := s.Roles
	if roles == nil {
		roles = []string{}
	}
	return database.Connection.CreateSession(ctx, database.CreateSessionParams{
		ID:                   s.ID,
		UserID:               s.UserID,
		Roles:                roles,
		OidcSid:              nullString(s.OIDCSid),
		RefreshToken:         nullString(s.RefreshToken),
		AccessTokenExpiresAt: s.AccessTokenExpiresAt,
		ExpiresAt:            s.ExpiresAt,
		IDToken:              nullString(s.IDToken),
	})
}

func GetSession(ctx context.Context, id string) (*Session, error) {
	row, err := database.Connection.GetSessionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &Session{
		ID:                   row.ID,
		UserID:               row.UserID,
		Roles:                row.Roles,
		OIDCSid:              row.OidcSid.String,
		IDToken:              row.IDToken.String,
		RefreshToken:         row.RefreshToken.String,
		AccessTokenExpiresAt: row.AccessTokenExpiresAt,
		ExpiresAt:            row.ExpiresAt,
	}, nil
}

func UpdateSessionTokens(ctx context.Context, id, idToken, refreshToken string, accessTokenExpiry time.Time, roles []string) error {
	if roles == nil {
		roles = []string{}
	}
	return database.Connection.UpdateSessionTokens(ctx, database.UpdateSessionTokensParams{
		ID:                   id,
		IDToken:              nullString(idToken),
		RefreshToken:         nullString(refreshToken),
		AccessTokenExpiresAt: accessTokenExpiry,
		Roles:                roles,
	})
}

func DeleteSession(ctx context.Context, id string) error {
	return database.Connection.DeleteSession(ctx, id)
}

func DeleteUserSessions(ctx context.Context, userID []byte) error {
	return database.Connection.DeleteSessionsByUserID(ctx, userID)
}

func DeleteSessionByOIDCSID(ctx context.Context, oidcSid string) error {
	return database.Connection.DeleteSessionByOIDCSID(ctx, sql.NullString{
		String: oidcSid, Valid: oidcSid != "",
	})
}

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
