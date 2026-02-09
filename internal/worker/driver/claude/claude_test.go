package claude

import (
	"log/slog"
	"os"
	"testing"

	"github.com/sebastianm/flowgentic/internal/worker/driver"
	"github.com/stretchr/testify/assert"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestDriver_ID(t *testing.T) {
	d := NewDriver(DriverDeps{Log: testLogger()})
	assert.Equal(t, "claude-code", d.Agent())
}

func TestDriver_Capabilities(t *testing.T) {
	d := NewDriver(DriverDeps{Log: testLogger()})
	caps := d.Capabilities()

	assert.Equal(t, "claude-code", caps.Agent)
	assert.True(t, caps.Has(driver.CapStreaming))
	assert.True(t, caps.Has(driver.CapSessionResume))
	assert.True(t, caps.Has(driver.CapCostTracking))
	assert.True(t, caps.Has(driver.CapCustomModel))
	assert.True(t, caps.Has(driver.CapSystemPrompt))
	assert.True(t, caps.Has(driver.CapYolo))
}

func TestDriver_HandleHookEvent_UnknownSession(t *testing.T) {
	d := NewDriver(DriverDeps{Log: testLogger()})

	err := d.HandleHookEvent(nil, "nonexistent", driver.HookEvent{
		SessionID: "nonexistent",
		HookName:  "Stop",
	})
	assert.ErrorContains(t, err, "session not found")
}

func TestBuildFlags(t *testing.T) {
	t.Run("basic flags", func(t *testing.T) {
		flags := buildFlags(driver.LaunchOpts{
			Mode:   driver.SessionModeHeadless,
			Prompt: "test",
		}, "")
		assert.Contains(t, flags, "--output-format")
		assert.Contains(t, flags, "stream-json")
	})

	t.Run("all optional flags with resume", func(t *testing.T) {
		flags := buildFlags(driver.LaunchOpts{
			Mode:         driver.SessionModeHeadless,
			Model:        "sonnet",
			SystemPrompt: "be helpful",
			AllowedTools: []string{"Read", "Write"},
			Yolo:         true,
			SessionID:    "sess-123",
			Cwd:          "/tmp/test",
		}, "sess-123")
		assert.Contains(t, flags, "--model")
		assert.Contains(t, flags, "sonnet")
		assert.Contains(t, flags, "--system-prompt")
		assert.Contains(t, flags, "be helpful")
		assert.Contains(t, flags, "--allowed-tools")
		assert.Contains(t, flags, "Read,Write")
		assert.Contains(t, flags, "--dangerously-skip-permissions")
		assert.Contains(t, flags, "--resume")
		assert.Contains(t, flags, "--session-id")
		assert.Contains(t, flags, "sess-123")
		assert.Contains(t, flags, "--add-dir")
		assert.Contains(t, flags, "/tmp/test")
	})

	t.Run("new session with generated agent session ID", func(t *testing.T) {
		flags := buildFlags(driver.LaunchOpts{
			Mode: driver.SessionModeHeadless,
		}, "generated-uuid")
		assert.Contains(t, flags, "--session-id")
		assert.Contains(t, flags, "generated-uuid")
		assert.NotContains(t, flags, "--resume")
	})
}
