package agentctl

import (
	"context"
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
	sess := &fakeSession{
		info: driver.SessionInfo{
			ID:        opts.SessionID,
			AgentID:   d.id,
			Status:    driver.SessionStatusRunning,
			Mode:      opts.Mode,
			StartedAt: time.Now(),
		},
		done: make(chan struct{}),
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
	info driver.SessionInfo
	done chan struct{}
}

func (s *fakeSession) Info() driver.SessionInfo { return s.info }

func (s *fakeSession) Stop(_ context.Context) error {
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
