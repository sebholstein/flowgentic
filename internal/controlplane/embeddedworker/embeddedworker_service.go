package embeddedworker

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"connectrpc.com/connect"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
	"github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1/workerv1connect"

	"github.com/sebastianm/flowgentic/internal/controlplane/worker"
	"github.com/sebastianm/flowgentic/internal/portutil"

	controlplanev1 "github.com/sebastianm/flowgentic/internal/proto/gen/controlplane/v1"
)

// RegistryUpdater keeps the relay registry in sync with dynamically managed
// workers.
type RegistryUpdater interface {
	AddWorker(id string, rawURL string, secret string) error
	RemoveWorker(id string)
}

// Store persists worker configurations.
type Store interface {
	GetWorker(ctx context.Context, id string) (worker.Worker, error)
	CreateWorker(ctx context.Context, w worker.Worker) (worker.Worker, error)
	UpdateWorker(ctx context.Context, w worker.Worker) (worker.Worker, error)
}

// EmbeddedWorkerService manages the lifecycle of an embedded worker child
// process.
type EmbeddedWorkerService struct {
	log        *slog.Logger
	binaryPath string
	secret     string
	configPath string
	registry   RegistryUpdater
	store      Store

	mu          sync.Mutex
	status      controlplanev1.EmbeddedWorkerStatus
	lastErr     string
	cmd         *exec.Cmd
	cancel      context.CancelFunc
	listenAddr  string
	pid         int
	subscribers map[chan struct{}]struct{}
}

const workerID = "local"

// NewEmbeddedWorkerService creates a new process manager for the embedded
// worker.
func NewEmbeddedWorkerService(
	log *slog.Logger,
	binaryPath string,
	secret string,
	configPath string,
	registry RegistryUpdater,
	store Store,
) *EmbeddedWorkerService {
	return &EmbeddedWorkerService{
		log:         log,
		binaryPath:  binaryPath,
		secret:      secret,
		configPath:  configPath,
		registry:    registry,
		store:       store,
		status:      controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_STOPPED,
		subscribers: make(map[chan struct{}]struct{}),
	}
}

// Start spawns the worker child process. It is safe to call from multiple
// goroutines.
func (s *EmbeddedWorkerService) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.status == controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_RUNNING ||
		s.status == controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_STARTING {
		s.mu.Unlock()
		return fmt.Errorf("worker is already %s", s.status)
	}
	s.setStatusLocked(controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_STARTING, "")
	s.mu.Unlock()

	port, err := portutil.FindFreePort()
	if err != nil {
		s.mu.Lock()
		s.setStatusLocked(controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_ERRORED, err.Error())
		s.mu.Unlock()
		return fmt.Errorf("finding free port: %w", err)
	}

	listenAddr := fmt.Sprintf("127.0.0.1:%d", port)
	workerURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	childCtx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(childCtx, s.binaryPath, "--listen-addr="+listenAddr)
	cmd.Env = append(os.Environ(),
		"FLOWGENTIC_WORKER_SECRET="+s.secret,
		"FLOWGENTIC_CONFIG="+s.configPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Cancel = func() error {
		return cmd.Process.Signal(syscall.SIGTERM)
	}
	cmd.WaitDelay = 5 * time.Second

	if err := cmd.Start(); err != nil {
		cancel()
		s.mu.Lock()
		s.setStatusLocked(controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_ERRORED, err.Error())
		s.mu.Unlock()
		return fmt.Errorf("starting worker process: %w", err)
	}

	s.mu.Lock()
	s.cmd = cmd
	s.cancel = cancel
	s.listenAddr = listenAddr
	s.pid = cmd.Process.Pid
	s.mu.Unlock()

	s.log.Info("embedded worker process started", "pid", cmd.Process.Pid, "addr", listenAddr)

	go s.watchForCrash(cmd)

	// Poll health until ready.
	if err := s.waitForHealthy(ctx, workerURL); err != nil {
		cancel()
		s.mu.Lock()
		s.setStatusLocked(controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_ERRORED, err.Error())
		s.mu.Unlock()
		return fmt.Errorf("worker health check failed: %w", err)
	}

	// Register with relay and upsert in DB.
	if err := s.registry.AddWorker(workerID, workerURL, s.secret); err != nil {
		s.log.Error("failed to register embedded worker in relay", "error", err)
	}
	s.upsertWorkerDB(ctx, workerURL)

	s.mu.Lock()
	s.setStatusLocked(controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_RUNNING, "")
	s.mu.Unlock()

	s.log.Info("embedded worker is running", "pid", cmd.Process.Pid, "url", workerURL)
	return nil
}

// Stop gracefully shuts down the worker process.
func (s *EmbeddedWorkerService) Stop(ctx context.Context) error {
	s.mu.Lock()
	if s.status != controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_RUNNING &&
		s.status != controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_ERRORED {
		s.mu.Unlock()
		return fmt.Errorf("worker is not running (status: %s)", s.status)
	}
	s.setStatusLocked(controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_STOPPING, "")
	cancel := s.cancel
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	s.registry.RemoveWorker(workerID)

	s.mu.Lock()
	s.cmd = nil
	s.cancel = nil
	s.pid = 0
	s.listenAddr = ""
	s.setStatusLocked(controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_STOPPED, "")
	s.mu.Unlock()

	s.log.Info("embedded worker stopped")
	return nil
}

// Restart stops and re-starts the worker process.
func (s *EmbeddedWorkerService) Restart(ctx context.Context) error {
	// Stop is safe to call even if the worker crashed.
	s.mu.Lock()
	running := s.status == controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_RUNNING ||
		s.status == controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_ERRORED
	s.mu.Unlock()

	if running {
		if err := s.Stop(ctx); err != nil {
			s.log.Warn("stop during restart returned error", "error", err)
		}
	}
	return s.Start(ctx)
}

// GetStatus returns the current state.
func (s *EmbeddedWorkerService) GetStatus() (controlplanev1.EmbeddedWorkerStatus, string, int, string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status, s.lastErr, s.pid, s.listenAddr
}

// Subscribe returns a channel that receives a notification on every status
// change. The caller must eventually call Unsubscribe.
func (s *EmbeddedWorkerService) Subscribe() chan struct{} {
	ch := make(chan struct{}, 1)
	s.mu.Lock()
	s.subscribers[ch] = struct{}{}
	s.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber channel.
func (s *EmbeddedWorkerService) Unsubscribe(ch chan struct{}) {
	s.mu.Lock()
	delete(s.subscribers, ch)
	s.mu.Unlock()
}

// watchForCrash waits for the given process to exit and auto-restarts it if the
// exit was unexpected (i.e. not an intentional stop or replacement).
func (s *EmbeddedWorkerService) watchForCrash(cmd *exec.Cmd) {
	err := cmd.Wait()
	s.mu.Lock()
	// Ignore if intentionally stopped or if a new process has already replaced this one.
	if s.status == controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_STOPPING ||
		s.status == controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_STOPPED ||
		s.cmd != cmd {
		s.mu.Unlock()
		return
	}
	errMsg := "process exited unexpectedly"
	if err != nil {
		errMsg = err.Error()
	}
	s.log.Error("embedded worker crashed", "error", errMsg)
	s.registry.RemoveWorker(workerID)
	s.setStatusLocked(controlplanev1.EmbeddedWorkerStatus_EMBEDDED_WORKER_STATUS_ERRORED, errMsg)
	s.mu.Unlock()

	// Auto-restart after a short delay.
	s.log.Info("auto-restarting embedded worker in 1s")
	time.Sleep(1 * time.Second)
	if err := s.Restart(context.Background()); err != nil {
		s.log.Error("auto-restart failed", "error", err)
	}
}

// setStatusLocked updates status, lastErr, and notifies subscribers. Must be
// called with s.mu held.
func (s *EmbeddedWorkerService) setStatusLocked(status controlplanev1.EmbeddedWorkerStatus, errMsg string) {
	s.status = status
	s.lastErr = errMsg
	for ch := range s.subscribers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// waitForHealthy polls the worker's Ping RPC until it responds or the timeout
// elapses.
func (s *EmbeddedWorkerService) waitForHealthy(ctx context.Context, workerURL string) error {
	client := workerv1connect.NewSystemServiceClient(
		http.DefaultClient,
		workerURL,
		connect.WithInterceptors(secretInterceptor(s.secret)),
	)

	deadline := time.After(15 * time.Second)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			return fmt.Errorf("timed out waiting for worker to become healthy")
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			_, err := client.Ping(ctx, connect.NewRequest(&workerv1.PingRequest{}))
			if err == nil {
				return nil
			}
		}
	}
}

// upsertWorkerDB creates or updates the "local" worker row in the database.
func (s *EmbeddedWorkerService) upsertWorkerDB(ctx context.Context, url string) {
	w := worker.Worker{
		ID:     workerID,
		Name:   "Local Worker",
		URL:    url,
		Secret: s.secret,
	}
	if _, err := s.store.GetWorker(ctx, workerID); err != nil {
		if _, err := s.store.CreateWorker(ctx, w); err != nil {
			s.log.Error("failed to create embedded worker in DB", "error", err)
		}
		return
	}
	if _, err := s.store.UpdateWorker(ctx, w); err != nil {
		s.log.Error("failed to update embedded worker in DB", "error", err)
	}
}

// secretInterceptor injects the Authorization header into outgoing requests.
func secretInterceptor(secret string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+secret)
			return next(ctx, req)
		}
	}
}

