package sqlite_driver

import (
	"context"
	"database/sql"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "modernc.org/sqlite"
)

const FOREIGN_KEY_PRAGMA = "PRAGMA foreign_keys = ON;"
const JOURNAL_MODE_PRAGMA = "PRAGMA journal_mode=WAL;"
const DRIVER_NAME = "sqlite"

func Connect(ctx context.Context, migration_dir string, database_location string) (*sql.DB, *migrate.Migrate, error) {
	db, err := sql.Open(DRIVER_NAME, database_location)
	if err != nil {
		return nil, nil, err
	}

	_, err = db.Exec(FOREIGN_KEY_PRAGMA + JOURNAL_MODE_PRAGMA)
	if err != nil {
		log.Fatal("Failed to enable foreign keys and WAL mode:", err)
		return nil, nil, err
	}

	driver, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		println("Error creating sqlite driver", err.Error())
		return nil, nil, err
	}
	m, err := migrate.NewWithDatabaseInstance(
		migration_dir,
		DRIVER_NAME, driver)
	if err != nil {
		println("Error initialising migration", err.Error())
		return nil, nil, err
	}
	println("Connected to sqlite database")

	return db, m, nil
}
