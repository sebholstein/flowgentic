package agentctl

import (
	"context"
	"log/slog"
	"os"
	"time"

	acp "github.com/coder/acp-go-sdk"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
	v2 "github.com/sebastianm/flowgentic/internal/worker/driver/v2"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// fakeDriver is a test double for v2.Driver.
type fakeDriver struct {
	id   string
	caps driver.Capabilities
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
	sess := &fakeSession{
		info: v2.SessionInfo{
			ID:        opts.ResumeSessionID,
			AgentID:   d.id,
			Status:    v2.SessionStatusRunning,
			Cwd:       opts.Cwd,
			StartedAt: time.Now(),
		},
		done: make(chan struct{}),
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
	info v2.SessionInfo
	done chan struct{}
}

func (s *fakeSession) Info() v2.SessionInfo { return s.info }

func (s *fakeSession) Stop(_ context.Context) error {
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
