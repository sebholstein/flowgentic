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
	assert.False(t, caps.Has(driver.CapSessionResume))
	assert.True(t, caps.Has(driver.CapCustomModel))
	assert.True(t, caps.Has(driver.CapYolo))
	assert.True(t, caps.Has(driver.CapSystemPrompt))
}

func TestNormalizeNotification_AgentMessageDelta(t *testing.T) {
	params, _ := json.Marshal(map[string]any{
		"threadId": "t1",
		"itemId":   "item_1",
		"delta":    "Hello ",
	})
	events := normalizeNotification("item/agentMessage/delta", params)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeMessage, events[0].Type)
	assert.Equal(t, "Hello ", events[0].Text)
	assert.True(t, events[0].Delta)
}

func TestNormalizeNotification_ReasoningDelta(t *testing.T) {
	params, _ := json.Marshal(map[string]any{
		"threadId": "t1",
		"delta":    "thinking...",
	})
	events := normalizeNotification("item/reasoning/textDelta", params)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeThinking, events[0].Type)
	assert.Equal(t, "thinking...", events[0].Text)
	assert.True(t, events[0].Delta)
}

func TestNormalizeNotification_ItemStarted_CommandExecution(t *testing.T) {
	params, _ := json.Marshal(map[string]any{
		"threadId": "t1",
		"item": map[string]any{
			"id":      "item_1",
			"type":    "commandExecution",
			"command": "ls -la",
		},
	})
	events := normalizeNotification("item/started", params)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeToolStart, events[0].Type)
	assert.Equal(t, "command_execution", events[0].ToolName)
	assert.Equal(t, "item_1", events[0].ToolID)
	assert.Equal(t, "ls -la", events[0].Text)
}

func TestNormalizeNotification_ItemCompleted_AgentMessage(t *testing.T) {
	params, _ := json.Marshal(map[string]any{
		"threadId": "t1",
		"item": map[string]any{
			"id":   "item_2",
			"type": "agentMessage",
			"text": "The capital of France is Paris.",
		},
	})
	events := normalizeNotification("item/completed", params)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeMessage, events[0].Type)
	assert.Equal(t, "The capital of France is Paris.", events[0].Text)
	assert.False(t, events[0].Delta)
}

func TestNormalizeNotification_ItemCompleted_Reasoning(t *testing.T) {
	params, _ := json.Marshal(map[string]any{
		"threadId": "t1",
		"item": map[string]any{
			"id":   "item_0",
			"type": "reasoning",
			"text": "thinking...",
		},
	})
	events := normalizeNotification("item/completed", params)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeThinking, events[0].Type)
	assert.Equal(t, "thinking...", events[0].Text)
}

func TestNormalizeNotification_ItemCompleted_CommandExecution(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		exitCode := 0
		params, _ := json.Marshal(map[string]any{
			"threadId": "t1",
			"item": map[string]any{
				"id":               "item_1",
				"type":             "commandExecution",
				"command":          "echo hi",
				"aggregatedOutput": "hi\n",
				"exitCode":         exitCode,
				"status":           "completed",
				"cwd":              "/tmp",
				"commandActions":   []any{},
			},
		})
		events := normalizeNotification("item/completed", params)
		require.Len(t, events, 1)
		assert.Equal(t, driver.EventTypeToolResult, events[0].Type)
		assert.Equal(t, "item_1", events[0].ToolID)
		assert.False(t, events[0].ToolError)
		assert.Equal(t, "hi\n", events[0].Text)
	})

	t.Run("failure", func(t *testing.T) {
		exitCode := 1
		params, _ := json.Marshal(map[string]any{
			"threadId": "t1",
			"item": map[string]any{
				"id":               "item_1",
				"type":             "commandExecution",
				"aggregatedOutput": "error",
				"exitCode":         exitCode,
				"status":           "completed",
				"cwd":              "/tmp",
				"commandActions":   []any{},
			},
		})
		events := normalizeNotification("item/completed", params)
		require.Len(t, events, 1)
		assert.True(t, events[0].ToolError)
	})
}

func TestNormalizeNotification_ItemCompleted_FileChange(t *testing.T) {
	params, _ := json.Marshal(map[string]any{
		"threadId": "t1",
		"item": map[string]any{
			"id":     "item_3",
			"type":   "fileChange",
			"status": "completed",
			"changes": []map[string]any{
				{"path": "main.go", "diff": "+new line", "kind": map[string]string{"type": "update"}},
			},
		},
	})
	events := normalizeNotification("item/completed", params)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeToolResult, events[0].Type)
	assert.Equal(t, "item_3", events[0].ToolID)
	assert.Equal(t, "main.go\n+new line", events[0].Text)
}

func TestNormalizeNotification_TurnCompleted(t *testing.T) {
	params, _ := json.Marshal(map[string]any{
		"threadId": "t1",
	})
	events := normalizeNotification("turn/completed", params)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeTurnComplete, events[0].Type)
}

func TestNormalizeNotification_Error(t *testing.T) {
	params := json.RawMessage(`{"message":"something went wrong"}`)
	events := normalizeNotification("error", params)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeError, events[0].Type)
}

func TestNormalizeNotification_Unknown(t *testing.T) {
	events := normalizeNotification("turn/started", json.RawMessage(`{}`))
	assert.Nil(t, events)
}

func TestExtractThreadID(t *testing.T) {
	t.Run("present", func(t *testing.T) {
		params := json.RawMessage(`{"threadId":"abc-123","delta":"hi"}`)
		assert.Equal(t, "abc-123", extractThreadID(params))
	})

	t.Run("missing", func(t *testing.T) {
		params := json.RawMessage(`{"delta":"hi"}`)
		assert.Equal(t, "", extractThreadID(params))
	})

	t.Run("empty", func(t *testing.T) {
		assert.Equal(t, "", extractThreadID(nil))
	})
}

