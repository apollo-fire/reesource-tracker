package postgres_driver

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

const DRIVER_NAME = "postgres"
const PASSWORD_FILE = ".pgpassword"

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
		log.Fatal("ERROR: No database provided")
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
