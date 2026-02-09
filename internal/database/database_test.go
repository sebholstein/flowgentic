package database

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen(t *testing.T) {
	t.Run("creates tables via migration", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "test.db")

		db, err := Open(context.Background(), dbPath)
		require.NoError(t, err)
		defer db.Close()

		row := db.QueryRow("SELECT COUNT(*) FROM workers")
		var count int
		require.NoError(t, row.Scan(&count))
		assert.Equal(t, 0, count)
	})

	t.Run("enables WAL journal mode", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "test.db")

		db, err := Open(context.Background(), dbPath)
		require.NoError(t, err)
		defer db.Close()

		var mode string
		require.NoError(t, db.QueryRow("PRAGMA journal_mode").Scan(&mode))
		assert.Equal(t, "wal", mode)
	})

	t.Run("creates parent directories", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "subdir", "nested", "test.db")

		db, err := Open(context.Background(), dbPath)
		require.NoError(t, err)
		defer db.Close()
	})

	t.Run("idempotent migrations", func(t *testing.T) {
		dbPath := filepath.Join(t.TempDir(), "test.db")

		db1, err := Open(context.Background(), dbPath)
		require.NoError(t, err)
		db1.Close()

		// Opening again should not fail (migrations already applied).
		db2, err := Open(context.Background(), dbPath)
		require.NoError(t, err)
		db2.Close()
	})
}
