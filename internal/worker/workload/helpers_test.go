package workload

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	acp "github.com/coder/acp-go-sdk"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
	v2 "github.com/sebastianm/flowgentic/internal/worker/driver/v2"
)

var _ v2.Session = (*fakeSession)(nil)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// fakeDriver is a test double for v2.Driver.
type fakeDriver struct {
	id   string
	caps driver.Capabilities

	launchErr  error
	launchSess *fakeSession
	lastOpts   v2.LaunchOpts
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

func (d *fakeDriver) Agent() string                     { return d.id }
func (d *fakeDriver) Capabilities() driver.Capabilities { return d.caps }
func (d *fakeDriver) DiscoverModels(_ context.Context, _ string) (v2.ModelInventory, error) {
	return v2.ModelInventory{
		Models:       []string{"test-model"},
		DefaultModel: "test-model",
	}, nil
}

func (d *fakeDriver) Launch(_ context.Context, opts v2.LaunchOpts, onEvent v2.EventCallback) (v2.Session, error) {
	if d.launchErr != nil {
		return nil, d.launchErr
	}
	d.mu.Lock()
	d.lastOpts = opts
	d.mu.Unlock()
	sess := d.launchSess
	if sess == nil {
		sess = newFakeSession(opts.ResumeSessionID, d.id)
	}
	if onEvent != nil {
		onEvent(acp.SessionNotification{
			SessionId: acp.SessionId(opts.ResumeSessionID),
			Update:    acp.UpdateAgentMessageText("session started"),
		})
	}
	return sess, nil
}

// fakeSession is a test double for v2.Session.
type fakeSession struct {
	info    v2.SessionInfo
	stopErr error
	done    chan struct{}
	mu      sync.Mutex
}

func newFakeSession(id, agentID string) *fakeSession {
	return &fakeSession{
		info: v2.SessionInfo{
			ID:             id,
			AgentID:        agentID,
			AgentSessionID: "fake-agent-session-id",
			Status:         v2.SessionStatusRunning,
			Cwd:            "/tmp",
			StartedAt:      time.Now(),
		},
		done: make(chan struct{}),
	}
}

func (s *fakeSession) Info() v2.SessionInfo { return s.info }

func (s *fakeSession) Stop(_ context.Context) error {
	if s.stopErr != nil {
		return s.stopErr
	}
	s.info.Status = v2.SessionStatusStopped
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

func (s *fakeSession) Prompt(_ context.Context, _ []acp.ContentBlock) (*acp.PromptResponse, error) {
	return &acp.PromptResponse{}, nil
}

func (s *fakeSession) Cancel(_ context.Context) error {
	return nil
}

func (s *fakeSession) RespondToPermission(_ context.Context, _ string, _ bool, _ string) error {
	return nil
}

func (s *fakeSession) SetSessionMode(_ context.Context, _ driver.SessionMode) error {
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
func (d *errDriver) DiscoverModels(_ context.Context, _ string) (v2.ModelInventory, error) {
	return v2.ModelInventory{}, fmt.Errorf("discover models failed")
}
func (d *errDriver) Launch(_ context.Context, _ v2.LaunchOpts, _ v2.EventCallback) (v2.Session, error) {
	return nil, fmt.Errorf("launch failed")
}
