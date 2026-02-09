package opencode

import (
	"encoding/json"
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
	assert.Equal(t, "opencode", d.Agent())
}

func TestDriver_Capabilities(t *testing.T) {
	d := NewDriver(DriverDeps{Log: testLogger()})
	caps := d.Capabilities()
	assert.True(t, caps.Has(driver.CapStreaming))
	assert.True(t, caps.Has(driver.CapSessionResume))
	assert.True(t, caps.Has(driver.CapCostTracking))
	assert.True(t, caps.Has(driver.CapCustomModel))
	assert.True(t, caps.Has(driver.CapSystemPrompt))
	assert.True(t, caps.Has(driver.CapYolo))
}

func TestNormalizeSSEEvent_Message(t *testing.T) {
	raw, _ := json.Marshal(sseEvent{
		Type: "message",
		Data: json.RawMessage(`{"content":"hello world"}`),
	})
	events := normalizeSSEEvent(raw)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeMessage, events[0].Type)
	assert.Equal(t, "hello world", events[0].Text)
	assert.True(t, events[0].Delta)
}

func TestNormalizeSSEEvent_ToolStart(t *testing.T) {
	raw, _ := json.Marshal(sseEvent{
		Type: "tool.start",
		Data: json.RawMessage(`{"name":"Read","id":"t1","input":{"path":"/tmp"}}`),
	})
	events := normalizeSSEEvent(raw)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeToolStart, events[0].Type)
	assert.Equal(t, "Read", events[0].ToolName)
	assert.Equal(t, "t1", events[0].ToolID)
}

func TestNormalizeSSEEvent_ToolResult(t *testing.T) {
	raw, _ := json.Marshal(sseEvent{
		Type: "tool.result",
		Data: json.RawMessage(`{"id":"t1","is_error":true}`),
	})
	events := normalizeSSEEvent(raw)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeToolResult, events[0].Type)
	assert.True(t, events[0].ToolError)
}

func TestNormalizeSSEEvent_TurnComplete(t *testing.T) {
	raw, _ := json.Marshal(sseEvent{Type: "turn.complete"})
	events := normalizeSSEEvent(raw)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeTurnComplete, events[0].Type)
}

func TestNormalizeSSEEvent_Error(t *testing.T) {
	raw, _ := json.Marshal(sseEvent{
		Type: "error",
		Data: json.RawMessage(`"connection timeout"`),
	})
	events := normalizeSSEEvent(raw)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeError, events[0].Type)
}

func TestNormalizeSSEEvent_Unknown(t *testing.T) {
	raw := []byte(`{"type":"unknown"}`)
	events := normalizeSSEEvent(raw)
	assert.Nil(t, events)
}

func TestExtractURL(t *testing.T) {
	t.Run("http URL", func(t *testing.T) {
		assert.Equal(t, "http://localhost:8080", extractURL("Server started at http://localhost:8080"))
	})

	t.Run("https URL", func(t *testing.T) {
		assert.Equal(t, "https://example.com/api", extractURL("Listening on https://example.com/api for requests"))
	})

	t.Run("no URL", func(t *testing.T) {
		assert.Equal(t, "", extractURL("no url here"))
	})
}

func TestDriver_HandleHookEvent_UnknownSession(t *testing.T) {
	d := NewDriver(DriverDeps{Log: testLogger()})
	err := d.HandleHookEvent(nil, "nonexistent", driver.HookEvent{})
	assert.ErrorContains(t, err, "session not found")
}
