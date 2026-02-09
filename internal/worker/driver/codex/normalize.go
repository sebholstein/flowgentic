package codex

import (
	"encoding/json"
	"time"

	"github.com/sebastianm/flowgentic/internal/worker/driver"
)

const agent = "codex"

// codexEvent represents a top-level JSONL event from `codex exec --json`.
// All fields live at the top level; the schema varies by Type.
type codexEvent struct {
	Type     string          `json:"type"`
	ThreadID string          `json:"thread_id,omitempty"`
	Item     *codexItem      `json:"item,omitempty"`
	Usage    json.RawMessage `json:"usage,omitempty"`
}

// codexItem represents the nested "item" object in item.started / item.completed events.
type codexItem struct {
	ID               string `json:"id"`
	Type             string `json:"type"`
	Text             string `json:"text,omitempty"`
	Command          string `json:"command,omitempty"`
	AggregatedOutput string `json:"aggregated_output,omitempty"`
	ExitCode         *int   `json:"exit_code,omitempty"`
	Status           string `json:"status,omitempty"`
}

// normalizeCodexEvent converts a codex JSONL event into normalized driver Events.
func normalizeCodexEvent(raw []byte) []driver.Event {
	var evt codexEvent
	if err := json.Unmarshal(raw, &evt); err != nil {
		return nil
	}

	now := time.Now()

	switch evt.Type {
	case "thread.started":
		return []driver.Event{{
			Type:      driver.EventTypeSessionStart,
			Timestamp: now,
			Agent:     agent,
			Text:      evt.ThreadID,
		}}

	case "turn.completed":
		return []driver.Event{{
			Type:      driver.EventTypeTurnComplete,
			Timestamp: now,
			Agent:     agent,
		}}

	case "item.started":
		if evt.Item == nil {
			return nil
		}
		switch evt.Item.Type {
		case "command_execution":
			return []driver.Event{{
				Type:      driver.EventTypeToolStart,
				Timestamp: now,
				Agent:     agent,
				ToolName:  "command_execution",
				ToolID:    evt.Item.ID,
				Text:      evt.Item.Command,
			}}
		}

	case "item.completed":
		if evt.Item == nil {
			return nil
		}
		switch evt.Item.Type {
		case "agent_message":
			return []driver.Event{{
				Type:      driver.EventTypeMessage,
				Timestamp: now,
				Agent:     agent,
				Text:      evt.Item.Text,
			}}
		case "reasoning":
			return []driver.Event{{
				Type:      driver.EventTypeThinking,
				Timestamp: now,
				Agent:     agent,
				Text:      evt.Item.Text,
			}}
		case "command_execution":
			isError := evt.Item.ExitCode != nil && *evt.Item.ExitCode != 0
			return []driver.Event{{
				Type:      driver.EventTypeToolResult,
				Timestamp: now,
				Agent:     agent,
				ToolID:    evt.Item.ID,
				ToolError: isError,
				Text:      evt.Item.AggregatedOutput,
			}}
		}

	case "error":
		return []driver.Event{{
			Type:      driver.EventTypeError,
			Timestamp: now,
			Agent:     agent,
			Error:     string(raw),
			Raw:       raw,
		}}
	}

	return nil
}
