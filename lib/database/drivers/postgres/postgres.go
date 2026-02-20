package postgres_driver

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

const DRIVER_NAME = "postgres"
const PASSWORD_FILE = ".pgpassword"

var embeddedDB *embeddedpostgres.EmbeddedPostgres

func Connect(ctx context.Context, migration_dir string) (*sql.DB, *migrate.Migrate, error) {
	// Check if DATABASE_URL is provided
	databaseURL := os.Getenv("DATABASE_URL")

	var connStr string

	if databaseURL != "" {
		// Use provided DATABASE_URL for external connection
		connStr = databaseURL
		log.Println("Connecting to external PostgreSQL database...")
	} else {
		// Use embedded PostgreSQL with local file storage
		var err error
		connStr, err = startEmbeddedPostgres()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to start embedded PostgreSQL: %w", err)
		}
		log.Println("Started embedded PostgreSQL with local storage in database/postgres_data")
	}

	db, err := sql.Open(DRIVER_NAME, connStr)
	if err != nil {
		return nil, nil, err
	}

	// Test the connection
	err = db.PingContext(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		log.Printf("Error creating postgres driver: %v", err)
		return nil, nil, err
	}

	m, err := migrate.NewWithDatabaseInstance(
		migration_dir,
		DRIVER_NAME, driver)
	if err != nil {
		log.Printf("Error initialising migration: %v", err)
		return nil, nil, err
	}

	return db, m, nil
}

func startEmbeddedPostgres() (string, error) {
	// Get absolute path for database directory
	dbDir, err := filepath.Abs("database/postgres_data")
	if err != nil {
		return "", fmt.Errorf("failed to resolve database directory: %w", err)
	}

	// Get or generate password (stored outside postgres_data to avoid PostgreSQL overwriting it)
	password, err := getOrCreatePassword()
	if err != nil {
		return "", fmt.Errorf("failed to get password: %w", err)
	}

	// Create database directory if it doesn't exist
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create database directory: %w", err)
	}

	// Configure embedded PostgreSQL
	embeddedDB = embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Username("postgres").
			Password(password).
			Database("reesource_tracker").
			Port(5433). // Use different port to avoid conflicts with system PostgreSQL instances
			DataPath(dbDir))

	// Start embedded PostgreSQL
	if err := embeddedDB.Start(); err != nil {
		return "", fmt.Errorf("failed to start embedded postgres: %w", err)
	}

	// Build connection string for embedded instance
	connStr := fmt.Sprintf("host=localhost port=5433 user=postgres password=%s dbname=reesource_tracker sslmode=disable",
		password)

	return connStr, nil
}

// getOrCreatePassword retrieves the stored password or generates a new one if it doesn't exist
// Password is stored in database/ directory (not in postgres_data/ to avoid PostgreSQL cleaning it)
func getOrCreatePassword() (string, error) {
	passwordPath := filepath.Join("database", PASSWORD_FILE)

	// Check if password file exists
	if data, err := os.ReadFile(passwordPath); err == nil {
		// Password file exists, validate and use it
		password := strings.TrimSpace(string(data))
		if len(password) == 0 {
			return "", fmt.Errorf("password file is empty")
		}
		log.Println("Using existing embedded PostgreSQL password")
		return password, nil
	}

	// Generate a new secure password
	password, err := generateRandomPassword(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate password: %w", err)
	}

	// Create file with restrictive permissions atomically
	f, err := os.OpenFile(passwordPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", fmt.Errorf("failed to create password file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(password); err != nil {
		return "", fmt.Errorf("failed to write password: %w", err)
	}

	log.Println("Generated new secure password for embedded PostgreSQL")
	return password, nil
}

// generateRandomPassword creates a cryptographically secure random password
func generateRandomPassword(length int) (string, error) {
	// Calculate bytes needed for desired base64 length
	// base64 encoding: 3 bytes -> 4 characters, so we need (length * 3 / 4) bytes
	byteCount := (length*3 + 3) / 4
	bytes := make([]byte, byteCount)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// Encode to base64 and take first 'length' characters
	encoded := base64.URLEncoding.EncodeToString(bytes)
	if len(encoded) > length {
		return encoded[:length], nil
	}
	return encoded, nil
}

// Stop stops the embedded PostgreSQL instance if running
func Stop() error {
	if embeddedDB != nil {
		return embeddedDB.Stop()
	}
	return nil
}
