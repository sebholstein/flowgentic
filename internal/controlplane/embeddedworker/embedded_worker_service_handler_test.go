package embeddedworker

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"connectrpc.com/connect"
	controlplanev1 "github.com/sebastianm/flowgentic/internal/proto/gen/controlplane/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestHandler(svc *EmbeddedWorkerService) *embeddedWorkerServiceHandler {
	return &embeddedWorkerServiceHandler{
		log: slog.New(slog.NewTextHandler(os.Stdout, nil)),
		svc: svc,
	}
}

func TestEmbeddedWorkerServiceHandler(t *testing.T) {
	t.Run("StartEmbeddedWorker", func(t *testing.T) {
		t.Run("returns Internal error when already running", func(t *testing.T) {
			reg := newFakeRegistry()
			store := newMemStore()
			svc := newTestService(reg, store)

			svc.mu.Lock()
			svc.status = controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_RUNNING
			svc.pid = 12345
			svc.listenAddr = "127.0.0.1:50051"
			svc.mu.Unlock()

			h := newTestHandler(svc)

			_, err := h.StartEmbeddedWorker(context.Background(), connect.NewRequest(&controlplanev1.StartEmbeddedWorkerRequest{}))
			require.Error(t, err)
			assert.Equal(t, connect.CodeInternal, connect.CodeOf(err))
		})

		t.Run("returns Internal error when already starting", func(t *testing.T) {
			reg := newFakeRegistry()
			store := newMemStore()
			svc := newTestService(reg, store)

			svc.mu.Lock()
			svc.status = controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_STARTING
			svc.mu.Unlock()

			h := newTestHandler(svc)

			_, err := h.StartEmbeddedWorker(context.Background(), connect.NewRequest(&controlplanev1.StartEmbeddedWorkerRequest{}))
			require.Error(t, err)
			assert.Equal(t, connect.CodeInternal, connect.CodeOf(err))
		})

		t.Run("returns Internal error on binary not found", func(t *testing.T) {
			reg := newFakeRegistry()
			store := newMemStore()
			svc := newTestService(reg, store)

			h := newTestHandler(svc)

			_, err := h.StartEmbeddedWorker(context.Background(), connect.NewRequest(&controlplanev1.StartEmbeddedWorkerRequest{}))
			require.Error(t, err)
			assert.Equal(t, connect.CodeInternal, connect.CodeOf(err))
		})
	})

	t.Run("StopEmbeddedWorker", func(t *testing.T) {
		t.Run("stops running worker", func(t *testing.T) {
			reg := newFakeRegistry()
			store := newMemStore()
			svc := newTestService(reg, store)

			svc.mu.Lock()
			svc.status = controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_RUNNING
			svc.cancel = func() {}
			svc.mu.Unlock()

			h := newTestHandler(svc)

			resp, err := h.StopEmbeddedWorker(context.Background(), connect.NewRequest(&controlplanev1.StopEmbeddedWorkerRequest{}))
			require.NoError(t, err)
			assert.Equal(t, controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_STOPPED, resp.Msg.Status)
		})

		t.Run("stops errored worker", func(t *testing.T) {
			reg := newFakeRegistry()
			store := newMemStore()
			svc := newTestService(reg, store)

			svc.mu.Lock()
			svc.status = controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_ERRORED
			svc.lastErr = "crashed"
			svc.mu.Unlock()

			h := newTestHandler(svc)

			resp, err := h.StopEmbeddedWorker(context.Background(), connect.NewRequest(&controlplanev1.StopEmbeddedWorkerRequest{}))
			require.NoError(t, err)
			assert.Equal(t, controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_STOPPED, resp.Msg.Status)
		})

		t.Run("returns Internal error when not running", func(t *testing.T) {
			reg := newFakeRegistry()
			store := newMemStore()
			svc := newTestService(reg, store)

			svc.mu.Lock()
			svc.status = controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_STARTING
			svc.mu.Unlock()

			h := newTestHandler(svc)

			_, err := h.StopEmbeddedWorker(context.Background(), connect.NewRequest(&controlplanev1.StopEmbeddedWorkerRequest{}))
			require.Error(t, err)
			assert.Equal(t, connect.CodeInternal, connect.CodeOf(err))
		})
	})

	t.Run("RestartEmbeddedWorker", func(t *testing.T) {
		t.Run("returns Internal error on failure", func(t *testing.T) {
			reg := newFakeRegistry()
			store := newMemStore()
			svc := newTestService(reg, store)

			h := newTestHandler(svc)

			_, err := h.RestartEmbeddedWorker(context.Background(), connect.NewRequest(&controlplanev1.RestartEmbeddedWorkerRequest{}))
			require.Error(t, err)
			assert.Equal(t, connect.CodeInternal, connect.CodeOf(err))
		})

		t.Run("returns Internal error when starting", func(t *testing.T) {
			reg := newFakeRegistry()
			store := newMemStore()
			svc := newTestService(reg, store)

			svc.mu.Lock()
			svc.status = controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_STARTING
			svc.mu.Unlock()

			h := newTestHandler(svc)

			_, err := h.RestartEmbeddedWorker(context.Background(), connect.NewRequest(&controlplanev1.RestartEmbeddedWorkerRequest{}))
			require.Error(t, err)
			assert.Equal(t, connect.CodeInternal, connect.CodeOf(err))
		})
	})

	t.Run("secretInterceptor", func(t *testing.T) {
		t.Run("injects Authorization header", func(t *testing.T) {
			interceptor := secretInterceptor("test-secret-123")

			var capturedAuth string
			next := func(_ context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
				capturedAuth = req.Header().Get("Authorization")
				return nil, errors.New("expected error")
			}

			handler := interceptor(next)
			_, _ = handler(context.Background(), connect.NewRequest(&controlplanev1.StartEmbeddedWorkerRequest{}))

			assert.Equal(t, "Bearer test-secret-123", capturedAuth)
		})
	})
}
