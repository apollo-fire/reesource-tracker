package auth

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"reesource-tracker/lib/database"
	id_helper "reesource-tracker/lib/id_helper"
)

const (
	MagicLinkLifetime = 10 * time.Minute
	MagicLinkCooldown = 30 * time.Second
)

// MagicLinkEnabled reports whether the magic link feature is configured.
func MagicLinkEnabled() bool {
	return os.Getenv("MAGIC_LINK_WEBHOOK_URL") != ""
}

// MagicLinkRequest holds the raw token and user details produced by
// PrepareMagicLink, ready for the caller to build the login URL and dispatch
// the notification.
type MagicLinkRequest struct {
	Token    string
	UserName string
	Email    string
}

// PrepareMagicLink looks up the user by email, enforces the per-user rate
// limit, discards any existing pending links, creates a new one and returns
// the raw token together with user details.
//
// Returns nil (no error) when the email address is not registered; the caller
// should respond with HTTP 200 without indicating whether the email exists.
func PrepareMagicLink(ctx context.Context, email string) (*MagicLinkRequest, error) {
	if !MagicLinkEnabled() {
		return nil, errors.New("magic link sign-in is not enabled")
	}

	user, err := database.Connection.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // unknown email – silently ignore
		}
		return nil, err
	}

	// Enforce rate limit using the most recently created link (used or not).
	existing, err := database.Connection.GetLatestMagicLinkForUser(ctx, user.ID)
	if err == nil {
		cooldownEnd := existing.CreatedAt.Add(MagicLinkCooldown)
		if time.Now().Before(cooldownEnd) {
			remaining := time.Until(cooldownEnd).Round(time.Second)
			return nil, fmt.Errorf("please wait %s before requesting another sign-in link", remaining)
		}
	}

	// Discard any unexpired pending links for this user.
	_ = database.Connection.DeleteMagicLinksForUser(ctx, user.ID)

	token, err := RandomHex(32)
	if err != nil {
		return nil, err
	}

	_, err = database.Connection.CreateMagicLink(ctx, database.CreateMagicLinkParams{
		TokenHash: HashToken(token),
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(MagicLinkLifetime),
	})
	if err != nil {
		return nil, err
	}

	userIDStr, _ := id_helper.UnmarshalUUID(user.ID)
	_ = AuditLog(ctx, nil, "magic_link_requested", "user", userIDStr, map[string]any{"email": email})

	return &MagicLinkRequest{Token: token, UserName: user.Name, Email: email}, nil
}

// SendMagicLinkNotification posts the magic link payload to the configured
// webhook URL. It is intended to be called in a goroutine so the HTTP
// response is not delayed by the outbound webhook call.
func SendMagicLinkNotification(email, userName, loginLink string) {
	webhookURL := os.Getenv("MAGIC_LINK_WEBHOOK_URL")
	if webhookURL == "" {
		return
	}

	body, err := json.Marshal(map[string]string{
		"user_email": email,
		"user_name":  userName,
		"login_link": loginLink,
	})
	if err != nil {
		return
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", bytes.NewReader(body))

	if err != nil {
		return
	}
	resp.Body.Close()
}

// ConsumeMagicLink validates the raw token, marks the link as used and
// returns the associated user ID so the caller can establish a session.
func ConsumeMagicLink(ctx context.Context, token string) ([]byte, error) {
	if !MagicLinkEnabled() {
		return nil, errors.New("magic link sign-in is not enabled")
	}

	link, err := database.Connection.GetActiveMagicLinkByTokenHash(ctx, HashToken(token))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("invalid or expired sign-in link")
		}
		return nil, err
	}

	if err := database.Connection.ConsumeMagicLink(ctx, link.ID); err != nil {
		return nil, err
	}

	userIDStr, _ := id_helper.UnmarshalUUID(link.UserID)
	_ = AuditLog(ctx, &link.UserID, "magic_link_consumed", "user", userIDStr, map[string]any{})

	return link.UserID, nil
}
