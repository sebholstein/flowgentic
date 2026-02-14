package thread

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memStore is an in-memory Store for testing.
type memStore struct {
	threads map[string]Thread
}

func newMemStore() *memStore {
	return &memStore{threads: make(map[string]Thread)}
}

func (m *memStore) ListThreads(_ context.Context, projectID string) ([]Thread, error) {
	var out []Thread
	for _, t := range m.threads {
		if t.ProjectID == projectID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (m *memStore) GetThread(_ context.Context, id string) (Thread, error) {
	t, ok := m.threads[id]
	if !ok {
		return Thread{}, fmt.Errorf("thread %q not found", id)
	}
	return t, nil
}

func (m *memStore) CreateThread(_ context.Context, t Thread) (Thread, error) {
	if _, exists := m.threads[t.ID]; exists {
		return Thread{}, fmt.Errorf("thread %q already exists", t.ID)
	}
	t.CreatedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t.UpdatedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	m.threads[t.ID] = t
	return t, nil
}

func (m *memStore) UpdateThreadTopic(_ context.Context, id, topic string) error {
	t, ok := m.threads[id]
	if !ok {
		return fmt.Errorf("thread %q not found", id)
	}
	t.Topic = topic
	m.threads[id] = t
	return nil
}

func (m *memStore) UpdateThreadPlan(_ context.Context, id, plan string) error {
	t, ok := m.threads[id]
	if !ok {
		return fmt.Errorf("thread %q not found", id)
	}
	t.Plan = plan
	m.threads[id] = t
	return nil
}

func (m *memStore) UpdateThreadArchived(_ context.Context, id string, archived bool) error {
	t, ok := m.threads[id]
	if !ok {
		return fmt.Errorf("thread %q not found", id)
	}
	t.Archived = archived
	m.threads[id] = t
	return nil
}

func (m *memStore) DeleteThread(_ context.Context, id string) error {
	if _, exists := m.threads[id]; !exists {
		return fmt.Errorf("thread %q not found", id)
	}
	delete(m.threads, id)
	return nil
}

func TestThreadService(t *testing.T) {
	t.Run("create and list", func(t *testing.T) {
		store := newMemStore()
		svc := NewThreadService(store)
		ctx := context.Background()

		created, err := svc.CreateThread(ctx, Thread{
			ProjectID: "my-project",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, created.ID)
		assert.Equal(t, "my-project", created.ProjectID)

		threads, err := svc.ListThreads(ctx, "my-project")
		require.NoError(t, err)
		assert.Len(t, threads, 1)
	})

	t.Run("create rejects empty project id", func(t *testing.T) {
		store := newMemStore()
		svc := NewThreadService(store)
		ctx := context.Background()

		_, err := svc.CreateThread(ctx, Thread{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project_id is required")
	})

	t.Run("list requires project id", func(t *testing.T) {
		store := newMemStore()
		svc := NewThreadService(store)
		ctx := context.Background()

		_, err := svc.ListThreads(ctx, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project_id is required")
	})

	t.Run("delete", func(t *testing.T) {
		store := newMemStore()
		svc := NewThreadService(store)
		ctx := context.Background()

		created, err := svc.CreateThread(ctx, Thread{
			ProjectID: "my-project",
		})
		require.NoError(t, err)

		err = svc.DeleteThread(ctx, created.ID)
		require.NoError(t, err)

		threads, err := svc.ListThreads(ctx, "my-project")
		require.NoError(t, err)
		assert.Empty(t, threads)
	})

	t.Run("archive and unarchive", func(t *testing.T) {
		store := newMemStore()
		svc := NewThreadService(store)
		ctx := context.Background()

		created, err := svc.CreateThread(ctx, Thread{
			ProjectID: "my-project",
		})
		require.NoError(t, err)
		assert.False(t, created.Archived)

		archived, err := svc.ArchiveThread(ctx, created.ID, true)
		require.NoError(t, err)
		assert.True(t, archived.Archived)

		unarchived, err := svc.ArchiveThread(ctx, created.ID, false)
		require.NoError(t, err)
		assert.False(t, unarchived.Archived)
	})

	t.Run("archive rejects empty id", func(t *testing.T) {
		store := newMemStore()
		svc := NewThreadService(store)
		ctx := context.Background()

		_, err := svc.ArchiveThread(ctx, "", true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "thread id is required")
	})

	t.Run("get", func(t *testing.T) {
		store := newMemStore()
		svc := NewThreadService(store)
		ctx := context.Background()

		created, err := svc.CreateThread(ctx, Thread{
			ProjectID: "my-project",
		})
		require.NoError(t, err)

		got, err := svc.GetThread(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
	})
}
