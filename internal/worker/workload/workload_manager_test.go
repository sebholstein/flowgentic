package workload

import (
	"context"
	"testing"
	"time"

	"github.com/sebastianm/flowgentic/internal/worker/driver"
	v2 "github.com/sebastianm/flowgentic/internal/worker/driver/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSessionManager(t *testing.T) {
	d := newFakeDriver("test-agent")
	m := NewSessionManager(testLogger(), "", "", d)

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

func TestSessionManager_Launch(t *testing.T) {
	t.Run("launches session successfully", func(t *testing.T) {
		d := newFakeDriver("test-agent")
		m := NewSessionManager(testLogger(), "", "", d)

		sess, err := m.Launch(context.Background(), "sess-1", "test-agent", v2.LaunchOpts{
			Prompt: "hello",
		}, nil)
		require.NoError(t, err)
		assert.NotNil(t, sess)
		assert.Equal(t, v2.SessionStatusRunning, sess.Info().Status)
	})

	t.Run("unknown driver returns error", func(t *testing.T) {
		m := NewSessionManager(testLogger(), "", "")
		_, err := m.Launch(context.Background(), "sess-1", "nonexistent", v2.LaunchOpts{}, nil)
		assert.ErrorContains(t, err, "unknown agent driver")
	})

	t.Run("rejects resume without capability", func(t *testing.T) {
		d := newFakeDriver("test-agent")
		m := NewSessionManager(testLogger(), "", "", d)
		_, err := m.Launch(context.Background(), "sess-1", "test-agent", v2.LaunchOpts{
			ResumeSessionID: "old-session",
		}, nil)
		assert.ErrorContains(t, err, "does not support session resume")
	})

	t.Run("rejects model without capability", func(t *testing.T) {
		d := newFakeDriver("test-agent")
		m := NewSessionManager(testLogger(), "", "", d)
		_, err := m.Launch(context.Background(), "sess-1", "test-agent", v2.LaunchOpts{
			Model: "gpt-4",
		}, nil)
		assert.ErrorContains(t, err, "does not support custom model")
	})

	t.Run("rejects system prompt without capability", func(t *testing.T) {
		d := newFakeDriver("test-agent")
		m := NewSessionManager(testLogger(), "", "", d)
		_, err := m.Launch(context.Background(), "sess-1", "test-agent", v2.LaunchOpts{
			SystemPrompt: "be helpful",
		}, nil)
		assert.ErrorContains(t, err, "does not support system prompts")
	})

	t.Run("accepts capabilities when supported", func(t *testing.T) {
		d := newFakeDriver("test-agent", driver.CapCustomModel, driver.CapSystemPrompt, driver.CapSessionResume)
		m := NewSessionManager(testLogger(), "", "", d)
		sess, err := m.Launch(context.Background(), "sess-1", "test-agent", v2.LaunchOpts{
			Model:           "gpt-4",
			SystemPrompt:    "be helpful",
			SessionMode:     "code",
			ResumeSessionID: "resume-id",
		}, nil)
		require.NoError(t, err)
		assert.NotNil(t, sess)
	})

	t.Run("driver launch error propagated", func(t *testing.T) {
		d := &errDriver{id: "broken"}
		m := NewSessionManager(testLogger(), "", "", d)
		_, err := m.Launch(context.Background(), "sess-1", "broken", v2.LaunchOpts{}, nil)
		assert.ErrorContains(t, err, "launch failed")
	})

	t.Run("agent session ID populated from session info", func(t *testing.T) {
		d := newFakeDriver("test-agent")
		m := NewSessionManager(testLogger(), "", "", d)
		sess, err := m.Launch(context.Background(), "sess-sid", "test-agent", v2.LaunchOpts{}, nil)
		require.NoError(t, err)
		assert.Equal(t, "fake-agent-session-id", sess.Info().AgentSessionID)
	})

	t.Run("injects AGENTCTL env vars for MCP server", func(t *testing.T) {
		d := newFakeDriver("codex")
		m := NewSessionManager(testLogger(), "http://127.0.0.1:7777", "worker-secret", d)
		_, err := m.Launch(context.Background(), "sess-env", "codex", v2.LaunchOpts{}, nil)
		require.NoError(t, err)

		d.mu.Lock()
		launchOpts := d.lastOpts
		d.mu.Unlock()

		assert.Equal(t, "http://127.0.0.1:7777", launchOpts.EnvVars["AGENTCTL_WORKER_URL"])
		assert.Equal(t, "worker-secret", launchOpts.EnvVars["AGENTCTL_WORKER_SECRET"])
		assert.Equal(t, "sess-env", launchOpts.EnvVars["AGENTCTL_AGENT_RUN_ID"])
		assert.Equal(t, "codex", launchOpts.EnvVars["AGENTCTL_AGENT"])
	})
}

func TestSessionManager_GetSession(t *testing.T) {
	d := newFakeDriver("test-agent")
	m := NewSessionManager(testLogger(), "", "", d)

	sessionID := "sess-get"
	_, err := m.Launch(context.Background(), sessionID, "test-agent", v2.LaunchOpts{}, nil)
	require.NoError(t, err)

	t.Run("finds existing session", func(t *testing.T) {
		found, ok := m.GetSession(sessionID)
		assert.True(t, ok)
		assert.NotNil(t, found)
	})

	t.Run("returns false for missing session", func(t *testing.T) {
		_, ok := m.GetSession("nonexistent")
		assert.False(t, ok)
	})
}

func TestSessionManager_ListSessions(t *testing.T) {
	d := newFakeDriver("test-agent")
	m := NewSessionManager(testLogger(), "", "", d)

	_, err := m.Launch(context.Background(), "sess-list-1", "test-agent", v2.LaunchOpts{}, nil)
	require.NoError(t, err)
	_, err = m.Launch(context.Background(), "sess-list-2", "test-agent", v2.LaunchOpts{}, nil)
	require.NoError(t, err)

	sessions := m.ListSessions()
	assert.Len(t, sessions, 2)
}

func TestSessionManager_StopSession(t *testing.T) {
	d := newFakeDriver("test-agent")
	m := NewSessionManager(testLogger(), "", "", d)

	sessionID := "sess-stop"
	_, err := m.Launch(context.Background(), sessionID, "test-agent", v2.LaunchOpts{}, nil)
	require.NoError(t, err)

	err = m.StopSession(context.Background(), sessionID)
	require.NoError(t, err)

	_, ok := m.GetSession(sessionID)
	assert.False(t, ok)
}

func TestSessionManager_StopSession_NotFound(t *testing.T) {
	m := NewSessionManager(testLogger(), "", "")
	err := m.StopSession(context.Background(), "nonexistent")
	assert.ErrorContains(t, err, "session not found")
}

func TestSessionManager_Subscribe(t *testing.T) {
	d := newFakeDriver("test-agent")
	m := NewSessionManager(testLogger(), "", "", d)

	t.Run("notifies on launch with update event", func(t *testing.T) {
		ch := m.Subscribe()
		defer m.Unsubscribe(ch)

		_, err := m.Launch(context.Background(), "sess-sub-1", "test-agent", v2.LaunchOpts{}, nil)
		require.NoError(t, err)

		select {
		case event := <-ch:
			assert.Equal(t, StateEventUpdate, event.Type)
			assert.Equal(t, "sess-sub-1", event.SessionID)
			require.NotNil(t, event.Snapshot)
		case <-time.After(time.Second):
			t.Fatal("expected notification on launch")
		}
	})

	t.Run("notifies on remove with removed event", func(t *testing.T) {
		ch := m.Subscribe()
		defer m.Unsubscribe(ch)

		// Drain any pending notification from previous launch.
		select {
		case <-ch:
		default:
		}

		err := m.StopSession(context.Background(), "sess-sub-1")
		require.NoError(t, err)

		select {
		case event := <-ch:
			assert.Equal(t, StateEventRemoved, event.Type)
			assert.Equal(t, "sess-sub-1", event.SessionID)
			assert.Nil(t, event.Snapshot)
		case <-time.After(time.Second):
			t.Fatal("expected notification on remove")
		}
	})

	t.Run("unsubscribe stops notifications", func(t *testing.T) {
		ch := m.Subscribe()
		m.Unsubscribe(ch)

		_, err := m.Launch(context.Background(), "sess-sub-2", "test-agent", v2.LaunchOpts{}, nil)
		require.NoError(t, err)

		select {
		case <-ch:
			t.Fatal("should not receive notification after unsubscribe")
		case <-time.After(50 * time.Millisecond):
			// ok
		}
	})
}

func TestSessionManager_HandleSetTopic(t *testing.T) {
	d := newFakeDriver("test-agent")
	m := NewSessionManager(testLogger(), "", "", d)

	sessionID := "sess-topic"
	_, err := m.Launch(context.Background(), sessionID, "test-agent", v2.LaunchOpts{}, nil)
	require.NoError(t, err)

	t.Run("sets topic and notifies", func(t *testing.T) {
		ch := m.Subscribe()
		defer m.Unsubscribe(ch)

		// Drain any pending notification from launch.
		select {
		case <-ch:
		default:
		}

		err := m.HandleSetTopic(context.Background(), sessionID, "working on tests")
		require.NoError(t, err)

		// Should have been notified with update event containing topic.
		select {
		case event := <-ch:
			assert.Equal(t, StateEventUpdate, event.Type)
			assert.Equal(t, sessionID, event.SessionID)
			require.NotNil(t, event.Snapshot)
			assert.Equal(t, "working on tests", event.Snapshot.Topic)
		case <-time.After(time.Second):
			t.Fatal("expected notification on topic change")
		}

		// Topic should also appear in full snapshot.
		snap := m.GetStateSnapshot()
		require.Len(t, snap, 1)
		assert.Equal(t, "working on tests", snap[0].Topic)
	})

	t.Run("unknown session returns error", func(t *testing.T) {
		err := m.HandleSetTopic(context.Background(), "nonexistent", "topic")
		assert.ErrorContains(t, err, "session not found")
	})
}

func TestSessionManager_GetStateSnapshot(t *testing.T) {
	d := newFakeDriver("test-agent")
	m := NewSessionManager(testLogger(), "", "", d)

	t.Run("empty when no sessions", func(t *testing.T) {
		snap := m.GetStateSnapshot()
		assert.Empty(t, snap)
	})

	t.Run("includes all sessions", func(t *testing.T) {
		_, err := m.Launch(context.Background(), "sess-snap-1", "test-agent", v2.LaunchOpts{}, nil)
		require.NoError(t, err)
		_, err = m.Launch(context.Background(), "sess-snap-2", "test-agent", v2.LaunchOpts{}, nil)
		require.NoError(t, err)

		snap := m.GetStateSnapshot()
		assert.Len(t, snap, 2)

		ids := map[string]bool{}
		for _, s := range snap {
			ids[s.SessionID] = true
			assert.Equal(t, "test-agent", s.Info.AgentID)
		}
		assert.True(t, ids["sess-snap-1"])
		assert.True(t, ids["sess-snap-2"])
	})
}
