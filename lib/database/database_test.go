package database_test

import (
	"reesource-tracker/lib/test_helpers/mock_db"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrations(t *testing.T) {
	require.NotPanics(t, mock_db.ResetMockDB, "Running migrations causes an error")
}
