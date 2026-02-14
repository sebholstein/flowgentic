package session

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
)

func TestRoundTrip_AgentMessageChunk(t *testing.T) {
	event := &workerv1.SessionEvent{
		SessionId: "sess-1",
		Sequence:  1,
		Timestamp: "2024-01-01T00:00:00Z",
		Payload: &workerv1.SessionEvent_AgentMessageChunk{
			AgentMessageChunk: &workerv1.AgentMessageChunk{Text: "hello world"},
		},
	}

	record := WorkerEventToRecord(event)
	assert.Equal(t, 1, record.V)
	assert.Equal(t, "agent_message_chunk", record.Type)
	assert.Equal(t, "hello world", record.Text)

	data, err := MarshalRecord(record)
	require.NoError(t, err)

	restored, err := UnmarshalRecord(data)
	require.NoError(t, err)

	cpEvent := RecordToCPEvent(restored)
	assert.Equal(t, "sess-1", cpEvent.SessionId)
	assert.Equal(t, int64(1), cpEvent.Sequence)
	assert.Equal(t, "hello world", cpEvent.GetAgentMessageChunk().Text)
}

func TestRoundTrip_ToolCall(t *testing.T) {
	event := &workerv1.SessionEvent{
		SessionId: "sess-1",
		Sequence:  2,
		Timestamp: "2024-01-01T00:00:01Z",
		Payload: &workerv1.SessionEvent_ToolCall{
			ToolCall: &workerv1.ToolCall{
				ToolCallId: "tc-1",
				Title:      "Read file.go",
				Kind:       workerv1.ToolCallKind_TOOL_CALL_KIND_READ,
				RawInput:   `{"path":"file.go"}`,
				Status:     workerv1.ToolCallStatus_TOOL_CALL_STATUS_IN_PROGRESS,
				Locations: []*workerv1.ToolCallLocation{
					{Path: "file.go", Line: 42},
				},
				Content: []*workerv1.ToolCallContentBlock{
					{Block: &workerv1.ToolCallContentBlock_Text{Text: &workerv1.ToolCallText{Text: "content"}}},
					{Block: &workerv1.ToolCallContentBlock_Diff{Diff: &workerv1.ToolCallDiff{Path: "a.go", NewText: "new", OldText: "old"}}},
				},
			},
		},
	}

	record := WorkerEventToRecord(event)
	assert.Equal(t, "tool_call", record.Type)
	assert.Equal(t, "tc-1", record.ToolCallID)
	assert.Equal(t, "read", record.Kind)
	assert.Equal(t, "in_progress", record.Status)
	assert.Len(t, record.Locations, 1)
	assert.Len(t, record.Content, 2)

	data, err := MarshalRecord(record)
	require.NoError(t, err)

	restored, err := UnmarshalRecord(data)
	require.NoError(t, err)

	cpEvent := RecordToCPEvent(restored)
	assert.Equal(t, "sess-1", cpEvent.SessionId)
	assert.Equal(t, int64(2), cpEvent.Sequence)
	assert.NotNil(t, cpEvent.GetToolCall())
	assert.Equal(t, "tc-1", cpEvent.GetToolCall().ToolCallId)
	assert.Equal(t, "Read file.go", cpEvent.GetToolCall().Title)
	assert.Len(t, cpEvent.GetToolCall().Locations, 1)
	assert.Len(t, cpEvent.GetToolCall().Content, 2)
}

func TestRoundTrip_UserMessage(t *testing.T) {
	event := &workerv1.SessionEvent{
		SessionId: "sess-1",
		Sequence:  3,
		Timestamp: "2024-01-01T00:00:02Z",
		Payload: &workerv1.SessionEvent_UserMessage{
			UserMessage: &workerv1.UserMessage{Text: "please fix the bug"},
		},
	}

	record := WorkerEventToRecord(event)
	assert.Equal(t, "user_message", record.Type)
	assert.Equal(t, "please fix the bug", record.Text)

	data, err := MarshalRecord(record)
	require.NoError(t, err)

	restored, err := UnmarshalRecord(data)
	require.NoError(t, err)

	cpEvent := RecordToCPEvent(restored)
	assert.Equal(t, "sess-1", cpEvent.SessionId)
	assert.Equal(t, "please fix the bug", cpEvent.GetUserMessage().Text)
}

func TestRoundTrip_CurrentModeUpdate(t *testing.T) {
	event := &workerv1.SessionEvent{
		SessionId: "sess-1",
		Sequence:  4,
		Timestamp: "2024-01-01T00:00:03Z",
		Payload: &workerv1.SessionEvent_CurrentModeUpdate{
			CurrentModeUpdate: &workerv1.CurrentModeUpdate{ModeId: "architect"},
		},
	}

	record := WorkerEventToRecord(event)
	assert.Equal(t, "current_mode_update", record.Type)
	assert.Equal(t, "architect", record.ModeID)

	data, err := MarshalRecord(record)
	require.NoError(t, err)

	restored, err := UnmarshalRecord(data)
	require.NoError(t, err)

	cpEvent := RecordToCPEvent(restored)
	assert.Equal(t, "architect", cpEvent.GetCurrentModeUpdate().ModeId)
}

func TestRoundTrip_ToolCallUpdate(t *testing.T) {
	event := &workerv1.SessionEvent{
		SessionId: "sess-1",
		Sequence:  5,
		Timestamp: "2024-01-01T00:00:04Z",
		Payload: &workerv1.SessionEvent_ToolCallUpdate{
			ToolCallUpdate: &workerv1.ToolCallUpdate{
				ToolCallId: "tc-1",
				Title:      "Read file.go",
				Status:     workerv1.ToolCallStatus_TOOL_CALL_STATUS_COMPLETED,
				RawOutput:  "file contents here",
				Content: []*workerv1.ToolCallContentBlock{
					{Block: &workerv1.ToolCallContentBlock_Text{Text: &workerv1.ToolCallText{Text: "output"}}},
				},
			},
		},
	}

	record := WorkerEventToRecord(event)
	assert.Equal(t, "tool_call_update", record.Type)
	assert.Equal(t, "completed", record.Status)
	assert.Equal(t, "file contents here", record.RawOutput)
	assert.Len(t, record.Content, 1)

	data, err := MarshalRecord(record)
	require.NoError(t, err)

	restored, err := UnmarshalRecord(data)
	require.NoError(t, err)

	cpEvent := RecordToCPEvent(restored)
	assert.Equal(t, "tc-1", cpEvent.GetToolCallUpdate().ToolCallId)
	assert.Len(t, cpEvent.GetToolCallUpdate().Content, 1)
}
