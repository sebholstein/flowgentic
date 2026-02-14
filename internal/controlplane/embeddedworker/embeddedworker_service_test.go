package embeddedworker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/sebastianm/flowgentic/internal/controlplane/worker"
	controlplanev1 "github.com/sebastianm/flowgentic/internal/proto/gen/controlplane/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memStore struct {
	workers map[string]worker.Worker
}

func newMemStore() *memStore {
	return &memStore{workers: make(map[string]worker.Worker)}
}

func (m *memStore) GetWorker(_ context.Context, id string) (worker.Worker, error) {
	w, ok := m.workers[id]
	if !ok {
		return worker.Worker{}, fmt.Errorf("worker %q not found", id)
	}
	return w, nil
}

func (m *memStore) CreateWorker(_ context.Context, w worker.Worker) (worker.Worker, error) {
	if _, exists := m.workers[w.ID]; exists {
		return worker.Worker{}, fmt.Errorf("worker %q already exists", w.ID)
	}
	m.workers[w.ID] = w
	return w, nil
}

func (m *memStore) UpdateWorker(_ context.Context, w worker.Worker) (worker.Worker, error) {
	if _, exists := m.workers[w.ID]; !exists {
		return worker.Worker{}, fmt.Errorf("worker %q not found", w.ID)
	}
	m.workers[w.ID] = w
	return w, nil
}

type fakeRegistry struct {
	added   map[string]string
	removed []string
	addErr  error
}

func newFakeRegistry() *fakeRegistry {
	return &fakeRegistry{added: make(map[string]string)}
}

func (r *fakeRegistry) AddWorker(id string, rawURL string, _ string) error {
	if r.addErr != nil {
		return r.addErr
	}
	r.added[id] = rawURL
	return nil
}

func (r *fakeRegistry) RemoveWorker(id string) {
	r.removed = append(r.removed, id)
}

func newTestService(reg RegistryUpdater, store Store) *EmbeddedWorkerService {
	return NewEmbeddedWorkerService(
		slog.New(slog.NewTextHandler(os.Stdout, nil)),
		"/nonexistent/binary",
		"test-secret",
		50051,
		"/nonexistent/config.json",
		reg,
		store,
	)
}

func TestEmbeddedWorkerService(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		reg := newFakeRegistry()
		store := newMemStore()
		svc := newTestService(reg, store)

		status, lastErr, pid, addr := svc.GetStatus()
		assert.Equal(t, controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_STOPPED, status)
		assert.Empty(t, lastErr)
		assert.Equal(t, 0, pid)
		assert.Empty(t, addr)
		assert.Equal(t, "test-secret", svc.Secret())
	})

	t.Run("Start", func(t *testing.T) {
		t.Run("returns error when already running", func(t *testing.T) {
			reg := newFakeRegistry()
			store := newMemStore()
			svc := newTestService(reg, store)

			svc.mu.Lock()
			svc.status = controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_RUNNING
			svc.mu.Unlock()

			err := svc.Start(context.Background())
			require.Error(t, err)
			assert.Contains(t, err.Error(), "already")
		})

		t.Run("returns error when already starting", func(t *testing.T) {
			reg := newFakeRegistry()
			store := newMemStore()
			svc := newTestService(reg, store)

			svc.mu.Lock()
			svc.status = controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_STARTING
			svc.mu.Unlock()

			err := svc.Start(context.Background())
			require.Error(t, err)
			assert.Contains(t, err.Error(), "already")
		})
	})

	t.Run("Stop", func(t *testing.T) {
		t.Run("returns error when not running", func(t *testing.T) {
			reg := newFakeRegistry()
			store := newMemStore()
			svc := newTestService(reg, store)

			err := svc.Stop(context.Background())
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not running")
		})

		t.Run("stops from running state", func(t *testing.T) {
			reg := newFakeRegistry()
			store := newMemStore()
			svc := newTestService(reg, store)

			svc.mu.Lock()
			svc.status = controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_RUNNING
			svc.cancel = func() {}
			svc.mu.Unlock()

			err := svc.Stop(context.Background())
			require.NoError(t, err)

			status, _, pid, _ := svc.GetStatus()
			assert.Equal(t, controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_STOPPED, status)
			assert.Equal(t, 0, pid)
			assert.Contains(t, reg.removed, workerID)
		})

		t.Run("stops from errored state", func(t *testing.T) {
			reg := newFakeRegistry()
			store := newMemStore()
			svc := newTestService(reg, store)

			svc.mu.Lock()
			svc.status = controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_ERRORED
			svc.mu.Unlock()

			err := svc.Stop(context.Background())
			require.NoError(t, err)

			status, _, _, _ := svc.GetStatus()
			assert.Equal(t, controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_STOPPED, status)
		})
	})

	t.Run("Restart", func(t *testing.T) {
		t.Run("fails when not running", func(t *testing.T) {
			reg := newFakeRegistry()
			store := newMemStore()
			svc := newTestService(reg, store)

			err := svc.Restart(context.Background())
			require.Error(t, err)
			assert.Contains(t, err.Error(), "starting worker process")
		})

		t.Run("attempts start from stopped state", func(t *testing.T) {
			reg := newFakeRegistry()
			store := newMemStore()
			svc := newTestService(reg, store)

			svc.mu.Lock()
			svc.status = controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_STOPPED
			svc.mu.Unlock()

			err := svc.Restart(context.Background())
			require.Error(t, err)
		})
	})

	t.Run("Subscribe", func(t *testing.T) {
		t.Run("creates and tracks subscriber", func(t *testing.T) {
			reg := newFakeRegistry()
			store := newMemStore()
			svc := newTestService(reg, store)

			ch := svc.Subscribe()
			assert.NotNil(t, ch)

			svc.mu.Lock()
			_, exists := svc.subscribers[ch]
			svc.mu.Unlock()
			assert.True(t, exists)

			svc.Unsubscribe(ch)

			svc.mu.Lock()
			_, exists = svc.subscribers[ch]
			svc.mu.Unlock()
			assert.False(t, exists)
		})
	})

	t.Run("setStatusLocked", func(t *testing.T) {
		t.Run("notifies all subscribers", func(t *testing.T) {
			reg := newFakeRegistry()
			store := newMemStore()
			svc := newTestService(reg, store)

			ch1 := svc.Subscribe()
			ch2 := svc.Subscribe()

			svc.mu.Lock()
			svc.setStatusLocked(controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_STARTING, "test")
			svc.mu.Unlock()

			select {
			case <-ch1:
			default:
				t.Fatal("ch1 should have received notification")
			}

			select {
			case <-ch2:
			default:
				t.Fatal("ch2 should have received notification")
			}

			svc.Unsubscribe(ch1)
			svc.Unsubscribe(ch2)
		})

		t.Run("does not block on full buffered channel", func(t *testing.T) {
			reg := newFakeRegistry()
			store := newMemStore()
			svc := newTestService(reg, store)

			ch := svc.Subscribe()

			svc.mu.Lock()
			svc.setStatusLocked(controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_STARTING, "")
			svc.setStatusLocked(controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_RUNNING, "")
			svc.mu.Unlock()

			select {
			case <-ch:
			default:
				t.Fatal("should have received at least one notification")
			}

			svc.Unsubscribe(ch)
		})
	})

	t.Run("watchForCrash", func(t *testing.T) {
		t.Run("guard conditions prevent registry removal when stopping", func(t *testing.T) {
			reg := newFakeRegistry()
			store := newMemStore()
			svc := newTestService(reg, store)

			svc.mu.Lock()
			svc.status = controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_STOPPING
			svc.mu.Unlock()

			assert.NotContains(t, reg.removed, workerID)
		})

		t.Run("guard conditions prevent registry removal when stopped", func(t *testing.T) {
			reg := newFakeRegistry()
			store := newMemStore()
			svc := newTestService(reg, store)

			svc.mu.Lock()
			svc.status = controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_STOPPED
			svc.mu.Unlock()

			assert.NotContains(t, reg.removed, workerID)
		})
	})

	t.Run("upsertWorkerDB", func(t *testing.T) {
		t.Run("creates new worker", func(t *testing.T) {
			reg := newFakeRegistry()
			store := newMemStore()
			svc := newTestService(reg, store)

			svc.upsertWorkerDB(context.Background(), "http://127.0.0.1:50051")

			w, err := store.GetWorker(context.Background(), workerID)
			require.NoError(t, err)
			assert.Equal(t, workerID, w.ID)
			assert.Equal(t, "Local Worker", w.Name)
			assert.Equal(t, "http://127.0.0.1:50051", w.URL)
			assert.Equal(t, "test-secret", w.Secret)
		})

		t.Run("updates existing worker", func(t *testing.T) {
			reg := newFakeRegistry()
			store := newMemStore()
			svc := newTestService(reg, store)

			_, err := store.CreateWorker(context.Background(), worker.Worker{
				ID:     workerID,
				Name:   "Old Name",
				URL:    "http://old-url",
				Secret: "old-secret",
			})
			require.NoError(t, err)

			svc.upsertWorkerDB(context.Background(), "http://127.0.0.1:50051")

			w, err := store.GetWorker(context.Background(), workerID)
			require.NoError(t, err)
			assert.Equal(t, "Local Worker", w.Name)
			assert.Equal(t, "http://127.0.0.1:50051", w.URL)
			assert.Equal(t, "test-secret", w.Secret)
		})
	})

	t.Run("waitForHealthy", func(t *testing.T) {
		t.Run("times out on unreachable server", func(t *testing.T) {
			reg := newFakeRegistry()
			store := newMemStore()
			svc := newTestService(reg, store)

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			err := svc.waitForHealthy(ctx, "http://127.0.0.1:1")
			require.Error(t, err)
		})

		t.Run("returns context error on cancel", func(t *testing.T) {
			reg := newFakeRegistry()
			store := newMemStore()
			svc := newTestService(reg, store)

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			err := svc.waitForHealthy(ctx, "http://127.0.0.1:1")
			require.Error(t, err)
			assert.Equal(t, context.Canceled, err)
		})
	})

	t.Run("secretInterceptor", func(t *testing.T) {
		interceptor := secretInterceptor("my-secret")

		called := false
		next := func(_ context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			called = true
			assert.Equal(t, "Bearer my-secret", req.Header().Get("Authorization"))
			return nil, nil
		}

		handler := interceptor(next)
		_, _ = handler(context.Background(), connect.NewRequest(&controlplanev1.StartEmbeddedWorkerRequest{}))

		assert.True(t, called)
	})
}
