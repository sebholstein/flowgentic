package project

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memStore is an in-memory Store for testing.
type memStore struct {
	projects map[string]Project
}

func newMemStore() *memStore {
	return &memStore{projects: make(map[string]Project)}
}

func (m *memStore) ListProjects(_ context.Context) ([]Project, error) {
	out := make([]Project, 0, len(m.projects))
	for _, p := range m.projects {
		out = append(out, p)
	}
	return out, nil
}

func (m *memStore) GetProject(_ context.Context, id string) (Project, error) {
	p, ok := m.projects[id]
	if !ok {
		return Project{}, fmt.Errorf("project %q not found", id)
	}
	return p, nil
}

func (m *memStore) CreateProject(_ context.Context, p Project) (Project, error) {
	if _, exists := m.projects[p.ID]; exists {
		return Project{}, fmt.Errorf("project %q already exists", p.ID)
	}
	p.CreatedAt = "2025-01-01T00:00:00.000Z"
	p.UpdatedAt = "2025-01-01T00:00:00.000Z"
	m.projects[p.ID] = p
	return p, nil
}

func (m *memStore) UpdateProject(_ context.Context, p Project) (Project, error) {
	if _, exists := m.projects[p.ID]; !exists {
		return Project{}, fmt.Errorf("project %q not found", p.ID)
	}
	p.UpdatedAt = "2025-01-01T00:00:01.000Z"
	m.projects[p.ID] = p
	return p, nil
}

func (m *memStore) ReorderProjects(_ context.Context, entries []SortEntry) error {
	for _, e := range entries {
		p, ok := m.projects[e.ID]
		if !ok {
			return fmt.Errorf("project %q not found", e.ID)
		}
		p.SortIndex = e.SortIndex
		m.projects[e.ID] = p
	}
	return nil
}

func (m *memStore) DeleteProject(_ context.Context, id string) error {
	if _, exists := m.projects[id]; !exists {
		return fmt.Errorf("project %q not found", id)
	}
	delete(m.projects, id)
	return nil
}

func TestProjectService(t *testing.T) {
	t.Run("create and list", func(t *testing.T) {
		store := newMemStore()
		svc := NewProjectService(store)
		ctx := context.Background()

		created, err := svc.CreateProject(ctx, Project{
			ID:   "my-project",
			Name: "My Project",
		})
		require.NoError(t, err)
		assert.Equal(t, "my-project", created.ID)
		assert.Equal(t, "My Project", created.Name)

		projects, err := svc.ListProjects(ctx)
		require.NoError(t, err)
		assert.Len(t, projects, 1)
	})

	t.Run("create with worker paths", func(t *testing.T) {
		store := newMemStore()
		svc := NewProjectService(store)
		ctx := context.Background()

		created, err := svc.CreateProject(ctx, Project{
			ID:          "my-project",
			Name:        "My Project",
			WorkerPaths: map[string]string{"worker-1": "/path/to/worker"},
		})
		require.NoError(t, err)
		assert.Equal(t, "/path/to/worker", created.WorkerPaths["worker-1"])
	})

	t.Run("create rejects invalid id", func(t *testing.T) {
		store := newMemStore()
		svc := NewProjectService(store)
		ctx := context.Background()

		_, err := svc.CreateProject(ctx, Project{
			ID:   "Invalid-ID",
			Name: "Test",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid project id")
	})

	t.Run("create rejects empty name", func(t *testing.T) {
		store := newMemStore()
		svc := NewProjectService(store)
		ctx := context.Background()

		_, err := svc.CreateProject(ctx, Project{
			ID:   "my-project",
			Name: "",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name is required")
	})

	t.Run("create rejects invalid worker name", func(t *testing.T) {
		store := newMemStore()
		svc := NewProjectService(store)
		ctx := context.Background()

		_, err := svc.CreateProject(ctx, Project{
			ID:          "my-project",
			Name:        "Test",
			WorkerPaths: map[string]string{"Invalid Worker": "/path"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid worker name")
	})

	t.Run("update", func(t *testing.T) {
		store := newMemStore()
		svc := NewProjectService(store)
		ctx := context.Background()

		_, err := svc.CreateProject(ctx, Project{
			ID:   "my-project",
			Name: "Original",
		})
		require.NoError(t, err)

		updated, err := svc.UpdateProject(ctx, Project{
			ID:   "my-project",
			Name: "Updated",
		})
		require.NoError(t, err)
		assert.Equal(t, "Updated", updated.Name)
	})

	t.Run("delete", func(t *testing.T) {
		store := newMemStore()
		svc := NewProjectService(store)
		ctx := context.Background()

		_, err := svc.CreateProject(ctx, Project{
			ID:   "my-project",
			Name: "Test",
		})
		require.NoError(t, err)

		err = svc.DeleteProject(ctx, "my-project")
		require.NoError(t, err)

		projects, err := svc.ListProjects(ctx)
		require.NoError(t, err)
		assert.Empty(t, projects)
	})

	t.Run("get", func(t *testing.T) {
		store := newMemStore()
		svc := NewProjectService(store)
		ctx := context.Background()

		_, err := svc.CreateProject(ctx, Project{
			ID:   "my-project",
			Name: "Test",
		})
		require.NoError(t, err)

		got, err := svc.GetProject(ctx, "my-project")
		require.NoError(t, err)
		assert.Equal(t, "Test", got.Name)
	})
}

func TestValidateResourceName(t *testing.T) {
	t.Run("valid names", func(t *testing.T) {
		for _, name := range []string{"a", "abc", "my-project", "project-123", "a1"} {
			assert.NoError(t, ValidateResourceName(name), "expected %q to be valid", name)
		}
	})

	t.Run("invalid names", func(t *testing.T) {
		for _, name := range []string{"", "A", "-abc", "abc-", "123", "my project", "my_project"} {
			assert.Error(t, ValidateResourceName(name), "expected %q to be invalid", name)
		}
	})
}
