package database

import (
	"context"
	"database/sql"
	"os"
	postgres_driver "reesource-tracker/lib/database/drivers/postgres"
	sqlite_driver "reesource-tracker/lib/database/drivers/sqlite"

	"github.com/golang-migrate/migrate/v4"
)

var Connection *Queries

func Connect(ctx context.Context) {
	migration_dir := "file://database/migrations"
	if _, err := os.Stat("migrations"); err == nil {
		migration_dir = "file://migrations"
	}
	
	// Determine which database driver to use based on DB_DRIVER environment variable
	dbDriver := os.Getenv("DB_DRIVER")
	if dbDriver == "" {
		dbDriver = "sqlite" // Default to SQLite for backwards compatibility
	}
	
	var db *sql.DB
	var m *migrate.Migrate
	var err error
	
	switch dbDriver {
	case "postgres", "postgresql":
		db, m, err = postgres_driver.Connect(ctx, migration_dir)
	case "sqlite":
		db, m, err = sqlite_driver.Connect(ctx, migration_dir)
	default:
		println("Unknown DB_DRIVER:", dbDriver, "- defaulting to sqlite")
		db, m, err = sqlite_driver.Connect(ctx, migration_dir)
	}
	
	if err != nil {
		println("Got error", err.Error())
		return
	}
	Connection = New(db)

	err = m.Up()
	if err != nil && err.Error() != "no change" {
		println("Error applying migration", err.Error())
		return
	}
}
