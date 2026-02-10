package worker

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
	workers map[string]Worker
}

func newMemStore() *memStore {
	return &memStore{workers: make(map[string]Worker)}
}

func (m *memStore) ListWorkers(_ context.Context) ([]Worker, error) {
	out := make([]Worker, 0, len(m.workers))
	for _, w := range m.workers {
		out = append(out, w)
	}
	return out, nil
}

func (m *memStore) CreateWorker(_ context.Context, w Worker) (Worker, error) {
	if _, exists := m.workers[w.ID]; exists {
		return Worker{}, fmt.Errorf("worker %q already exists", w.ID)
	}
	w.CreatedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	w.UpdatedAt = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	m.workers[w.ID] = w
	return w, nil
}

func (m *memStore) UpdateWorker(_ context.Context, w Worker) (Worker, error) {
	if _, exists := m.workers[w.ID]; !exists {
		return Worker{}, fmt.Errorf("worker %q not found", w.ID)
	}
	w.UpdatedAt = time.Date(2025, 1, 1, 0, 0, 1, 0, time.UTC)
	m.workers[w.ID] = w
	return w, nil
}

func (m *memStore) DeleteWorker(_ context.Context, id string) error {
	if _, exists := m.workers[id]; !exists {
		return fmt.Errorf("worker %q not found", id)
	}
	delete(m.workers, id)
	return nil
}

func (m *memStore) GetWorker(_ context.Context, id string) (Worker, error) {
	w, ok := m.workers[id]
	if !ok {
		return Worker{}, fmt.Errorf("worker %q not found", id)
	}
	return w, nil
}

// fakeRegistry records calls for verification.
type fakeRegistry struct {
	added   map[string]string // id -> url
	removed []string
}

func newFakeRegistry() *fakeRegistry {
	return &fakeRegistry{added: make(map[string]string)}
}

func (r *fakeRegistry) AddWorker(id string, rawURL string, _ string) error {
	r.added[id] = rawURL
	return nil
}

func (r *fakeRegistry) RemoveWorker(id string) {
	r.removed = append(r.removed, id)
}

func TestWorkerService(t *testing.T) {
	t.Run("create and list", func(t *testing.T) {
		store := newMemStore()
		reg := newFakeRegistry()
		svc := NewWorkerService(store, reg)
		ctx := context.Background()

		created, err := svc.CreateWorker(ctx, "my-worker", "My Worker", "http://localhost:8081", "secret")
		require.NoError(t, err)
		assert.Equal(t, "my-worker", created.ID)
		assert.Equal(t, "My Worker", created.Name)

		// Verify relay was updated.
		assert.Equal(t, "http://localhost:8081", reg.added[created.ID])

		workers, err := svc.ListWorkers(ctx)
		require.NoError(t, err)
		assert.Len(t, workers, 1)
	})

	t.Run("update", func(t *testing.T) {
		store := newMemStore()
		reg := newFakeRegistry()
		svc := NewWorkerService(store, reg)
		ctx := context.Background()

		created, err := svc.CreateWorker(ctx, "worker-1", "Worker", "http://localhost:8081", "s")
		require.NoError(t, err)

		updated, err := svc.UpdateWorker(ctx, created.ID, "Updated", "http://localhost:9090", "s2")
		require.NoError(t, err)
		assert.Equal(t, "Updated", updated.Name)
		assert.Equal(t, "http://localhost:9090", reg.added[created.ID])
	})

	t.Run("delete", func(t *testing.T) {
		store := newMemStore()
		reg := newFakeRegistry()
		svc := NewWorkerService(store, reg)
		ctx := context.Background()

		created, err := svc.CreateWorker(ctx, "worker-1", "Worker", "http://localhost:8081", "s")
		require.NoError(t, err)

		err = svc.DeleteWorker(ctx, created.ID)
		require.NoError(t, err)
		assert.Contains(t, reg.removed, created.ID)

		workers, err := svc.ListWorkers(ctx)
		require.NoError(t, err)
		assert.Empty(t, workers)
	})
}
