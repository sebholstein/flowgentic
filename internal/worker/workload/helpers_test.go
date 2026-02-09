package workload

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/sebastianm/flowgentic/internal/worker/driver"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// fakeDriver is a test double for driver.Driver.
type fakeDriver struct {
	id   string
	caps driver.Capabilities

	launchErr  error
	launchSess *fakeSession
	hookEvents []driver.HookEvent
	mu         sync.Mutex
}

func newFakeDriver(id string, caps ...driver.Capability) *fakeDriver {
	return &fakeDriver{
		id: id,
		caps: driver.Capabilities{
			Agent:     id,
			Supported: caps,
		},
	}
}

func (d *fakeDriver) Agent() string                      { return d.id }
func (d *fakeDriver) Capabilities() driver.Capabilities { return d.caps }

func (d *fakeDriver) Launch(_ context.Context, opts driver.LaunchOpts, onEvent driver.EventCallback) (driver.Session, error) {
	if d.launchErr != nil {
		return nil, d.launchErr
	}
	sess := d.launchSess
	if sess == nil {
		sess = newFakeSession(opts.SessionID, d.id, opts.Mode)
	}
	if onEvent != nil {
		onEvent(driver.Event{
			Type:      driver.EventTypeSessionStart,
			Timestamp: time.Now(),
			Agent:     d.id,
		})
	}
	return sess, nil
}

func (d *fakeDriver) HandleHookEvent(_ context.Context, _ string, event driver.HookEvent) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.hookEvents = append(d.hookEvents, event)
	return nil
}

// fakeSession is a test double for driver.Session.
type fakeSession struct {
	info    driver.SessionInfo
	stopErr error
	done    chan struct{}
	mu      sync.Mutex
}

func newFakeSession(id, agentID string, mode driver.SessionMode) *fakeSession {
	return &fakeSession{
		info: driver.SessionInfo{
			ID:             id,
			AgentID:        agentID,
			AgentSessionID: "fake-agent-session-id",
			Status:         driver.SessionStatusRunning,
			Mode:           mode,
			StartedAt:      time.Now(),
		},
		done: make(chan struct{}),
	}
}

func (s *fakeSession) Info() driver.SessionInfo { return s.info }

func (s *fakeSession) Stop(_ context.Context) error {
	if s.stopErr != nil {
		return s.stopErr
	}
	s.info.Status = driver.SessionStatusStopped
	select {
	case <-s.done:
	default:
		close(s.done)
	}
	return nil
}

func (s *fakeSession) Wait(_ context.Context) error {
	<-s.done
	return nil
}

// errDriver is a driver that always fails to launch.
type errDriver struct {
	id string
}

func (d *errDriver) Agent() string { return d.id }
func (d *errDriver) Capabilities() driver.Capabilities {
	return driver.Capabilities{Agent: d.id, Supported: []driver.Capability{}}
}
func (d *errDriver) Launch(_ context.Context, _ driver.LaunchOpts, _ driver.EventCallback) (driver.Session, error) {
	return nil, fmt.Errorf("launch failed")
}
func (d *errDriver) HandleHookEvent(_ context.Context, _ string, _ driver.HookEvent) error {
	return fmt.Errorf("not supported")
}
