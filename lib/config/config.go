package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	AuditRetentionDays int `json:"audit_retention_days" yaml:"audit_retention_days"`
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
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		err = json.Unmarshal(raw, &cfg)
	default:
		err = yaml.Unmarshal(raw, &cfg)
	}
	if err != nil {
		return AppConfig{}, err
	}
	if cfg.AuditRetentionDays <= 0 {
		return AppConfig{}, errors.New("audit_retention_days must be > 0")
	}
	return cfg, nil
}
