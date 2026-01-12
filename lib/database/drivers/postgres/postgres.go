package postgres_driver

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

const DRIVER_NAME = "postgres"

func Connect(ctx context.Context, migration_dir string) (*sql.DB, *migrate.Migrate, error) {
	// Get PostgreSQL connection parameters from environment variables
	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		host = "localhost"
	}
	
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
		password = "postgres"
	}
	
	dbname := os.Getenv("POSTGRES_DB")
	if dbname == "" {
		dbname = "reesource_tracker"
	}
	
	sslmode := os.Getenv("POSTGRES_SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}
	
	// Build connection string
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode)
	
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
		println("Error creating postgres driver", err.Error())
		return nil, nil, err
	}
	
	m, err := migrate.NewWithDatabaseInstance(
		migration_dir,
		DRIVER_NAME, driver)
	if err != nil {
		println("Error initialising migration", err.Error())
		return nil, nil, err
	}
	println("Connected to PostgreSQL database")
	
	return db, m, nil
}
