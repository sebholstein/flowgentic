package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/sebastianm/flowgentic/internal/controlplane/worker"
	"github.com/sebastianm/flowgentic/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := database.Open(context.Background(), dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return NewSQLiteStore(db)
}

func TestSQLiteStore(t *testing.T) {
	t.Run("create and list", func(t *testing.T) {
		s := newTestStore(t)
		ctx := context.Background()

		w := worker.Worker{ID: "w1", Name: "Worker 1", URL: "http://localhost:8081", Secret: "s1"}
		created, err := s.CreateWorker(ctx, w)
		require.NoError(t, err)
		assert.Equal(t, "w1", created.ID)
		assert.Equal(t, "Worker 1", created.Name)
		assert.NotEmpty(t, created.CreatedAt)

		workers, err := s.ListWorkers(ctx)
		require.NoError(t, err)
		assert.Len(t, workers, 1)
		assert.Equal(t, "w1", workers[0].ID)
	})

	t.Run("update", func(t *testing.T) {
		s := newTestStore(t)
		ctx := context.Background()

		w := worker.Worker{ID: "w1", Name: "Worker 1", URL: "http://localhost:8081", Secret: "s1"}
		_, err := s.CreateWorker(ctx, w)
		require.NoError(t, err)

		w.Name = "Updated Worker"
		w.URL = "http://localhost:9090"
		updated, err := s.UpdateWorker(ctx, w)
		require.NoError(t, err)
		assert.Equal(t, "Updated Worker", updated.Name)
		assert.Equal(t, "http://localhost:9090", updated.URL)
	})

	t.Run("update not found", func(t *testing.T) {
		s := newTestStore(t)
		ctx := context.Background()

		w := worker.Worker{ID: "nonexistent", Name: "X", URL: "http://x", Secret: "x"}
		_, err := s.UpdateWorker(ctx, w)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("delete", func(t *testing.T) {
		s := newTestStore(t)
		ctx := context.Background()

		w := worker.Worker{ID: "w1", Name: "Worker 1", URL: "http://localhost:8081", Secret: "s1"}
		_, err := s.CreateWorker(ctx, w)
		require.NoError(t, err)

		err = s.DeleteWorker(ctx, "w1")
		require.NoError(t, err)

		workers, err := s.ListWorkers(ctx)
		require.NoError(t, err)
		assert.Empty(t, workers)
	})

	t.Run("delete not found", func(t *testing.T) {
		s := newTestStore(t)
		ctx := context.Background()

		err := s.DeleteWorker(ctx, "nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("get", func(t *testing.T) {
		s := newTestStore(t)
		ctx := context.Background()

		w := worker.Worker{ID: "w1", Name: "Worker 1", URL: "http://localhost:8081", Secret: "s1"}
		_, err := s.CreateWorker(ctx, w)
		require.NoError(t, err)

		got, err := s.GetWorker(ctx, "w1")
		require.NoError(t, err)
		assert.Equal(t, "Worker 1", got.Name)
	})
}
