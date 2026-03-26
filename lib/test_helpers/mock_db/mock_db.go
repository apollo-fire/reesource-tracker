package mock_db

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"time"

	"reesource-tracker/lib/database"
	postgres_driver "reesource-tracker/lib/database/drivers/postgres"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var MockConnection *database.Queries
var MockDb *sql.DB
var MockPgContainer testcontainers.Container
var containerStarted bool
var migrationsURL string
var expectedConnStr string

func init() {
	err := os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	if err != nil {
		panic("unable to set environment variable: " + err.Error())
	}
	setupContainer()
	ResetMockDB()
}

func setupContainer() {
	if containerStarted {
		return
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
		relativePath += "../"
		dir = parent
	}

	migrationsURL = "file://" + relativePath + "database/migrations"

	ctx := context.Background()
	pgContainer, err := postgres.Run(ctx,
		"postgres:15.3-alpine",
		postgres.WithDatabase("test-db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(2*time.Minute),
		),
	)
	if err != nil {
		panic(err)
	}
	MockPgContainer = pgContainer

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic(err)
	}

	_ = os.Setenv("DATABASE_URL", connStr)
	expectedConnStr = connStr
	containerStarted = true
}

func closeDBConnection() {
	if MockConnection != nil {
		MockConnection = nil
	}
	if MockDb != nil {
		_ = MockDb.Close()
		MockDb = nil
	}
}

func ResetMockDB() {
	setupContainer()

	// Safety check: ensure DATABASE_URL matches the test container
	if os.Getenv("DATABASE_URL") != expectedConnStr {
		panic("DATABASE_URL does not match test container connection string")
	}

	// Close existing DB connection
	closeDBConnection()

	// Drop all tables by dropping and recreating the public schema
	ctx := context.Background()
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.PingContext(ctx)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec(`DO $$ DECLARE
		r RECORD;
	BEGIN
		FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = current_schema()) LOOP
			EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
		END LOOP;
	END $$;`)
	if err != nil {
		panic(err)
	}

	// Reconnect and perform migrations
	db, m, err := postgres_driver.Connect(ctx, migrationsURL)
	if err != nil {
		panic(err)
	}
	MockDb = db
	MockConnection = database.New(MockDb)

	err = m.Up()
	if err != nil && err.Error() != "no change" {
		panic(err)
	}
}
