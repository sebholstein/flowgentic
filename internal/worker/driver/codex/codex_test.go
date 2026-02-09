package codex

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
	assert.Equal(t, "codex", d.Agent())
}

func TestDriver_Capabilities(t *testing.T) {
	d := NewDriver(DriverDeps{Log: testLogger()})
	caps := d.Capabilities()
	assert.True(t, caps.Has(driver.CapStreaming))
	assert.True(t, caps.Has(driver.CapSessionResume))
	assert.True(t, caps.Has(driver.CapCustomModel))
	assert.True(t, caps.Has(driver.CapYolo))
	assert.False(t, caps.Has(driver.CapSystemPrompt))
}

func TestNormalizeCodexEvent_ThreadStarted(t *testing.T) {
	raw := []byte(`{"type":"thread.started","thread_id":"t123"}`)
	events := normalizeCodexEvent(raw)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeSessionStart, events[0].Type)
	assert.Equal(t, "t123", events[0].Text)
}

func TestNormalizeCodexEvent_TurnCompleted(t *testing.T) {
	raw := []byte(`{"type":"turn.completed","usage":{"input_tokens":100}}`)
	events := normalizeCodexEvent(raw)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeTurnComplete, events[0].Type)
}

func TestNormalizeCodexEvent_ItemStarted_CommandExecution(t *testing.T) {
	raw := []byte(`{"type":"item.started","item":{"id":"item_1","type":"command_execution","command":"ls","status":"in_progress"}}`)
	events := normalizeCodexEvent(raw)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeToolStart, events[0].Type)
	assert.Equal(t, "command_execution", events[0].ToolName)
	assert.Equal(t, "item_1", events[0].ToolID)
	assert.Equal(t, "ls", events[0].Text)
}

func TestNormalizeCodexEvent_ItemCompleted_AgentMessage(t *testing.T) {
	raw := []byte(`{"type":"item.completed","item":{"id":"item_2","type":"agent_message","text":"The capital of France is Paris."}}`)
	events := normalizeCodexEvent(raw)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeMessage, events[0].Type)
	assert.Equal(t, "The capital of France is Paris.", events[0].Text)
}

func TestNormalizeCodexEvent_ItemCompleted_Reasoning(t *testing.T) {
	raw := []byte(`{"type":"item.completed","item":{"id":"item_0","type":"reasoning","text":"thinking..."}}`)
	events := normalizeCodexEvent(raw)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeThinking, events[0].Type)
	assert.Equal(t, "thinking...", events[0].Text)
}

func TestNormalizeCodexEvent_ItemCompleted_CommandExecution(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		exitCode := 0
		raw, _ := json.Marshal(codexEvent{
			Type: "item.completed",
			Item: &codexItem{
				ID:               "item_1",
				Type:             "command_execution",
				Command:          "echo hi",
				AggregatedOutput: "hi\n",
				ExitCode:         &exitCode,
				Status:           "completed",
			},
		})
		events := normalizeCodexEvent(raw)
		require.Len(t, events, 1)
		assert.Equal(t, driver.EventTypeToolResult, events[0].Type)
		assert.Equal(t, "item_1", events[0].ToolID)
		assert.False(t, events[0].ToolError)
		assert.Equal(t, "hi\n", events[0].Text)
	})

	t.Run("failure", func(t *testing.T) {
		exitCode := 1
		raw, _ := json.Marshal(codexEvent{
			Type: "item.completed",
			Item: &codexItem{
				ID:               "item_1",
				Type:             "command_execution",
				AggregatedOutput: "error",
				ExitCode:         &exitCode,
				Status:           "completed",
			},
		})
		events := normalizeCodexEvent(raw)
		require.Len(t, events, 1)
		assert.True(t, events[0].ToolError)
	})
}

func TestNormalizeCodexEvent_Error(t *testing.T) {
	raw := []byte(`{"type":"error","message":"something went wrong"}`)
	events := normalizeCodexEvent(raw)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeError, events[0].Type)
}

func TestNormalizeCodexEvent_Unknown(t *testing.T) {
	raw := []byte(`{"type":"turn.started"}`)
	events := normalizeCodexEvent(raw)
	assert.Nil(t, events)
}

func TestNormalizeCodexEvent_InvalidJSON(t *testing.T) {
	events := normalizeCodexEvent([]byte(`{invalid`))
	assert.Nil(t, events)
}

func TestSessionMetaParsing(t *testing.T) {
	// thread.started contains the session ID as thread_id at the top level.
	raw := []byte(`{"type":"thread.started","thread_id":"abc-123-def"}`)
	var evt codexEvent
	require.NoError(t, json.Unmarshal(raw, &evt))
	assert.Equal(t, "thread.started", evt.Type)
	assert.Equal(t, "abc-123-def", evt.ThreadID)
}

func TestDriver_ImplementsSessionResolver(t *testing.T) {
	d := NewDriver(DriverDeps{Log: testLogger()})
	var _ driver.SessionResolver = d
}
