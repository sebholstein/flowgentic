package session

import (
	"encoding/json"
	"fmt"
	"strings"

	controlplanev1 "github.com/sebastianm/flowgentic/internal/proto/gen/controlplane/v1"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
)

// SessionEventRecord is the JSON-serializable form persisted to SQLite.
// String enum values match ACP naming exactly.
type SessionEventRecord struct {
	V         int    `json:"v"`          // Schema version (1)
	SessionID string `json:"session_id"`
	Sequence  int64  `json:"seq"`
	Timestamp string `json:"ts"`
	Type      string `json:"type"` // ACP names: "agent_message_chunk", "tool_call", etc.

	Text       string `json:"text,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	Title      string `json:"title,omitempty"`
	Kind       string `json:"kind,omitempty"`      // ACP: "read", "edit", "execute", etc.
	RawInput   string `json:"raw_input,omitempty"`
	RawOutput  string `json:"raw_output,omitempty"`
	Status     string `json:"status,omitempty"` // ACP: "in_progress", "completed", "failed"
	ModeID     string `json:"mode_id,omitempty"`

	Locations []LocationRecord     `json:"locations,omitempty"`
	Content   []ContentBlockRecord `json:"content,omitempty"`
}

// LocationRecord is a JSON-serializable tool call location.
type LocationRecord struct {
	Path string `json:"path"`
	Line int64  `json:"line,omitempty"`
}

// ContentBlockRecord is a JSON-serializable tool call content block.
type ContentBlockRecord struct {
	Type    string `json:"type"`              // "diff" or "text"
	Path    string `json:"path,omitempty"`    // diff only
	NewText string `json:"new_text,omitempty"` // diff only
	OldText string `json:"old_text,omitempty"` // diff only
	Text    string `json:"text,omitempty"`     // text only
}

const eventRecordVersion = 1

// --- Write path: worker proto → JSON record ---

// WorkerEventToRecord converts a worker-side SessionEvent to a JSON-serializable record.
func WorkerEventToRecord(e *workerv1.SessionEvent) SessionEventRecord {
	r := SessionEventRecord{
		V:         eventRecordVersion,
		SessionID: e.GetSessionId(),
		Sequence:  e.GetSequence(),
		Timestamp: e.GetTimestamp(),
	}

	switch p := e.Payload.(type) {
	case *workerv1.SessionEvent_AgentMessageChunk:
		r.Type = "agent_message_chunk"
		r.Text = p.AgentMessageChunk.GetText()
	case *workerv1.SessionEvent_AgentThoughtChunk:
		r.Type = "agent_thought_chunk"
		r.Text = p.AgentThoughtChunk.GetText()
	case *workerv1.SessionEvent_ToolCall:
		r.Type = "tool_call"
		tc := p.ToolCall
		r.ToolCallID = tc.GetToolCallId()
		r.Title = tc.GetTitle()
		r.Kind = toolCallKindToString(tc.GetKind())
		r.RawInput = tc.GetRawInput()
		r.Status = toolCallStatusToString(tc.GetStatus())
		r.Locations = locationsToRecord(tc.GetLocations())
		r.Content = contentBlocksToRecord(tc.GetContent())
	case *workerv1.SessionEvent_ToolCallUpdate:
		r.Type = "tool_call_update"
		tc := p.ToolCallUpdate
		r.ToolCallID = tc.GetToolCallId()
		r.Title = tc.GetTitle()
		r.Status = toolCallStatusToString(tc.GetStatus())
		r.RawOutput = tc.GetRawOutput()
		r.Locations = locationsToRecord(tc.GetLocations())
		r.Content = contentBlocksToRecord(tc.GetContent())
	case *workerv1.SessionEvent_StatusChange:
		r.Type = "status_change"
		r.Status = p.StatusChange.GetStatus().String()
	case *workerv1.SessionEvent_CurrentModeUpdate:
		r.Type = "current_mode_update"
		r.ModeID = p.CurrentModeUpdate.GetModeId()
	case *workerv1.SessionEvent_UserMessage:
		r.Type = "user_message"
		r.Text = p.UserMessage.GetText()
	default:
		r.Type = "unknown"
	}

	return r
}

// --- Read path: JSON record → CP proto ---

// RecordToCPEvent converts a JSON record to a CP-side SessionEvent.
func RecordToCPEvent(r SessionEventRecord) *controlplanev1.SessionEvent {
	e := &controlplanev1.SessionEvent{
		SessionId: r.SessionID,
		Sequence:  r.Sequence,
		Timestamp: r.Timestamp,
	}

	switch r.Type {
	case "agent_message_chunk":
		e.Payload = &controlplanev1.SessionEvent_AgentMessageChunk{
			AgentMessageChunk: &controlplanev1.AgentMessageChunk{Text: r.Text},
		}
	case "agent_thought_chunk":
		e.Payload = &controlplanev1.SessionEvent_AgentThoughtChunk{
			AgentThoughtChunk: &controlplanev1.AgentThoughtChunk{Text: r.Text},
		}
	case "tool_call":
		tc := &controlplanev1.ToolCall{
			ToolCallId: r.ToolCallID,
			Title:      r.Title,
			Kind:       stringToToolCallKind(r.Kind),
			RawInput:   r.RawInput,
			Status:     stringToToolCallStatus(r.Status),
		}
		for _, loc := range r.Locations {
			tc.Locations = append(tc.Locations, &controlplanev1.ToolCallLocation{
				Path: loc.Path,
				Line: loc.Line,
			})
		}
		tc.Content = recordContentBlocksToCP(r.Content)
		e.Payload = &controlplanev1.SessionEvent_ToolCall{ToolCall: tc}
	case "tool_call_update":
		tc := &controlplanev1.ToolCallUpdate{
			ToolCallId: r.ToolCallID,
			Title:      r.Title,
			Status:     stringToToolCallStatus(r.Status),
			RawOutput:  r.RawOutput,
		}
		for _, loc := range r.Locations {
			tc.Locations = append(tc.Locations, &controlplanev1.ToolCallLocation{
				Path: loc.Path,
				Line: loc.Line,
			})
		}
		tc.Content = recordContentBlocksToCP(r.Content)
		e.Payload = &controlplanev1.SessionEvent_ToolCallUpdate{ToolCallUpdate: tc}
	case "status_change":
		e.Payload = &controlplanev1.SessionEvent_StatusChange{
			StatusChange: &controlplanev1.StatusChange{Status: r.Status},
		}
	case "current_mode_update":
		e.Payload = &controlplanev1.SessionEvent_CurrentModeUpdate{
			CurrentModeUpdate: &controlplanev1.CurrentModeUpdate{ModeId: r.ModeID},
		}
	case "user_message":
		e.Payload = &controlplanev1.SessionEvent_UserMessage{
			UserMessage: &controlplanev1.UserMessage{Text: r.Text},
		}
	}

	return e
}

// MarshalRecord serializes a SessionEventRecord to JSON bytes.
func MarshalRecord(r SessionEventRecord) ([]byte, error) {
	return json.Marshal(r)
}

// UnmarshalRecord deserializes JSON bytes into a SessionEventRecord.
func UnmarshalRecord(data []byte) (SessionEventRecord, error) {
	var r SessionEventRecord
	if err := json.Unmarshal(data, &r); err != nil {
		return r, fmt.Errorf("unmarshal event record: %w", err)
	}
	return r, nil
}

// --- Enum string converters ---

func toolCallKindToString(k workerv1.ToolCallKind) string {
	switch k {
	case workerv1.ToolCallKind_TOOL_CALL_KIND_READ:
		return "read"
	case workerv1.ToolCallKind_TOOL_CALL_KIND_EDIT:
		return "edit"
	case workerv1.ToolCallKind_TOOL_CALL_KIND_DELETE:
		return "delete"
	case workerv1.ToolCallKind_TOOL_CALL_KIND_MOVE:
		return "move"
	case workerv1.ToolCallKind_TOOL_CALL_KIND_SEARCH:
		return "search"
	case workerv1.ToolCallKind_TOOL_CALL_KIND_EXECUTE:
		return "execute"
	case workerv1.ToolCallKind_TOOL_CALL_KIND_THINK:
		return "think"
	case workerv1.ToolCallKind_TOOL_CALL_KIND_FETCH:
		return "fetch"
	case workerv1.ToolCallKind_TOOL_CALL_KIND_OTHER:
		return "other"
	default:
		return "other"
	}
}

func toolCallStatusToString(s workerv1.ToolCallStatus) string {
	switch s {
	case workerv1.ToolCallStatus_TOOL_CALL_STATUS_IN_PROGRESS:
		return "in_progress"
	case workerv1.ToolCallStatus_TOOL_CALL_STATUS_COMPLETED:
		return "completed"
	case workerv1.ToolCallStatus_TOOL_CALL_STATUS_FAILED:
		return "failed"
	default:
		return "in_progress"
	}
}

func stringToToolCallKind(s string) controlplanev1.ToolCallKind {
	switch strings.ToLower(s) {
	case "read":
		return controlplanev1.ToolCallKind_TOOL_CALL_KIND_READ
	case "edit":
		return controlplanev1.ToolCallKind_TOOL_CALL_KIND_EDIT
	case "delete":
		return controlplanev1.ToolCallKind_TOOL_CALL_KIND_DELETE
	case "move":
		return controlplanev1.ToolCallKind_TOOL_CALL_KIND_MOVE
	case "search":
		return controlplanev1.ToolCallKind_TOOL_CALL_KIND_SEARCH
	case "execute":
		return controlplanev1.ToolCallKind_TOOL_CALL_KIND_EXECUTE
	case "think":
		return controlplanev1.ToolCallKind_TOOL_CALL_KIND_THINK
	case "fetch":
		return controlplanev1.ToolCallKind_TOOL_CALL_KIND_FETCH
	case "other":
		return controlplanev1.ToolCallKind_TOOL_CALL_KIND_OTHER
	default:
		return controlplanev1.ToolCallKind_TOOL_CALL_KIND_OTHER
	}
}

func stringToToolCallStatus(s string) controlplanev1.ToolCallStatus {
	switch strings.ToLower(s) {
	case "in_progress":
		return controlplanev1.ToolCallStatus_TOOL_CALL_STATUS_IN_PROGRESS
	case "completed":
		return controlplanev1.ToolCallStatus_TOOL_CALL_STATUS_COMPLETED
	case "failed":
		return controlplanev1.ToolCallStatus_TOOL_CALL_STATUS_FAILED
	default:
		return controlplanev1.ToolCallStatus_TOOL_CALL_STATUS_IN_PROGRESS
	}
}

// --- Helper converters ---

func locationsToRecord(locs []*workerv1.ToolCallLocation) []LocationRecord {
	if len(locs) == 0 {
		return nil
	}
	out := make([]LocationRecord, len(locs))
	for i, l := range locs {
		out[i] = LocationRecord{Path: l.GetPath(), Line: l.GetLine()}
	}
	return out
}

func contentBlocksToRecord(blocks []*workerv1.ToolCallContentBlock) []ContentBlockRecord {
	if len(blocks) == 0 {
		return nil
	}
	out := make([]ContentBlockRecord, 0, len(blocks))
	for _, b := range blocks {
		switch cb := b.Block.(type) {
		case *workerv1.ToolCallContentBlock_Diff:
			out = append(out, ContentBlockRecord{
				Type:    "diff",
				Path:    cb.Diff.GetPath(),
				NewText: cb.Diff.GetNewText(),
				OldText: cb.Diff.GetOldText(),
			})
		case *workerv1.ToolCallContentBlock_Text:
			out = append(out, ContentBlockRecord{
				Type: "text",
				Text: cb.Text.GetText(),
			})
		}
	}
	return out
}

func recordContentBlocksToCP(blocks []ContentBlockRecord) []*controlplanev1.ToolCallContentBlock {
	if len(blocks) == 0 {
		return nil
	}
	out := make([]*controlplanev1.ToolCallContentBlock, 0, len(blocks))
	for _, b := range blocks {
		switch b.Type {
		case "diff":
			out = append(out, &controlplanev1.ToolCallContentBlock{
				Block: &controlplanev1.ToolCallContentBlock_Diff{
					Diff: &controlplanev1.ToolCallDiff{
						Path:    b.Path,
						NewText: b.NewText,
						OldText: b.OldText,
					},
				},
			})
		case "text":
			out = append(out, &controlplanev1.ToolCallContentBlock{
				Block: &controlplanev1.ToolCallContentBlock_Text{
					Text: &controlplanev1.ToolCallText{
						Text: b.Text,
					},
				},
			})
		}
	}
	return out
}
