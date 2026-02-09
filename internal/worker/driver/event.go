package driver

import (
	"encoding/json"
	"time"
)

// EventType identifies the kind of normalized event.
type EventType string

const (
	EventTypeMessage      EventType = "message"
	EventTypeToolStart    EventType = "tool_start"
	EventTypeToolResult   EventType = "tool_result"
	EventTypeThinking     EventType = "thinking"
	EventTypeTurnComplete EventType = "turn_complete"
	EventTypeSessionStart EventType = "session_start"
	EventTypeError        EventType = "error"
	EventTypeCostUpdate   EventType = "cost_update"
)

// Event is the normalized event type all drivers produce.
type Event struct {
	Type       EventType       `json:"type"`
	Timestamp  time.Time       `json:"timestamp"`
	SessionID  string          `json:"session_id,omitempty"`
	Agent      string          `json:"agent_id"`
	Text       string          `json:"text,omitempty"`
	Delta      bool            `json:"delta,omitempty"`
	ToolName   string          `json:"tool_name,omitempty"`
	ToolID     string          `json:"tool_id,omitempty"`
	ToolInput  json.RawMessage `json:"tool_input,omitempty"`
	ToolError  bool            `json:"tool_error,omitempty"`
	StopReason string          `json:"stop_reason,omitempty"`
	Cost       *CostInfo       `json:"cost,omitempty"`
	Error      string          `json:"error,omitempty"`
	Raw        json.RawMessage `json:"raw,omitempty"`
}

// CostInfo tracks token usage and cost.
type CostInfo struct {
	InputTokens  int     `json:"input_tokens,omitempty"`
	OutputTokens int     `json:"output_tokens,omitempty"`
	TotalCostUSD float64 `json:"total_cost_usd,omitempty"`
	DurationMs   int     `json:"duration_ms,omitempty"`
	NumTurns     int     `json:"num_turns,omitempty"`
}

// EventCallback is called by drivers when events occur.
type EventCallback func(Event)

// HookEvent is the raw event received from an agent hook via RPC.
type HookEvent struct {
	SessionID string `json:"session_id"`
	Agent     string `json:"agent"`
	HookName  string `json:"hook_name"`
	Payload   []byte `json:"payload"`
}
