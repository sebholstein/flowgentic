package gemini

import (
	"log/slog"
	"os"
	"testing"

	"github.com/sebastianm/flowgentic/internal/worker/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestDriver_ID(t *testing.T) {
	d := NewDriver(DriverDeps{Log: testLogger()})
	assert.Equal(t, "gemini", d.Agent())
}

func TestDriver_Capabilities(t *testing.T) {
	d := NewDriver(DriverDeps{Log: testLogger()})
	caps := d.Capabilities()
	assert.True(t, caps.Has(driver.CapCustomModel))
	assert.True(t, caps.Has(driver.CapYolo))
	assert.False(t, caps.Has(driver.CapStreaming))
	assert.False(t, caps.Has(driver.CapSessionResume))
}

func TestDriver_HandleHookEvent_AfterAgent(t *testing.T) {
	d := NewDriver(DriverDeps{Log: testLogger()})

	// Manually register a session so HandleHookEvent can find it.
	sess := &geminiSession{
		info: driver.SessionInfo{
			ID:      "hook-test",
			AgentID: agent,
			Status:  driver.SessionStatusRunning,
			Mode:    driver.SessionModeHeadless,
		},
		driver:  d,
		onEvent: func(_ driver.Event) {},
		done:    make(chan struct{}),
	}
	d.mu.Lock()
	d.sessions["hook-test"] = sess
	d.mu.Unlock()

	err := d.HandleHookEvent(nil, "hook-test", driver.HookEvent{
		SessionID: "hook-test",
		HookName:  "AfterAgent",
	})
	require.NoError(t, err)
	assert.Equal(t, driver.SessionStatusIdle, sess.info.Status)
}

func TestDriver_HandleHookEvent_UnknownSession(t *testing.T) {
	d := NewDriver(DriverDeps{Log: testLogger()})
	err := d.HandleHookEvent(nil, "nonexistent", driver.HookEvent{})
	assert.ErrorContains(t, err, "session not found")
}

func TestParseLatestSessionID(t *testing.T) {
	t.Run("parses UUID from typical output", func(t *testing.T) {
		output := `1. Fix authentication bug (2 hours ago) [550e8400-e29b-41d4-a716-446655440000]
2. Add tests (1 day ago) [660f9500-f30c-52e5-b827-557766551111]`
		id, err := parseLatestSessionID(output)
		require.NoError(t, err)
		assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", id)
	})

	t.Run("returns error for empty output", func(t *testing.T) {
		_, err := parseLatestSessionID("")
		assert.ErrorContains(t, err, "no session UUID found")
	})

	t.Run("returns error for output without brackets", func(t *testing.T) {
		_, err := parseLatestSessionID("no brackets here")
		assert.ErrorContains(t, err, "no session UUID found")
	})
}
