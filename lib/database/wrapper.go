package database

import (
	"context"
	"os"
	sqlite_driver "reesource-tracker/lib/database/drivers/sqlite"
)

var Connection *Queries

const DATABASE_LOCATION = "database/db.sqlite"

func Connect(ctx context.Context) error {
	migration_dir := "file://database/migrations"
	if _, err := os.Stat("migrations"); err == nil {
		migration_dir = "file://migrations"
	}
	db, m, err := sqlite_driver.Connect(ctx, migration_dir, DATABASE_LOCATION)
	if err != nil {
		println("Got error", err.Error())
		return err
	}
	Connection = New(db)

	err = m.Up()
	if err != nil && err.Error() != "no change" {
		println("Error applying migration", err.Error())
		return err
	}
	return nil
}
