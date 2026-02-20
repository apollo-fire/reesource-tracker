package database

import (
"context"
"log"
"os"
postgres_driver "reesource-tracker/lib/database/drivers/postgres"
)

var Connection *Queries

func Connect(ctx context.Context) error {
migration_dir := "file://database/migrations"
if _, err := os.Stat("migrations"); err == nil {
migration_dir = "file://migrations"
}

db, m, err := postgres_driver.Connect(ctx, migration_dir)
if err != nil {
log.Printf("Database connection error: %v", err)
return err
}
Connection = New(db)

err = m.Up()
if err != nil && err.Error() != "no change" {
log.Printf("Error applying migration: %v", err)
return err
}
return nil
}

// Cleanup stops the embedded database if running
func Cleanup() error {
return postgres_driver.Stop()
}
