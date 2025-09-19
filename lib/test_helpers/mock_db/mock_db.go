package mock_db

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"reesource-tracker/lib/database"
	sqlite_driver "reesource-tracker/lib/database/drivers/sqlite"
)

var MockConnection *database.Queries
var MockDb *sql.DB

func init() {
	ResetMockDB()
}

func ResetMockDB() {
	if MockConnection != nil {
		MockConnection = nil
	}
	if MockDb != nil {
		MockDb.Close()
		MockDb = nil
	}

	// Find project root by searching for go.mod upwards
	dir, err := os.Getwd()
	if err != nil {
		panic("unable to get working directory: " + err.Error())
	}
	relativePath := ""
	for {
		candidate := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(candidate); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("could not find go.mod in any parent directory")
		}
		relativePath = relativePath + "../"
		dir = parent
	}

	migrationsURL := "file://" + relativePath + "database/migrations"
	MockDb, m, err := sqlite_driver.Connect(context.Background(), migrationsURL, ":memory:")
	if err != nil {
		panic(err)
	}
	MockConnection = database.New(MockDb)
	err = m.Up()
	if err != nil {
		panic(err)
	}
}
