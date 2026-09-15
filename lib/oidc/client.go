package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Client wraps the OIDC provider and OAuth2 config. Initialise once at startup
// via Initialize; access the singleton with Get().
type Client struct {
	provider      *gooidc.Provider
	cfg           oauth2.Config
	verifier      *gooidc.IDTokenVerifier
	roleClaimPath string
	roleMap       map[string]string
}

var instance *Client

func Initialize(ctx context.Context, issuerURL, clientID, clientSecret, redirectURL, roleClaimPath string, roleMap map[string]string) error {
	provider, err := gooidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return fmt.Errorf("oidc provider discovery at %s: %w", issuerURL, err)
	}

	instance = &Client{
		provider: provider,
		cfg: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     provider.Endpoint(),
			// offline_access is required for refresh tokens; minimal scope set.
			Scopes: []string{gooidc.ScopeOpenID, "profile", "offline_access"},
		},
		verifier:      provider.Verifier(&gooidc.Config{ClientID: clientID}),
		roleClaimPath: roleClaimPath,
		roleMap:       roleMap,
	}
	return nil
}

func (c *Client) RoleClaimPath() string            { return c.roleClaimPath }
func (c *Client) RoleMap() map[string]string        { return c.roleMap }

func Get() *Client { return instance }

// TokenSet holds the results of a successful token exchange or refresh.
type TokenSet struct {
	RawIDToken        string
	AccessToken       string
	RefreshToken      string
	AccessTokenExpiry time.Time
	// OIDCSid is the provider session ID from the id_token "sid" claim, if present.
	OIDCSid   string
	Subject   string
	Name      string
	RawClaims json.RawMessage
}

func (c *Client) AuthCodeURL(state string) string {
	return c.cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (c *Client) Exchange(ctx context.Context, code string) (*TokenSet, error) {
	token, err := c.cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("code exchange: %w", err)
	}
	return c.tokenSetFrom(ctx, token)
}

func (c *Client) Refresh(ctx context.Context, refreshToken string) (*TokenSet, error) {
	src := c.cfg.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken})
	token, err := src.Token()
	if err != nil {
		return nil, fmt.Errorf("token refresh: %w", err)
	}
	return c.tokenSetFrom(ctx, token)
}

func (c *Client) tokenSetFrom(ctx context.Context, token *oauth2.Token) (*TokenSet, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no id_token in token response")
	}
	idToken, err := c.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("id_token verify: %w", err)
	}

	var claims struct {
		Name string `json:"name"`
		SID  string `json:"sid"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("id_token claims: %w", err)
	}

	var raw json.RawMessage
	if err := idToken.Claims(&raw); err != nil {
		return nil, fmt.Errorf("id_token raw claims: %w", err)
	}

	return &TokenSet{
		RawIDToken:        rawIDToken,
		AccessToken:       token.AccessToken,
		RefreshToken:      token.RefreshToken,
		AccessTokenExpiry: token.Expiry,
		OIDCSid:           claims.SID,
		Subject:           idToken.Subject,
		Name:              claims.Name,
		RawClaims:         raw,
	}, nil
}

// VerifyLogoutToken verifies an OIDC back-channel logout token (RFC OIDC-BACKCHANNEL)
// and returns the (sub, sid) claims. Returns an error if the token is invalid or
// violates spec requirements (nonce present, missing events claim).
func (c *Client) VerifyLogoutToken(ctx context.Context, tokenString string) (sub, sid string, err error) {
	idToken, err := c.verifier.Verify(ctx, tokenString)
	if err != nil {
		return "", "", fmt.Errorf("logout token verify: %w", err)
	}

	var claims struct {
		Nonce  string                     `json:"nonce"`
		Events map[string]json.RawMessage `json:"events"`
		SID    string                     `json:"sid"`
		Sub    string                     `json:"sub"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return "", "", fmt.Errorf("logout token claims: %w", err)
	}
	// Per spec: logout tokens must not contain a nonce (prevents replaying ID tokens).
	if claims.Nonce != "" {
		return "", "", fmt.Errorf("logout token must not contain nonce")
	}
	const backchannelEvent = "http://schemas.openid.net/event/backchannel-logout"
	if _, ok := claims.Events[backchannelEvent]; !ok {
		return "", "", fmt.Errorf("logout token missing backchannel-logout event")
	}
	if claims.Sub == "" && claims.SID == "" {
		return "", "", fmt.Errorf("logout token must contain sub or sid")
	}
	return claims.Sub, claims.SID, nil
}

// EndSessionURL returns the provider's end_session_endpoint with id_token_hint.
// Falls back to postLogoutRedirectURI if the provider does not advertise the endpoint.
func (c *Client) EndSessionURL(idTokenHint, postLogoutRedirectURI string) string {
	var d struct {
		EndSessionEndpoint string `json:"end_session_endpoint"`
	}
	_ = c.provider.Claims(&d)
	if d.EndSessionEndpoint == "" {
		return postLogoutRedirectURI
	}
	u := d.EndSessionEndpoint
	sep := "?"
	if idTokenHint != "" {
		u += sep + "id_token_hint=" + idTokenHint
		sep = "&"
	}
	if postLogoutRedirectURI != "" {
		u += sep + "post_logout_redirect_uri=" + url.QueryEscape(postLogoutRedirectURI)
	}
	return u
}
