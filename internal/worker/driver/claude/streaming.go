package claude

import "encoding/json"

// StreamingResponse represents a Claude Code streaming JSON response.
type StreamingResponse struct {
	Type       string   `json:"type"`
	Subtype    string   `json:"subtype,omitempty"`
	Event      *Event   `json:"event,omitempty"`
	SessionID  string   `json:"session_id,omitempty"`
	UUID       string   `json:"uuid,omitempty"`
	Message    *Message `json:"message,omitempty"`
	Result     string   `json:"result,omitempty"`
	IsError    bool     `json:"is_error,omitempty"`
	StopReason *string  `json:"stop_reason,omitempty"`

	DurationMs    *int     `json:"duration_ms,omitempty"`
	DurationApiMs *int     `json:"duration_api_ms,omitempty"`
	NumTurns      *int     `json:"num_turns,omitempty"`
	TotalCostUSD  *float64 `json:"total_cost_usd,omitempty"`
	Usage         *Usage   `json:"usage,omitempty"`

	ParentToolUseID *string `json:"parent_tool_use_id,omitempty"`
}

type Message struct {
	ID         string    `json:"id,omitempty"`
	Model      string    `json:"model,omitempty"`
	Type       string    `json:"type,omitempty"`
	Role       string    `json:"role,omitempty"`
	Content    []Content `json:"content,omitempty"`
	StopReason *string   `json:"stop_reason,omitempty"`
	Usage      *Usage    `json:"usage,omitempty"`
}

type Content struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
	Thinking  string          `json:"thinking,omitempty"`
}

type Event struct {
	Type         string        `json:"type"`
	Index        *int          `json:"index,omitempty"`
	Message      *Message      `json:"message,omitempty"`
	ContentBlock *ContentBlock `json:"content_block,omitempty"`
	Delta        *EventDelta   `json:"delta,omitempty"`
	Usage        *Usage        `json:"usage,omitempty"`
}

type ContentBlock struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Thinking string `json:"thinking,omitempty"`
}

type EventDelta struct {
	Type         string  `json:"type,omitempty"`
	Text         string  `json:"text,omitempty"`
	PartialJSON  string  `json:"partial_json,omitempty"`
	StopReason   *string `json:"stop_reason,omitempty"`
	StopSequence *string `json:"stop_sequence,omitempty"`
}

type Usage struct {
	InputTokens              int `json:"input_tokens,omitempty"`
	OutputTokens             int `json:"output_tokens,omitempty"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}
