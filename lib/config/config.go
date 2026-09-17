package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type OIDCConfig struct {
	IssuerURL    string `json:"issuer_url"    yaml:"issuer_url"`
	ClientID     string `json:"client_id"     yaml:"client_id"`
	ClientSecret string `json:"client_secret" yaml:"client_secret"`
	// Dot-notation path into ID-token claims that holds roles/groups.
	// Keycloak examples: "groups", "realm_access.roles"
	RoleClaimPath string `json:"role_claim_path" yaml:"role_claim_path"`
	// Maps OIDC claim values to app roles (admin / maintainer / user).
	// If empty, claim values are matched directly against role names.
	RoleMap map[string]string `json:"role_map" yaml:"role_map"`
}

type AppConfig struct {
	// BaseURL is the externally-reachable root of this app (no trailing slash).
	BaseURL string     `json:"base_url" yaml:"base_url"`
	OIDC    OIDCConfig `json:"oidc"     yaml:"oidc"`
}

func Load() (AppConfig, error) {
	path := os.Getenv("APP_CONFIG_PATH")
	if path == "" {
		path = "./config/app.yaml"
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return AppConfig{}, err
	}

	var cfg AppConfig
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		err = json.Unmarshal(raw, &cfg)
	default:
		err = yaml.Unmarshal(raw, &cfg)
	}
	if err != nil {
		return AppConfig{}, err
	}

	if cfg.BaseURL == "" {
		return AppConfig{}, errors.New("base_url is required")
	}
	if cfg.OIDC.IssuerURL == "" {
		return AppConfig{}, errors.New("oidc.issuer_url is required")
	}
	if cfg.OIDC.ClientID == "" {
		return AppConfig{}, errors.New("oidc.client_id is required")
	}
	if cfg.OIDC.RoleClaimPath == "" {
		cfg.OIDC.RoleClaimPath = "groups"
	}
	return cfg, nil
}
