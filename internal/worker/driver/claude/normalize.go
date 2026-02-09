package claude

import (
	"encoding/json"
	"time"

	"github.com/sebastianm/flowgentic/internal/worker/driver"
)

const agent = "claude-code"

// normalizeStreamingResponse converts a Claude Code StreamingResponse into
// zero or more normalized driver Events.
func normalizeStreamingResponse(resp StreamingResponse) []driver.Event {
	now := time.Now()

	switch resp.Type {
	case "stream_event":
		return normalizeStreamEvent(resp, now)
	case "result":
		return normalizeResult(resp, now)
	case "system":
		return normalizeSystem(resp, now)
	case "assistant":
		return normalizeAssistant(resp, now)
	default:
		return nil
	}
}

func normalizeStreamEvent(resp StreamingResponse, now time.Time) []driver.Event {
	if resp.Event == nil {
		return nil
	}

	switch resp.Event.Type {
	case "message_start":
		return []driver.Event{{
			Type:      driver.EventTypeSessionStart,
			Timestamp: now,
			Agent:     agent,
		}}

	case "content_block_start":
		if resp.Event.ContentBlock == nil {
			return nil
		}
		cb := resp.Event.ContentBlock
		switch cb.Type {
		case "tool_use":
			raw, _ := json.Marshal(resp)
			return []driver.Event{{
				Type:      driver.EventTypeToolStart,
				Timestamp: now,
				Agent:     agent,
				ToolName:  cb.Name,
				ToolID:    cb.ID,
				Raw:       raw,
			}}
		case "thinking":
			if cb.Thinking != "" {
				return []driver.Event{{
					Type:      driver.EventTypeThinking,
					Timestamp: now,
					Agent:     agent,
					Text:      cb.Thinking,
				}}
			}
		}
		return nil

	case "content_block_delta":
		if resp.Event.Delta == nil {
			return nil
		}
		d := resp.Event.Delta
		switch d.Type {
		case "text_delta":
			return []driver.Event{{
				Type:      driver.EventTypeMessage,
				Timestamp: now,
				Agent:     agent,
				Text:      d.Text,
				Delta:     true,
			}}
		case "thinking_delta":
			return []driver.Event{{
				Type:      driver.EventTypeThinking,
				Timestamp: now,
				Agent:     agent,
				Text:      d.Text,
				Delta:     true,
			}}
		}
		return nil

	case "message_delta":
		if resp.Event.Delta != nil && resp.Event.Delta.StopReason != nil {
			return []driver.Event{{
				Type:       driver.EventTypeTurnComplete,
				Timestamp:  now,
				Agent:      agent,
				StopReason: *resp.Event.Delta.StopReason,
			}}
		}
		return nil

	default:
		return nil
	}
}

func normalizeResult(resp StreamingResponse, now time.Time) []driver.Event {
	var events []driver.Event

	// Skip emitting a message for Result text — it duplicates the assistant
	// message content that was already emitted via normalizeAssistant.

	if resp.IsError {
		events = append(events, driver.Event{
			Type:      driver.EventTypeError,
			Timestamp: now,
			Agent:     agent,
			Error:     resp.Result,
		})
	}

	if resp.TotalCostUSD != nil || resp.Usage != nil || resp.DurationMs != nil {
		cost := &driver.CostInfo{}
		if resp.TotalCostUSD != nil {
			cost.TotalCostUSD = *resp.TotalCostUSD
		}
		if resp.DurationMs != nil {
			cost.DurationMs = *resp.DurationMs
		}
		if resp.NumTurns != nil {
			cost.NumTurns = *resp.NumTurns
		}
		if resp.Usage != nil {
			cost.InputTokens = resp.Usage.InputTokens
			cost.OutputTokens = resp.Usage.OutputTokens
		}
		events = append(events, driver.Event{
			Type:      driver.EventTypeCostUpdate,
			Timestamp: now,
			Agent:     agent,
			Cost:      cost,
		})
	}

	stopReason := ""
	if resp.StopReason != nil {
		stopReason = *resp.StopReason
	}
	events = append(events, driver.Event{
		Type:       driver.EventTypeTurnComplete,
		Timestamp:  now,
		Agent:      agent,
		StopReason: stopReason,
	})

	return events
}

func normalizeSystem(resp StreamingResponse, now time.Time) []driver.Event {
	if resp.Message == nil {
		return nil
	}
	var text string
	for _, c := range resp.Message.Content {
		if c.Type == "text" {
			text += c.Text
		}
	}
	if text == "" {
		return nil
	}
	return []driver.Event{{
		Type:      driver.EventTypeMessage,
		Timestamp: now,
		Agent:     agent,
		Text:      text,
	}}
}

func normalizeAssistant(resp StreamingResponse, now time.Time) []driver.Event {
	if resp.Message == nil {
		return nil
	}

	var events []driver.Event
	for _, c := range resp.Message.Content {
		switch c.Type {
		case "text":
			if c.Text != "" {
				events = append(events, driver.Event{
					Type:      driver.EventTypeMessage,
					Timestamp: now,
					Agent:     agent,
					Text:      c.Text,
				})
			}
		case "tool_use":
			events = append(events, driver.Event{
				Type:      driver.EventTypeToolStart,
				Timestamp: now,
				Agent:     agent,
				ToolName:  c.Name,
				ToolID:    c.ID,
				ToolInput: c.Input,
			})
		case "tool_result":
			events = append(events, driver.Event{
				Type:      driver.EventTypeToolResult,
				Timestamp: now,
				Agent:     agent,
				ToolID:    c.ToolUseID,
				ToolError: c.IsError,
				Raw:       c.Content,
			})
		case "thinking":
			if c.Thinking != "" {
				events = append(events, driver.Event{
					Type:      driver.EventTypeThinking,
					Timestamp: now,
					Agent:     agent,
					Text:      c.Thinking,
				})
			}
		}
	}
	return events
}
