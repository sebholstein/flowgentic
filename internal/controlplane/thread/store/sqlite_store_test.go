package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/sebastianm/flowgentic/internal/controlplane/project"
	projectstore "github.com/sebastianm/flowgentic/internal/controlplane/project/store"
	"github.com/sebastianm/flowgentic/internal/controlplane/thread"
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

	// Create a parent project for FK constraint.
	ps := projectstore.NewSQLiteStore(db)
	_, err = ps.CreateProject(context.Background(), project.Project{
		ID:   "test-project",
		Name: "Test Project",
	})
	require.NoError(t, err)

	return NewSQLiteStore(db)
}

func TestSQLiteStore(t *testing.T) {
	t.Run("create and list", func(t *testing.T) {
		s := newTestStore(t)
		ctx := context.Background()

		th := thread.Thread{ID: "t1", ProjectID: "test-project"}
		created, err := s.CreateThread(ctx, th)
		require.NoError(t, err)
		assert.Equal(t, "t1", created.ID)
		assert.Equal(t, "test-project", created.ProjectID)
		assert.NotEmpty(t, created.CreatedAt)

		threads, err := s.ListThreads(ctx, "test-project")
		require.NoError(t, err)
		assert.Len(t, threads, 1)
		assert.Equal(t, "t1", threads[0].ID)
	})

	t.Run("list filters by project", func(t *testing.T) {
		s := newTestStore(t)
		ctx := context.Background()

		_, err := s.CreateThread(ctx, thread.Thread{ID: "t1", ProjectID: "test-project"})
		require.NoError(t, err)

		threads, err := s.ListThreads(ctx, "other-project")
		require.NoError(t, err)
		assert.Empty(t, threads)
	})

	t.Run("delete", func(t *testing.T) {
		s := newTestStore(t)
		ctx := context.Background()

		_, err := s.CreateThread(ctx, thread.Thread{ID: "t1", ProjectID: "test-project"})
		require.NoError(t, err)

		err = s.DeleteThread(ctx, "t1")
		require.NoError(t, err)

		threads, err := s.ListThreads(ctx, "test-project")
		require.NoError(t, err)
		assert.Empty(t, threads)
	})

	t.Run("delete not found", func(t *testing.T) {
		s := newTestStore(t)
		ctx := context.Background()

		err := s.DeleteThread(ctx, "nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("get", func(t *testing.T) {
		s := newTestStore(t)
		ctx := context.Background()

		_, err := s.CreateThread(ctx, thread.Thread{ID: "t1", ProjectID: "test-project"})
		require.NoError(t, err)

		got, err := s.GetThread(ctx, "t1")
		require.NoError(t, err)
		assert.Equal(t, "t1", got.ID)
	})
}
