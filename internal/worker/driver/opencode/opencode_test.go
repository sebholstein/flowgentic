package opencode

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

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

func TestParseModel(t *testing.T) {
	t.Run("provider/model", func(t *testing.T) {
		p, m := parseModel("anthropic/claude-sonnet-4-20250514")
		assert.Equal(t, "anthropic", p)
		assert.Equal(t, "claude-sonnet-4-20250514", m)
	})

	t.Run("model only", func(t *testing.T) {
		p, m := parseModel("gpt-4o")
		assert.Equal(t, "", p)
		assert.Equal(t, "gpt-4o", m)
	})

	t.Run("nested slash", func(t *testing.T) {
		p, m := parseModel("openai/gpt-4o/2024")
		assert.Equal(t, "openai", p)
		assert.Equal(t, "gpt-4o/2024", m)
	})

	t.Run("empty string", func(t *testing.T) {
		p, m := parseModel("")
		assert.Equal(t, "", p)
		assert.Equal(t, "", m)
	})
}

// --- SSE event normalization tests ---

func TestNormalizeSSEEvent_MessagePartUpdated_Text(t *testing.T) {
	raw, _ := json.Marshal(sseEvent{
		Type:       "message.part.updated",
		Properties: json.RawMessage(`{"part":{"type":"text","text":"hello world"}}`),
	})
	events := normalizeSSEEvent(raw)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeMessage, events[0].Type)
	assert.Equal(t, "hello world", events[0].Text)
	assert.True(t, events[0].Delta)
}

func TestNormalizeSSEEvent_MessagePartUpdated_ToolRunning(t *testing.T) {
	raw, _ := json.Marshal(sseEvent{
		Type:       "message.part.updated",
		Properties: json.RawMessage(`{"part":{"type":"tool","callID":"t1","tool":"read","state":{"status":"running","input":{"filePath":"/tmp"}}}}`),
	})
	events := normalizeSSEEvent(raw)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeToolStart, events[0].Type)
	assert.Equal(t, "read", events[0].ToolName)
	assert.Equal(t, "t1", events[0].ToolID)
	assert.JSONEq(t, `{"filePath":"/tmp"}`, string(events[0].ToolInput))
}

func TestNormalizeSSEEvent_MessagePartUpdated_ToolCompleted(t *testing.T) {
	raw, _ := json.Marshal(sseEvent{
		Type:       "message.part.updated",
		Properties: json.RawMessage(`{"part":{"type":"tool","callID":"t1","tool":"read","state":{"status":"completed","input":{}}}}`),
	})
	events := normalizeSSEEvent(raw)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeToolResult, events[0].Type)
	assert.Equal(t, "t1", events[0].ToolID)
	assert.False(t, events[0].ToolError)
}

func TestNormalizeSSEEvent_MessagePartUpdated_ToolError(t *testing.T) {
	raw, _ := json.Marshal(sseEvent{
		Type:       "message.part.updated",
		Properties: json.RawMessage(`{"part":{"type":"tool","callID":"t2","tool":"read","state":{"status":"error","input":{}}}}`),
	})
	events := normalizeSSEEvent(raw)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeToolResult, events[0].Type)
	assert.Equal(t, "t2", events[0].ToolID)
	assert.True(t, events[0].ToolError)
}

func TestNormalizeSSEEvent_MessagePartUpdated_StepFinish(t *testing.T) {
	raw, _ := json.Marshal(sseEvent{
		Type:       "message.part.updated",
		Properties: json.RawMessage(`{"part":{"type":"step-finish","cost":0.006966,"tokens":{"input":1000,"output":200}}}`),
	})
	events := normalizeSSEEvent(raw)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeCostUpdate, events[0].Type)
	require.NotNil(t, events[0].Cost)
	assert.InDelta(t, 0.006966, events[0].Cost.TotalCostUSD, 0.0001)
	assert.Equal(t, 1000, events[0].Cost.InputTokens)
	assert.Equal(t, 200, events[0].Cost.OutputTokens)
}

func TestNormalizeSSEEvent_MessagePartUpdated_Reasoning(t *testing.T) {
	raw, _ := json.Marshal(sseEvent{
		Type:       "message.part.updated",
		Properties: json.RawMessage(`{"part":{"type":"reasoning","text":"thinking about it..."}}`),
	})
	events := normalizeSSEEvent(raw)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeThinking, events[0].Type)
	assert.Equal(t, "thinking about it...", events[0].Text)
}

func TestNormalizeSSEEvent_MessageUpdated_Cost(t *testing.T) {
	raw, _ := json.Marshal(sseEvent{
		Type:       "message.updated",
		Properties: json.RawMessage(`{"info":{"cost":{"inputTokens":1000,"outputTokens":500,"totalCost":0.015}}}`),
	})
	events := normalizeSSEEvent(raw)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeCostUpdate, events[0].Type)
	require.NotNil(t, events[0].Cost)
	assert.Equal(t, 1000, events[0].Cost.InputTokens)
	assert.Equal(t, 500, events[0].Cost.OutputTokens)
	assert.InDelta(t, 0.015, events[0].Cost.TotalCostUSD, 0.0001)
}

func TestNormalizeSSEEvent_MessageUpdated_NoCost(t *testing.T) {
	raw, _ := json.Marshal(sseEvent{
		Type:       "message.updated",
		Properties: json.RawMessage(`{"info":{"cost":{"inputTokens":0,"outputTokens":0,"totalCost":0}}}`),
	})
	events := normalizeSSEEvent(raw)
	assert.Nil(t, events)
}

func TestNormalizeSSEEvent_SessionStatus_Idle(t *testing.T) {
	raw, _ := json.Marshal(sseEvent{
		Type:       "session.status",
		Properties: json.RawMessage(`{"sessionID":"ses_abc","status":{"type":"idle"}}`),
	})
	events := normalizeSSEEvent(raw)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeTurnComplete, events[0].Type)
}

func TestNormalizeSSEEvent_SessionStatus_Busy(t *testing.T) {
	raw, _ := json.Marshal(sseEvent{
		Type:       "session.status",
		Properties: json.RawMessage(`{"sessionID":"ses_abc","status":{"type":"busy"}}`),
	})
	events := normalizeSSEEvent(raw)
	assert.Nil(t, events)
}

func TestNormalizeSSEEvent_SessionIdle(t *testing.T) {
	raw, _ := json.Marshal(sseEvent{
		Type:       "session.idle",
		Properties: json.RawMessage(`{"sessionID":"ses_abc"}`),
	})
	events := normalizeSSEEvent(raw)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeTurnComplete, events[0].Type)
}

func TestNormalizeSSEEvent_SessionUpdated_Error(t *testing.T) {
	raw, _ := json.Marshal(sseEvent{
		Type:       "session.updated",
		Properties: json.RawMessage(`{"info":{"error":"rate limit exceeded"}}`),
	})
	events := normalizeSSEEvent(raw)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeError, events[0].Type)
	assert.Equal(t, "rate limit exceeded", events[0].Error)
}

func TestNormalizeSSEEvent_SessionUpdated_NoError(t *testing.T) {
	raw, _ := json.Marshal(sseEvent{
		Type:       "session.updated",
		Properties: json.RawMessage(`{"info":{}}`),
	})
	events := normalizeSSEEvent(raw)
	assert.Nil(t, events)
}

func TestNormalizeSSEEvent_Unknown(t *testing.T) {
	raw := []byte(`{"type":"unknown"}`)
	events := normalizeSSEEvent(raw)
	assert.Nil(t, events)
}

func TestDriver_HandleHookEvent_UnknownSession(t *testing.T) {
	d := NewDriver(DriverDeps{Log: testLogger()})
	err := d.HandleHookEvent(context.TODO(), "nonexistent", driver.HookEvent{})
	assert.ErrorContains(t, err, "session not found")
}

func TestWaitForHealthy(t *testing.T) {
	t.Run("healthy on first try", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/global/health" {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		err := waitForHealthy(context.Background(), srv.URL, 2*time.Second)
		assert.NoError(t, err)
	})

	t.Run("healthy after retries", func(t *testing.T) {
		calls := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/global/health" {
				calls++
				if calls < 3 {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		err := waitForHealthy(context.Background(), srv.URL, 5*time.Second)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, calls, 3)
	})

	t.Run("context cancelled", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer srv.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := waitForHealthy(ctx, srv.URL, 5*time.Second)
		assert.Error(t, err)
	})
}

func TestFindFreePort(t *testing.T) {
	port, err := findFreePort()
	require.NoError(t, err)
	assert.Greater(t, port, 0)
	assert.Less(t, port, 65536)

	// Verify port is usable by briefly listening on it.
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err)
	l.Close()
}
