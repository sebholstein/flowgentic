package acp

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"sync"
	"testing"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeUpdateSender struct {
	mu      sync.Mutex
	updates []acpsdk.SessionNotification
}

func (f *fakeUpdateSender) SessionUpdate(_ context.Context, n acpsdk.SessionNotification) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updates = append(f.updates, n)
	return nil
}

func (f *fakeUpdateSender) allUpdates() []acpsdk.SessionNotification {
	f.mu.Lock()
	defer f.mu.Unlock()
	cloned := make([]acpsdk.SessionNotification, len(f.updates))
	copy(cloned, f.updates)
	return cloned
}

type fakeBridge struct {
	threadID          string
	modelState        *acpsdk.SessionModelState
	availableCommands []acpsdk.AvailableCommand
	requestResult     json.RawMessage
	done              chan struct{}
}

func (f *fakeBridge) start(context.Context, map[string]string) error { return nil }
func (f *fakeBridge) threadStart(string, string, string, string, []acpsdk.McpServer) (string, error) {
	return f.threadID, nil
}
func (f *fakeBridge) turnStart(string, string, string, string) (string, error) { return "turn-1", nil }
func (f *fakeBridge) turnInterrupt(string, string) error                       { return nil }
func (f *fakeBridge) respondToServerRequest(int64, any)                        {}
func (f *fakeBridge) request(string, any) (json.RawMessage, error)             { return f.requestResult, nil }
func (f *fakeBridge) modelSnapshot() *acpsdk.SessionModelState                 { return f.modelState }
func (f *fakeBridge) availableCommandsSnapshot() []acpsdk.AvailableCommand {
	return append([]acpsdk.AvailableCommand(nil), f.availableCommands...)
}
func (f *fakeBridge) doneChan() <-chan struct{} {
	if f.done == nil {
		f.done = make(chan struct{})
	}
	return f.done
}
func (f *fakeBridge) close() {}

func newCodexTestAdapter() (*Adapter, *fakeUpdateSender) {
	updater := &fakeUpdateSender{}
	return &Adapter{
		log:                slog.New(slog.NewTextHandler(io.Discard, nil)),
		updater:            updater,
		pendingPermissions: make(map[string]pendingPermission),
	}, updater
}

func TestNewSession_EmitsStartupAvailableCommandsUpdate(t *testing.T) {
	a, updater := newCodexTestAdapter()
	fakeSrv := &fakeBridge{
		threadID: "thread-1",
		availableCommands: []acpsdk.AvailableCommand{
			{Name: "init", Description: "create AGENTS.md"},
			{Name: "review", Description: "review local changes"},
		},
	}
	a.bridgeFactory = func(_ *slog.Logger, _ func(threadID string, method string, params json.RawMessage, serverRequestID *int64)) bridgeClient {
		return fakeSrv
	}

	resp, err := a.NewSession(context.Background(), acpsdk.NewSessionRequest{Cwd: "/tmp"})
	require.NoError(t, err)
	assert.Equal(t, acpsdk.SessionId("thread-1"), resp.SessionId)

	updates := updater.allUpdates()
	require.Len(t, updates, 1)
	require.NotNil(t, updates[0].Update.AvailableCommandsUpdate)
	require.Len(t, updates[0].Update.AvailableCommandsUpdate.AvailableCommands, 2)
	assert.Equal(t, "init", updates[0].Update.AvailableCommandsUpdate.AvailableCommands[0].Name)
}

func TestNewSession_NoStartupCommandsNoUpdate(t *testing.T) {
	a, updater := newCodexTestAdapter()
	fakeSrv := &fakeBridge{
		threadID: "thread-1",
	}
	a.bridgeFactory = func(_ *slog.Logger, _ func(threadID string, method string, params json.RawMessage, serverRequestID *int64)) bridgeClient {
		return fakeSrv
	}

	_, err := a.NewSession(context.Background(), acpsdk.NewSessionRequest{Cwd: "/tmp"})
	require.NoError(t, err)
	assert.Empty(t, updater.allUpdates())
}

func TestDispatchNotification_SkillsUpdateRefreshesAvailableCommands(t *testing.T) {
	a, updater := newCodexTestAdapter()
	refreshed := map[string]any{
		"data": []any{
			map[string]any{
				"skills": []any{
					map[string]any{"name": "vercel-react-best-practices", "description": "React guidance"},
				},
			},
		},
	}
	raw, err := json.Marshal(refreshed)
	require.NoError(t, err)

	a.mu.Lock()
	a.threadID = "thread-1"
	a.server = &fakeBridge{requestResult: raw}
	a.mu.Unlock()

	a.dispatchNotification("thread-1", methodSkillsUpdated, rawJSON(t, map[string]any{"type": "skills_update_available"}), nil)

	updates := updater.allUpdates()
	require.Len(t, updates, 1)
	require.NotNil(t, updates[0].Update.AvailableCommandsUpdate)
	require.Len(t, updates[0].Update.AvailableCommandsUpdate.AvailableCommands, 1)
	assert.Equal(t, "vercel-react-best-practices", updates[0].Update.AvailableCommandsUpdate.AvailableCommands[0].Name)
}

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
