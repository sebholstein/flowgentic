package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/sebastianm/flowgentic/internal/controlplane/project"
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

		p := project.Project{ID: "p1", Name: "Project 1"}
		created, err := s.CreateProject(ctx, p)
		require.NoError(t, err)
		assert.Equal(t, "p1", created.ID)
		assert.Equal(t, "Project 1", created.Name)
		assert.NotEmpty(t, created.CreatedAt)

		projects, err := s.ListProjects(ctx)
		require.NoError(t, err)
		assert.Len(t, projects, 1)
		assert.Equal(t, "p1", projects[0].ID)
	})

	t.Run("create with worker paths", func(t *testing.T) {
		s := newTestStore(t)
		ctx := context.Background()

		p := project.Project{
			ID:          "p1",
			Name:        "Project 1",
			WorkerPaths: map[string]string{"worker-1": "/path/to/worker"},
		}
		created, err := s.CreateProject(ctx, p)
		require.NoError(t, err)
		assert.Equal(t, "/path/to/worker", created.WorkerPaths["worker-1"])

		got, err := s.GetProject(ctx, "p1")
		require.NoError(t, err)
		assert.Equal(t, "/path/to/worker", got.WorkerPaths["worker-1"])
	})

	t.Run("update", func(t *testing.T) {
		s := newTestStore(t)
		ctx := context.Background()

		p := project.Project{ID: "p1", Name: "Project 1"}
		_, err := s.CreateProject(ctx, p)
		require.NoError(t, err)

		p.Name = "Updated Project"
		p.DefaultPlannerAgent = "claude"
		updated, err := s.UpdateProject(ctx, p)
		require.NoError(t, err)
		assert.Equal(t, "Updated Project", updated.Name)
		assert.Equal(t, "claude", updated.DefaultPlannerAgent)
	})

	t.Run("update not found", func(t *testing.T) {
		s := newTestStore(t)
		ctx := context.Background()

		p := project.Project{ID: "nonexistent", Name: "X"}
		_, err := s.UpdateProject(ctx, p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("delete", func(t *testing.T) {
		s := newTestStore(t)
		ctx := context.Background()

		p := project.Project{ID: "p1", Name: "Project 1"}
		_, err := s.CreateProject(ctx, p)
		require.NoError(t, err)

		err = s.DeleteProject(ctx, "p1")
		require.NoError(t, err)

		projects, err := s.ListProjects(ctx)
		require.NoError(t, err)
		assert.Empty(t, projects)
	})

	t.Run("delete not found", func(t *testing.T) {
		s := newTestStore(t)
		ctx := context.Background()

		err := s.DeleteProject(ctx, "nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("get", func(t *testing.T) {
		s := newTestStore(t)
		ctx := context.Background()

		p := project.Project{ID: "p1", Name: "Project 1"}
		_, err := s.CreateProject(ctx, p)
		require.NoError(t, err)

		got, err := s.GetProject(ctx, "p1")
		require.NoError(t, err)
		assert.Equal(t, "Project 1", got.Name)
	})
}
