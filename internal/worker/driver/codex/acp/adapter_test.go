package acp

import (
	"encoding/json"
	"testing"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationHandlers_McpToolCallLifecycle(t *testing.T) {
	a := &Adapter{}

	started := notificationHandlers[methodItemStarted](a, rawJSON(t, map[string]any{
		"item": map[string]any{
			"id":        "mcp-1",
			"type":      "mcpToolCall",
			"server":    "flowgentic",
			"toolName":  "plan_commit",
			"arguments": map[string]any{"foo": "bar"},
		},
	}))
	require.Len(t, started, 1)
	require.NotNil(t, started[0].ToolCall)
	assert.Equal(t, acpsdk.ToolCallId("mcp-1"), started[0].ToolCall.ToolCallId)
	assert.Equal(t, "flowgentic.plan_commit", started[0].ToolCall.Title)
	assert.Equal(t, acpsdk.ToolCallStatusInProgress, started[0].ToolCall.Status)

	progress := notificationHandlers[methodMCPToolCallProgress](a, rawJSON(t, map[string]any{
		"itemId":   "mcp-1",
		"progress": "running",
	}))
	require.Len(t, progress, 1)
	require.NotNil(t, progress[0].ToolCallUpdate)
	assert.Equal(t, acpsdk.ToolCallId("mcp-1"), progress[0].ToolCallUpdate.ToolCallId)
	require.NotNil(t, progress[0].ToolCallUpdate.Status)
	assert.Equal(t, acpsdk.ToolCallStatusInProgress, *progress[0].ToolCallUpdate.Status)

	completed := notificationHandlers[methodItemCompleted](a, rawJSON(t, map[string]any{
		"item": map[string]any{
			"id":     "mcp-1",
			"type":   "mcpToolCall",
			"result": map[string]any{"ok": true},
		},
	}))
	require.Len(t, completed, 1)
	require.NotNil(t, completed[0].ToolCallUpdate)
	assert.Equal(t, acpsdk.ToolCallId("mcp-1"), completed[0].ToolCallUpdate.ToolCallId)
	require.NotNil(t, completed[0].ToolCallUpdate.Status)
	assert.Equal(t, acpsdk.ToolCallStatusCompleted, *completed[0].ToolCallUpdate.Status)
}

func TestNotificationHandlers_McpStartupUpdate(t *testing.T) {
	a := &Adapter{}

	updates := notificationHandlers[methodMCPStartupUpdate](a, rawJSON(t, map[string]any{
		"server":  "flowgentic",
		"message": "connected",
	}))
	require.Len(t, updates, 1)
	require.NotNil(t, updates[0].AgentThoughtChunk)
	assert.Contains(t, updates[0].AgentThoughtChunk.Content.Text.Text, "[mcp startup]")
	assert.Contains(t, updates[0].AgentThoughtChunk.Content.Text.Text, "flowgentic")
}

func rawJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}
