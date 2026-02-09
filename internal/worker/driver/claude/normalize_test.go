package claude

import (
	"encoding/json"
	"testing"

	"github.com/sebastianm/flowgentic/internal/worker/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T { return &v }

func TestNormalizeStreamEvent_MessageStart(t *testing.T) {
	resp := StreamingResponse{
		Type: "stream_event",
		Event: &Event{
			Type: "message_start",
		},
	}
	events := normalizeStreamingResponse(resp)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeSessionStart, events[0].Type)
}

func TestNormalizeStreamEvent_TextDelta(t *testing.T) {
	resp := StreamingResponse{
		Type: "stream_event",
		Event: &Event{
			Type: "content_block_delta",
			Delta: &EventDelta{
				Type: "text_delta",
				Text: "hello world",
			},
		},
	}
	events := normalizeStreamingResponse(resp)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeMessage, events[0].Type)
	assert.Equal(t, "hello world", events[0].Text)
	assert.True(t, events[0].Delta)
}

func TestNormalizeStreamEvent_ToolStart(t *testing.T) {
	resp := StreamingResponse{
		Type: "stream_event",
		Event: &Event{
			Type: "content_block_start",
			ContentBlock: &ContentBlock{
				Type: "tool_use",
				Name: "Read",
				ID:   "tool_123",
			},
		},
	}
	events := normalizeStreamingResponse(resp)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeToolStart, events[0].Type)
	assert.Equal(t, "Read", events[0].ToolName)
	assert.Equal(t, "tool_123", events[0].ToolID)
}

func TestNormalizeStreamEvent_ThinkingStart(t *testing.T) {
	resp := StreamingResponse{
		Type: "stream_event",
		Event: &Event{
			Type: "content_block_start",
			ContentBlock: &ContentBlock{
				Type:     "thinking",
				Thinking: "I need to think about this...",
			},
		},
	}
	events := normalizeStreamingResponse(resp)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeThinking, events[0].Type)
	assert.Equal(t, "I need to think about this...", events[0].Text)
}

func TestNormalizeStreamEvent_MessageDelta_StopReason(t *testing.T) {
	resp := StreamingResponse{
		Type: "stream_event",
		Event: &Event{
			Type: "message_delta",
			Delta: &EventDelta{
				StopReason: ptr("end_turn"),
			},
		},
	}
	events := normalizeStreamingResponse(resp)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeTurnComplete, events[0].Type)
	assert.Equal(t, "end_turn", events[0].StopReason)
}

func TestNormalizeResult(t *testing.T) {
	cost := 0.05
	dur := 1234
	turns := 3
	resp := StreamingResponse{
		Type:         "result",
		Result:       "final result text",
		StopReason:   ptr("end_turn"),
		TotalCostUSD: &cost,
		DurationMs:   &dur,
		NumTurns:     &turns,
		Usage: &Usage{
			InputTokens:  100,
			OutputTokens: 200,
		},
	}
	events := normalizeStreamingResponse(resp)
	// Should produce: cost_update, turn_complete (message is skipped to avoid
	// duplicating the assistant message already emitted).
	require.Len(t, events, 2)

	assert.Equal(t, driver.EventTypeCostUpdate, events[0].Type)
	assert.InDelta(t, 0.05, events[0].Cost.TotalCostUSD, 0.001)
	assert.Equal(t, 100, events[0].Cost.InputTokens)
	assert.Equal(t, 200, events[0].Cost.OutputTokens)
	assert.Equal(t, 1234, events[0].Cost.DurationMs)
	assert.Equal(t, 3, events[0].Cost.NumTurns)

	assert.Equal(t, driver.EventTypeTurnComplete, events[1].Type)
	assert.Equal(t, "end_turn", events[1].StopReason)
}

func TestNormalizeResult_Error(t *testing.T) {
	resp := StreamingResponse{
		Type:    "result",
		Subtype: "error",
		Result:  "something went wrong",
		IsError: true,
	}
	events := normalizeStreamingResponse(resp)
	// error, turn_complete (message skipped to avoid duplicate)
	require.Len(t, events, 2)
	assert.Equal(t, driver.EventTypeError, events[0].Type)
	assert.Equal(t, "something went wrong", events[0].Error)
	assert.Equal(t, driver.EventTypeTurnComplete, events[1].Type)
}

func TestNormalizeAssistant(t *testing.T) {
	resp := StreamingResponse{
		Type: "assistant",
		Message: &Message{
			Content: []Content{
				{Type: "text", Text: "Hello!"},
				{Type: "tool_use", Name: "Write", ID: "t1", Input: json.RawMessage(`{"path":"/tmp/test"}`)},
				{Type: "tool_result", ToolUseID: "t1", Content: json.RawMessage(`"ok"`)},
			},
		},
	}
	events := normalizeStreamingResponse(resp)
	require.Len(t, events, 3)
	assert.Equal(t, driver.EventTypeMessage, events[0].Type)
	assert.Equal(t, driver.EventTypeToolStart, events[1].Type)
	assert.Equal(t, driver.EventTypeToolResult, events[2].Type)
}

func TestNormalizeSystem(t *testing.T) {
	resp := StreamingResponse{
		Type: "system",
		Message: &Message{
			Content: []Content{
				{Type: "text", Text: "System message"},
			},
		},
	}
	events := normalizeStreamingResponse(resp)
	require.Len(t, events, 1)
	assert.Equal(t, driver.EventTypeMessage, events[0].Type)
	assert.Equal(t, "System message", events[0].Text)
}

func TestNormalizeUnknownType(t *testing.T) {
	resp := StreamingResponse{Type: "unknown"}
	events := normalizeStreamingResponse(resp)
	assert.Nil(t, events)
}

func TestNormalizeStreamEvent_NilEvent(t *testing.T) {
	resp := StreamingResponse{Type: "stream_event"}
	events := normalizeStreamingResponse(resp)
	assert.Nil(t, events)
}
