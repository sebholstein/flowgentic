package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/sebastianm/flowgentic/internal/worker/driver"
)

// DriverDeps are the dependencies for the Codex driver.
type DriverDeps struct {
	Log *slog.Logger
}

// Driver implements driver.Driver for Codex using a shared app-server process.
type Driver struct {
	log *slog.Logger

	mu       sync.Mutex
	sessions map[string]*codexSession // keyed by threadID
	server   *appServer              // shared, lazily initialized
	serverCtx    context.Context
	serverCancel context.CancelFunc
}

// NewDriver creates a new Codex driver.
func NewDriver(deps DriverDeps) *Driver {
	return &Driver{
		log:      deps.Log.With("driver", agent),
		sessions: make(map[string]*codexSession),
	}
}

func (d *Driver) Agent() string { return agent }

func (d *Driver) Capabilities() driver.Capabilities {
	return driver.Capabilities{
		Agent: agent,
		Supported: []driver.Capability{
			driver.CapStreaming,
			driver.CapCustomModel,
			driver.CapYolo,
			driver.CapSystemPrompt,
		},
	}
}

// ensureServer lazily starts the shared app-server process.
func (d *Driver) ensureServer(ctx context.Context) error {
	if d.server != nil {
		select {
		case <-d.server.done:
			// Server died, need to restart.
			d.server = nil
		default:
			return nil
		}
	}

	d.serverCtx, d.serverCancel = context.WithCancel(ctx)

	srv := newAppServer(d.log, d.dispatchNotification)
	if err := srv.start(d.serverCtx); err != nil {
		d.serverCancel()
		return err
	}

	d.server = srv

	// Monitor server lifetime — clear sessions if server dies.
	go func() {
		<-srv.done
		d.mu.Lock()
		defer d.mu.Unlock()
		// Notify all sessions that the server died.
		for threadID, sess := range d.sessions {
			sess.emit(driver.Event{
				Type:      driver.EventTypeError,
				Timestamp: currentTime(),
				Agent:     agent,
				Error:     "codex app-server process exited",
			})
			sess.setStatus(driver.SessionStatusErrored)
			sess.closeDone()
			delete(d.sessions, threadID)
		}
	}()

	return nil
}

// dispatchNotification routes app-server notifications to the correct session.
func (d *Driver) dispatchNotification(threadID string, method string, params json.RawMessage, serverRequestID *int64) {
	// Handle approval requests — auto-accept.
	if method == "item/commandExecution/requestApproval" && serverRequestID != nil {
		d.log.Debug("auto-accepting approval request", "threadID", threadID)
		d.mu.Lock()
		srv := d.server
		d.mu.Unlock()
		if srv != nil {
			srv.respondToServerRequest(*serverRequestID, map[string]string{"decision": "accept"})
		}
		return
	}

	d.mu.Lock()
	sess, ok := d.sessions[threadID]
	d.mu.Unlock()

	if !ok {
		d.log.Debug("notification for unknown thread", "threadID", threadID, "method", method)
		return
	}

	events := normalizeNotification(method, params)
	sess.emit(events...)

	// If turn completed, mark session idle.
	if method == "turn/completed" {
		sess.setStatus(driver.SessionStatusIdle)
	}
}

func (d *Driver) Launch(ctx context.Context, opts driver.LaunchOpts, onEvent driver.EventCallback) (driver.Session, error) {
	// Grab server ref under lock, but release before RPC calls to avoid
	// deadlock: readLoop dispatches notifications via dispatchNotification
	// which also acquires d.mu.
	d.mu.Lock()
	if err := d.ensureServer(ctx); err != nil {
		d.mu.Unlock()
		return nil, fmt.Errorf("ensure app-server: %w", err)
	}
	srv := d.server
	d.mu.Unlock()

	// Create a thread (RPC call — must not hold d.mu).
	threadID, err := srv.threadStart(opts.Model, opts.Cwd, opts.SystemPrompt, opts.Yolo)
	if err != nil {
		return nil, fmt.Errorf("thread/start: %w", err)
	}

	sess := &codexSession{
		info: driver.SessionInfo{
			ID:             opts.SessionID,
			AgentID:        agent,
			AgentSessionID: threadID,
			Status:         driver.SessionStatusRunning,
			Mode:           opts.Mode,
			Cwd:            opts.Cwd,
			StartedAt:      time.Now(),
		},
		driver:   d,
		threadID: threadID,
		onEvent:  onEvent,
		done:     make(chan struct{}),
	}

	// Register session so notifications can be routed to it.
	d.mu.Lock()
	d.sessions[threadID] = sess
	d.mu.Unlock()

	// Emit session start event.
	sess.emit(driver.Event{
		Type:      driver.EventTypeSessionStart,
		Timestamp: currentTime(),
		Agent:     agent,
		Text:      threadID,
	})

	// Start the turn (RPC call — must not hold d.mu).
	turnID, err := srv.turnStart(threadID, opts.Prompt)
	if err != nil {
		d.removeSession(threadID)
		return nil, fmt.Errorf("turn/start: %w", err)
	}
	sess.mu.Lock()
	sess.turnID = turnID
	sess.mu.Unlock()

	return sess, nil
}

func (d *Driver) HandleHookEvent(_ context.Context, sessionID string, event driver.HookEvent) error {
	d.mu.Lock()
	// Find session by sessionID (not threadID).
	var sess *codexSession
	for _, s := range d.sessions {
		if s.info.ID == sessionID {
			sess = s
			break
		}
	}
	d.mu.Unlock()

	if sess == nil {
		return fmt.Errorf("codex session not found: %s", sessionID)
	}

	now := currentTime()

	switch event.HookName {
	case "Stop", "TurnComplete":
		sess.setStatus(driver.SessionStatusIdle)
		sess.emit(driver.Event{
			Type:      driver.EventTypeTurnComplete,
			Timestamp: now,
			Agent:     agent,
			Raw:       event.Payload,
		})
	default:
		d.log.Debug("unhandled codex hook", "hook", event.HookName)
	}

	return nil
}

func (d *Driver) removeSession(threadID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.sessions, threadID)
}

// codexSession represents a Codex session backed by a thread on the shared app-server.
type codexSession struct {
	info     driver.SessionInfo
	driver   *Driver
	threadID string
	turnID   string
	onEvent  driver.EventCallback
	done     chan struct{}
	mu       sync.Mutex
}

func (s *codexSession) Info() driver.SessionInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.info
}

func (s *codexSession) Stop(_ context.Context) error {
	s.mu.Lock()
	s.info.Status = driver.SessionStatusStopping
	s.mu.Unlock()

	// Try to interrupt the turn, but don't fail if the server is gone.
	s.driver.mu.Lock()
	srv := s.driver.server
	s.driver.mu.Unlock()

	if srv != nil {
		s.mu.Lock()
		turnID := s.turnID
		s.mu.Unlock()
		if turnID != "" {
			_ = srv.turnInterrupt(s.threadID, turnID)
		}
	}

	s.driver.removeSession(s.threadID)

	s.mu.Lock()
	s.info.Status = driver.SessionStatusStopped
	s.mu.Unlock()

	s.closeDone()
	return nil
}

func (s *codexSession) Wait(_ context.Context) error {
	<-s.done
	return nil
}

func (s *codexSession) closeDone() {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
}

func (s *codexSession) setStatus(status driver.SessionStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.info.Status = status
}

func (s *codexSession) emit(events ...driver.Event) {
	if s.onEvent == nil {
		return
	}
	for _, e := range events {
		s.onEvent(e)
	}
}
