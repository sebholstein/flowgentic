package workload

import (
	"context"
	"testing"

	"github.com/sebastianm/flowgentic/internal/worker/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWorkloadManager(t *testing.T) {
	d := newFakeDriver("test-agent")
	m := NewWorkloadManager(testLogger(), "", "", d)

	t.Run("lists drivers", func(t *testing.T) {
		drivers := m.ListDrivers()
		require.Len(t, drivers, 1)
		assert.Equal(t, "test-agent", drivers[0].Agent)
	})

	t.Run("no sessions initially", func(t *testing.T) {
		sessions := m.ListSessions()
		assert.Empty(t, sessions)
	})
}

func TestWorkloadManager_Launch(t *testing.T) {
	t.Run("launches session successfully", func(t *testing.T) {
		d := newFakeDriver("test-agent")
		m := NewWorkloadManager(testLogger(), "", "", d)

		var events []driver.Event
		sess, err := m.Launch(context.Background(), "wl-1", "test-agent", driver.LaunchOpts{
			Mode:   driver.SessionModeHeadless,
			Prompt: "hello",
		}, func(e driver.Event) {
			events = append(events, e)
		})
		require.NoError(t, err)
		assert.NotNil(t, sess)
		assert.Equal(t, driver.SessionStatusRunning, sess.Info().Status)

		// Should have received session start event.
		require.Len(t, events, 1)
		assert.Equal(t, driver.EventTypeSessionStart, events[0].Type)
	})

	t.Run("unknown driver returns error", func(t *testing.T) {
		m := NewWorkloadManager(testLogger(), "", "")
		_, err := m.Launch(context.Background(), "wl-1", "nonexistent", driver.LaunchOpts{}, nil)
		assert.ErrorContains(t, err, "unknown agent driver")
	})

	t.Run("rejects resume without capability", func(t *testing.T) {
		d := newFakeDriver("test-agent")
		m := NewWorkloadManager(testLogger(), "", "", d)
		_, err := m.Launch(context.Background(), "wl-1", "test-agent", driver.LaunchOpts{
			SessionID: "old-session",
		}, nil)
		assert.ErrorContains(t, err, "does not support session resume")
	})

	t.Run("rejects model without capability", func(t *testing.T) {
		d := newFakeDriver("test-agent")
		m := NewWorkloadManager(testLogger(), "", "", d)
		_, err := m.Launch(context.Background(), "wl-1", "test-agent", driver.LaunchOpts{
			Model: "gpt-4",
		}, nil)
		assert.ErrorContains(t, err, "does not support custom model")
	})

	t.Run("rejects system prompt without capability", func(t *testing.T) {
		d := newFakeDriver("test-agent")
		m := NewWorkloadManager(testLogger(), "", "", d)
		_, err := m.Launch(context.Background(), "wl-1", "test-agent", driver.LaunchOpts{
			SystemPrompt: "be helpful",
		}, nil)
		assert.ErrorContains(t, err, "does not support system prompts")
	})

	t.Run("rejects yolo without capability", func(t *testing.T) {
		d := newFakeDriver("test-agent")
		m := NewWorkloadManager(testLogger(), "", "", d)
		_, err := m.Launch(context.Background(), "wl-1", "test-agent", driver.LaunchOpts{
			Yolo: true,
		}, nil)
		assert.ErrorContains(t, err, "does not support yolo")
	})

	t.Run("accepts capabilities when supported", func(t *testing.T) {
		d := newFakeDriver("test-agent", driver.CapCustomModel, driver.CapSystemPrompt, driver.CapYolo, driver.CapSessionResume)
		m := NewWorkloadManager(testLogger(), "", "", d)
		sess, err := m.Launch(context.Background(), "wl-1", "test-agent", driver.LaunchOpts{
			Mode:         driver.SessionModeHeadless,
			Model:        "gpt-4",
			SystemPrompt: "be helpful",
			Yolo:         true,
			SessionID:    "resume-id",
		}, nil)
		require.NoError(t, err)
		assert.NotNil(t, sess)
	})

	t.Run("driver launch error propagated", func(t *testing.T) {
		d := &errDriver{id: "broken"}
		m := NewWorkloadManager(testLogger(), "", "", d)
		_, err := m.Launch(context.Background(), "wl-1", "broken", driver.LaunchOpts{}, nil)
		assert.ErrorContains(t, err, "launch failed")
	})

	t.Run("agent session ID populated from session info", func(t *testing.T) {
		d := newFakeDriver("test-agent")
		m := NewWorkloadManager(testLogger(), "", "", d)
		sess, err := m.Launch(context.Background(), "wl-sid", "test-agent", driver.LaunchOpts{
			Mode: driver.SessionModeHeadless,
		}, nil)
		require.NoError(t, err)
		assert.Equal(t, "fake-agent-session-id", sess.Info().AgentSessionID)
	})
}

func TestWorkloadManager_GetSession(t *testing.T) {
	d := newFakeDriver("test-agent")
	m := NewWorkloadManager(testLogger(), "", "", d)

	workloadID := "wl-get"
	_, err := m.Launch(context.Background(), workloadID, "test-agent", driver.LaunchOpts{
		Mode: driver.SessionModeHeadless,
	}, nil)
	require.NoError(t, err)

	t.Run("finds existing session", func(t *testing.T) {
		found, ok := m.GetSession(workloadID)
		assert.True(t, ok)
		assert.NotNil(t, found)
	})

	t.Run("returns false for missing session", func(t *testing.T) {
		_, ok := m.GetSession("nonexistent")
		assert.False(t, ok)
	})
}

func TestWorkloadManager_ListSessions(t *testing.T) {
	d := newFakeDriver("test-agent")
	m := NewWorkloadManager(testLogger(), "", "", d)

	_, err := m.Launch(context.Background(), "wl-list-1", "test-agent", driver.LaunchOpts{Mode: driver.SessionModeHeadless}, nil)
	require.NoError(t, err)
	_, err = m.Launch(context.Background(), "wl-list-2", "test-agent", driver.LaunchOpts{Mode: driver.SessionModeHeadless}, nil)
	require.NoError(t, err)

	sessions := m.ListSessions()
	assert.Len(t, sessions, 2)
}

func TestWorkloadManager_StopSession(t *testing.T) {
	d := newFakeDriver("test-agent")
	m := NewWorkloadManager(testLogger(), "", "", d)

	workloadID := "wl-stop"
	_, err := m.Launch(context.Background(), workloadID, "test-agent", driver.LaunchOpts{
		Mode: driver.SessionModeHeadless,
	}, nil)
	require.NoError(t, err)

	err = m.StopSession(context.Background(), workloadID)
	require.NoError(t, err)

	_, ok := m.GetSession(workloadID)
	assert.False(t, ok)
}

func TestWorkloadManager_StopSession_NotFound(t *testing.T) {
	m := NewWorkloadManager(testLogger(), "", "")
	err := m.StopSession(context.Background(), "nonexistent")
	assert.ErrorContains(t, err, "session not found")
}

func TestWorkloadManager_HandleHookEvent(t *testing.T) {
	d := newFakeDriver("test-agent")
	m := NewWorkloadManager(testLogger(), "", "", d)

	workloadID := "wl-hook"
	_, err := m.Launch(context.Background(), workloadID, "test-agent", driver.LaunchOpts{
		Mode: driver.SessionModeHeadless,
	}, nil)
	require.NoError(t, err)

	t.Run("routes to driver", func(t *testing.T) {
		hookEvt := driver.HookEvent{
			SessionID: workloadID,
			Agent:     "test-agent",
			HookName:  "Stop",
			Payload:   []byte(`{}`),
		}
		err := m.HandleHookEvent(context.Background(), hookEvt)
		require.NoError(t, err)

		d.mu.Lock()
		defer d.mu.Unlock()
		require.Len(t, d.hookEvents, 1)
		assert.Equal(t, "Stop", d.hookEvents[0].HookName)
	})

	t.Run("unknown session returns error", func(t *testing.T) {
		err := m.HandleHookEvent(context.Background(), driver.HookEvent{SessionID: "unknown"})
		assert.ErrorContains(t, err, "session not found")
	})
}
