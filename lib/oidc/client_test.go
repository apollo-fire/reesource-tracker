package oidc

import (
	"testing"

	"golang.org/x/oauth2"
)

func TestRawIDTokenFromUsesResponseValue(t *testing.T) {
	token := (&oauth2.Token{}).WithExtra(map[string]any{"id_token": "new-id-token"})

	rawIDToken, err := rawIDTokenFrom(token, "existing-id-token")

	if err != nil {
		t.Fatalf("rawIDTokenFrom returned error: %v", err)
	}
	if rawIDToken != "new-id-token" {
		t.Fatalf("rawIDTokenFrom returned %q, want %q", rawIDToken, "new-id-token")
	}
}

func TestRawIDTokenFromUsesFallback(t *testing.T) {
	rawIDToken, err := rawIDTokenFrom(&oauth2.Token{}, "existing-id-token")

	if err != nil {
		t.Fatalf("rawIDTokenFrom returned error: %v", err)
	}
	if rawIDToken != "existing-id-token" {
		t.Fatalf("rawIDTokenFrom returned %q, want %q", rawIDToken, "existing-id-token")
	}
}

func TestRawIDTokenFromRequiresValue(t *testing.T) {
	rawIDToken, err := rawIDTokenFrom(&oauth2.Token{}, "")

	if err == nil {
		t.Fatalf("rawIDTokenFrom returned token %q without error", rawIDToken)
	}
}

func TestRefreshTokenFromUsesResponseValue(t *testing.T) {
	refreshToken := refreshTokenFrom(&oauth2.Token{RefreshToken: "new-refresh-token"}, "existing-refresh-token")

	if refreshToken != "new-refresh-token" {
		t.Fatalf("refreshTokenFrom returned %q, want %q", refreshToken, "new-refresh-token")
	}
}

func TestRefreshTokenFromUsesFallback(t *testing.T) {
	refreshToken := refreshTokenFrom(&oauth2.Token{}, "existing-refresh-token")

	if refreshToken != "existing-refresh-token" {
		t.Fatalf("refreshTokenFrom returned %q, want %q", refreshToken, "existing-refresh-token")
	}
}
