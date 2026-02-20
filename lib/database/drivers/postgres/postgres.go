package postgres_driver

import (
"context"
"database/sql"
"fmt"
"log"
"os"
"path/filepath"

embeddedpostgres "github.com/fergusstrange/embedded-postgres"
"github.com/golang-migrate/migrate/v4"
"github.com/golang-migrate/migrate/v4/database/postgres"
_ "github.com/golang-migrate/migrate/v4/source/file"
_ "github.com/lib/pq"
)

const DRIVER_NAME = "postgres"

var embeddedDB *embeddedpostgres.EmbeddedPostgres

func Connect(ctx context.Context, migration_dir string) (*sql.DB, *migrate.Migrate, error) {
// Check if external PostgreSQL connection is specified
externalHost := os.Getenv("POSTGRES_HOST")

var connStr string

if externalHost != "" {
// Use external PostgreSQL connection
connStr = buildExternalConnectionString()
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

// Create database directory if it doesn't exist
if err := os.MkdirAll(dbDir, 0755); err != nil {
return "", fmt.Errorf("failed to create database directory: %w", err)
}

// Configure embedded PostgreSQL
// Don't specify RuntimePath - let it use temp directory
embeddedDB = embeddedpostgres.NewDatabase(
embeddedpostgres.DefaultConfig().
Username("postgres").
Password("postgres").
Database("reesource_tracker").
Port(5433). // Use different port to avoid conflicts
DataPath(dbDir))

// Start embedded PostgreSQL
if err := embeddedDB.Start(); err != nil {
return "", fmt.Errorf("failed to start embedded postgres: %w", err)
}

// Build connection string for embedded instance
connStr := "host=localhost port=5433 user=postgres password=postgres dbname=reesource_tracker sslmode=disable"
return connStr, nil
}

func buildExternalConnectionString() string {
host := os.Getenv("POSTGRES_HOST")

port := os.Getenv("POSTGRES_PORT")
if port == "" {
port = "5432"
}

user := os.Getenv("POSTGRES_USER")
if user == "" {
user = "postgres"
}

password := os.Getenv("POSTGRES_PASSWORD")
if password == "" {
log.Println("WARNING: POSTGRES_PASSWORD not set, using empty password")
password = ""
}

dbname := os.Getenv("POSTGRES_DB")
if dbname == "" {
dbname = "reesource_tracker"
}

sslmode := os.Getenv("POSTGRES_SSLMODE")
if sslmode == "" {
sslmode = "disable"
}

return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
host, port, user, password, dbname, sslmode)
}

// Stop stops the embedded PostgreSQL instance if running
func Stop() error {
if embeddedDB != nil {
return embeddedDB.Stop()
}
return nil
}
