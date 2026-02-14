package acp

import (
	"context"
	"testing"

	acpsdk "github.com/coder/acp-go-sdk"
	claudecode "github.com/sebastianm/flowgentic/internal/claude-agent-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSessionID = acpsdk.SessionId("test-session")

func TestNormalizeAndSend_ResultMessage_ReturnsTrueToSignalCompletion(t *testing.T) {
	a, _ := newTestAdapter()
	ctx := context.Background()

	result := a.normalizeAndSend(ctx, testSessionID, &claudecode.ResultMessage{
		MessageType: "result",
		Subtype:     "success",
	})

	assert.True(t, result, "normalizeAndSend should return true for ResultMessage")
}

type staticModelProvider struct {
	state *acpsdk.SessionModelState
}

func (p staticModelProvider) SessionModelState(context.Context) (*acpsdk.SessionModelState, error) {
	return p.state, nil
}

func TestNewSession_ReturnsModelStateWhenProviderAvailable(t *testing.T) {
	a, _ := newTestAdapter()
	a.modelProvider = staticModelProvider{
		state: &acpsdk.SessionModelState{
			AvailableModels: []acpsdk.ModelInfo{
				{ModelId: "claude-sonnet"},
				{ModelId: "claude-opus"},
			},
			CurrentModelId: "claude-sonnet",
		},
	}

	resp, err := a.NewSession(context.Background(), acpsdk.NewSessionRequest{Cwd: "/tmp"})
	require.NoError(t, err)
	require.NotNil(t, resp.Models)
	assert.Equal(t, "claude-sonnet", string(resp.Models.CurrentModelId))
	require.Len(t, resp.Models.AvailableModels, 2)
	assert.Equal(t, "claude-sonnet", string(resp.Models.AvailableModels[0].ModelId))
}

func TestNewSession_DoesNotEmitAvailableCommandsOnStartup(t *testing.T) {
	a, fake := newTestAdapter()
	resp, err := a.NewSession(context.Background(), acpsdk.NewSessionRequest{Cwd: t.TempDir()})
	require.NoError(t, err)
	require.NotEmpty(t, resp.SessionId)

	updates := fake.allUpdates()
	require.Empty(t, updates)
}

func TestBuildSDKOptions_SetsAllSettingSources(t *testing.T) {
	a, _ := newTestAdapter()
	a.cwd = t.TempDir()

	opts := claudecode.NewOptions(a.buildSDKOptions()...)

	require.Equal(t, []claudecode.SettingSource{
		claudecode.SettingSourceUser,
		claudecode.SettingSourceProject,
		claudecode.SettingSourceLocal,
	}, opts.SettingSources)
}

func TestNormalizeAndSend_AssistantMessage_SkipsTextAndThinking(t *testing.T) {
	a, fake := newTestAdapter()
	ctx := context.Background()

	msg := &claudecode.AssistantMessage{
		MessageType: "assistant",
		Content: []claudecode.ContentBlock{
			&claudecode.TextBlock{
				MessageType: "text",
				Text:        "Hello world",
			},
			&claudecode.ThinkingBlock{
				MessageType: "thinking",
				Thinking:    "Let me think about this...",
			},
		},
	}

	result := a.normalizeAndSend(ctx, testSessionID, msg)

	assert.False(t, result, "normalizeAndSend should return false for AssistantMessage")
	updates := fake.allUpdates()
	for _, u := range updates {
		assert.Nil(t, u.Update.AgentMessageChunk, "should not send AgentMessageChunk for TextBlock (handled by stream events)")
		assert.Nil(t, u.Update.AgentThoughtChunk, "should not send AgentThoughtChunk for ThinkingBlock (handled by stream events)")
	}
}

func TestToolCallLifecycle_StreamThenBatch(t *testing.T) {
	a, fake := newTestAdapter()
	ctx := context.Background()

	// 1. StreamEvent content_block_start type=tool_use
	a.normalizeAndSend(ctx, testSessionID, &claudecode.StreamEvent{
		Event: map[string]any{
			"type": "content_block_start",
			"content_block": map[string]any{
				"type": "tool_use",
				"id":   "t1",
				"name": "Read",
			},
		},
	})

	// 2. AssistantMessage with ToolUseBlock (same id) — upgrades to in_progress.
	a.normalizeAndSend(ctx, testSessionID, &claudecode.AssistantMessage{
		MessageType: "assistant",
		Content: []claudecode.ContentBlock{
			&claudecode.ToolUseBlock{
				MessageType: "tool_use",
				ToolUseID:   "t1",
				Name:        "Read",
				Input:       map[string]any{"file_path": "/tmp/foo"},
			},
		},
	})

	// 3. Next AssistantMessage (triggers completeActiveTools for re-tracked t1)
	a.normalizeAndSend(ctx, testSessionID, &claudecode.AssistantMessage{
		MessageType: "assistant",
		Content: []claudecode.ContentBlock{
			&claudecode.TextBlock{
				MessageType: "text",
				Text:        "Here is the result",
			},
		},
	})

	updates := fake.allUpdates()
	// Expected sequence:
	// 0: StartToolCall(t1, "Read", pending) — from stream event
	// 1: UpdateToolCall(t1, in_progress+input+title) — from ToolUseBlock (upgrade)
	// 2: UpdateToolCall(t1, completed) — completeActiveTools at start of next AssistantMessage
	require.Len(t, updates, 3, "expected 3 updates")

	// Update 0: StartToolCall(t1, "Read", pending) with rich metadata
	u0 := updates[0].Update
	require.NotNil(t, u0.ToolCall, "first update should be ToolCall (start)")
	assert.Equal(t, acpsdk.ToolCallId("t1"), u0.ToolCall.ToolCallId)
	assert.Equal(t, "Read", u0.ToolCall.Title) // no input yet → default title
	assert.Equal(t, acpsdk.ToolCallStatusPending, u0.ToolCall.Status)
	assert.Equal(t, acpsdk.ToolKindRead, u0.ToolCall.Kind)
	assert.NotNil(t, u0.ToolCall.Meta, "should have _meta with claudeCode.toolName")

	// Update 1: UpdateToolCall(t1, in_progress) — upgrade with input & enriched title
	u1 := updates[1].Update
	require.NotNil(t, u1.ToolCallUpdate, "second update should be ToolCallUpdate (in_progress)")
	assert.Equal(t, acpsdk.ToolCallId("t1"), u1.ToolCallUpdate.ToolCallId)
	require.NotNil(t, u1.ToolCallUpdate.Status)
	assert.Equal(t, acpsdk.ToolCallStatusInProgress, *u1.ToolCallUpdate.Status)
	assert.NotNil(t, u1.ToolCallUpdate.RawInput, "upgrade should include raw input")
	require.NotNil(t, u1.ToolCallUpdate.Title)
	assert.Equal(t, "Read /tmp/foo", *u1.ToolCallUpdate.Title)
	require.NotNil(t, u1.ToolCallUpdate.Kind)
	assert.Equal(t, acpsdk.ToolKindRead, *u1.ToolCallUpdate.Kind)

	// Update 2: UpdateToolCall(t1, completed) — from completeActiveTools on next message
	u2 := updates[2].Update
	require.NotNil(t, u2.ToolCallUpdate, "third update should be ToolCallUpdate (completed)")
	assert.Equal(t, acpsdk.ToolCallId("t1"), u2.ToolCallUpdate.ToolCallId)
	require.NotNil(t, u2.ToolCallUpdate.Status)
	assert.Equal(t, acpsdk.ToolCallStatusCompleted, *u2.ToolCallUpdate.Status)

	// No AgentMessageChunk from the TextBlock in the last AssistantMessage.
	for _, u := range updates {
		assert.Nil(t, u.Update.AgentMessageChunk, "TextBlock should be skipped (handled by stream events)")
	}
}

func TestToolCallLifecycle_BatchOnly(t *testing.T) {
	a, fake := newTestAdapter()
	ctx := context.Background()

	// 1. AssistantMessage with ToolUseBlock (no prior stream event)
	a.normalizeAndSend(ctx, testSessionID, &claudecode.AssistantMessage{
		MessageType: "assistant",
		Content: []claudecode.ContentBlock{
			&claudecode.ToolUseBlock{
				MessageType: "tool_use",
				ToolUseID:   "t1",
				Name:        "Write",
				Input:       map[string]any{"file_path": "/tmp/bar", "content": "hello"},
			},
		},
	})

	// 2. Next AssistantMessage (triggers completion)
	a.normalizeAndSend(ctx, testSessionID, &claudecode.AssistantMessage{
		MessageType: "assistant",
		Content: []claudecode.ContentBlock{
			&claudecode.TextBlock{
				MessageType: "text",
				Text:        "Done",
			},
		},
	})

	updates := fake.allUpdates()
	require.GreaterOrEqual(t, len(updates), 2, "expected at least 2 updates")

	// Update 0: StartToolCall(t1, in_progress) — direct start with rich metadata
	u0 := updates[0].Update
	require.NotNil(t, u0.ToolCall, "first update should be ToolCall (start)")
	assert.Equal(t, acpsdk.ToolCallId("t1"), u0.ToolCall.ToolCallId)
	assert.Equal(t, "Write /tmp/bar", u0.ToolCall.Title)
	assert.Equal(t, acpsdk.ToolCallStatusInProgress, u0.ToolCall.Status)
	assert.Equal(t, acpsdk.ToolKindEdit, u0.ToolCall.Kind)
	assert.NotNil(t, u0.ToolCall.RawInput, "should have rawInput")
	// Write should have diff content
	require.Len(t, u0.ToolCall.Content, 1)
	require.NotNil(t, u0.ToolCall.Content[0].Diff)
	assert.Equal(t, "/tmp/bar", u0.ToolCall.Content[0].Diff.Path)
	// Write should have location
	require.Len(t, u0.ToolCall.Locations, 1)
	assert.Equal(t, "/tmp/bar", u0.ToolCall.Locations[0].Path)

	// Update 1: UpdateToolCall(t1, completed)
	u1 := updates[1].Update
	require.NotNil(t, u1.ToolCallUpdate, "second update should be ToolCallUpdate (completed)")
	assert.Equal(t, acpsdk.ToolCallId("t1"), u1.ToolCallUpdate.ToolCallId)
	require.NotNil(t, u1.ToolCallUpdate.Status)
	assert.Equal(t, acpsdk.ToolCallStatusCompleted, *u1.ToolCallUpdate.Status)
}

func TestToolCallLifecycle_MultipleTools(t *testing.T) {
	a, fake := newTestAdapter()
	ctx := context.Background()

	// 1. StreamEvent tool_use t1
	a.normalizeAndSend(ctx, testSessionID, &claudecode.StreamEvent{
		Event: map[string]any{
			"type": "content_block_start",
			"content_block": map[string]any{
				"type": "tool_use",
				"id":   "t1",
				"name": "Read",
			},
		},
	})

	// 2. StreamEvent tool_use t2
	a.normalizeAndSend(ctx, testSessionID, &claudecode.StreamEvent{
		Event: map[string]any{
			"type": "content_block_start",
			"content_block": map[string]any{
				"type": "tool_use",
				"id":   "t2",
				"name": "Write",
			},
		},
	})

	// 3. AssistantMessage with both ToolUseBlocks.
	a.normalizeAndSend(ctx, testSessionID, &claudecode.AssistantMessage{
		MessageType: "assistant",
		Content: []claudecode.ContentBlock{
			&claudecode.ToolUseBlock{
				MessageType: "tool_use",
				ToolUseID:   "t1",
				Name:        "Read",
				Input:       map[string]any{"file_path": "/a"},
			},
			&claudecode.ToolUseBlock{
				MessageType: "tool_use",
				ToolUseID:   "t2",
				Name:        "Write",
				Input:       map[string]any{"file_path": "/b", "content": "x"},
			},
		},
	})

	// 4. Next AssistantMessage (triggers completeActiveTools for t1, t2)
	a.normalizeAndSend(ctx, testSessionID, &claudecode.AssistantMessage{
		MessageType: "assistant",
		Content: []claudecode.ContentBlock{
			&claudecode.TextBlock{
				MessageType: "text",
				Text:        "All done",
			},
		},
	})

	updates := fake.allUpdates()

	// Collect events by type for each tool.
	var starts, inProgress, completions []acpsdk.ToolCallId
	for _, u := range updates {
		if u.Update.ToolCall != nil {
			starts = append(starts, u.Update.ToolCall.ToolCallId)
		}
		if u.Update.ToolCallUpdate != nil && u.Update.ToolCallUpdate.Status != nil {
			switch *u.Update.ToolCallUpdate.Status {
			case acpsdk.ToolCallStatusInProgress:
				inProgress = append(inProgress, u.Update.ToolCallUpdate.ToolCallId)
			case acpsdk.ToolCallStatusCompleted:
				completions = append(completions, u.Update.ToolCallUpdate.ToolCallId)
			}
		}
	}

	// Each tool started once (pending from stream).
	assert.Len(t, starts, 2, "each tool started once (pending from stream)")
	assert.Contains(t, starts, acpsdk.ToolCallId("t1"))
	assert.Contains(t, starts, acpsdk.ToolCallId("t2"))

	// Each tool upgraded once (in_progress from batch).
	assert.Len(t, inProgress, 2, "each tool upgraded once (in_progress from batch)")
	assert.Contains(t, inProgress, acpsdk.ToolCallId("t1"))
	assert.Contains(t, inProgress, acpsdk.ToolCallId("t2"))

	// Each tool completed once (from completeActiveTools on next message).
	assert.Len(t, completions, 2, "each tool completed once")
	assert.Contains(t, completions, acpsdk.ToolCallId("t1"))
	assert.Contains(t, completions, acpsdk.ToolCallId("t2"))
}

func TestToolCallLifecycle_CompletedOnResult(t *testing.T) {
	a, fake := newTestAdapter()
	ctx := context.Background()

	// 1. StreamEvent tool_use t1
	a.normalizeAndSend(ctx, testSessionID, &claudecode.StreamEvent{
		Event: map[string]any{
			"type": "content_block_start",
			"content_block": map[string]any{
				"type": "tool_use",
				"id":   "t1",
				"name": "Bash",
			},
		},
	})

	// 2. AssistantMessage with ToolUseBlock
	a.normalizeAndSend(ctx, testSessionID, &claudecode.AssistantMessage{
		MessageType: "assistant",
		Content: []claudecode.ContentBlock{
			&claudecode.ToolUseBlock{
				MessageType: "tool_use",
				ToolUseID:   "t1",
				Name:        "Bash",
				Input:       map[string]any{"command": "ls"},
			},
		},
	})

	// 3. ResultMessage (no next AssistantMessage)
	result := a.normalizeAndSend(ctx, testSessionID, &claudecode.ResultMessage{
		MessageType: "result",
		Subtype:     "success",
	})
	assert.True(t, result)

	updates := fake.allUpdates()

	// Find completion for t1.
	var completed bool
	for _, u := range updates {
		if u.Update.ToolCallUpdate != nil &&
			u.Update.ToolCallUpdate.ToolCallId == "t1" &&
			u.Update.ToolCallUpdate.Status != nil &&
			*u.Update.ToolCallUpdate.Status == acpsdk.ToolCallStatusCompleted {
			completed = true
		}
	}
	assert.True(t, completed, "t1 should be completed when ResultMessage arrives")
}

func TestConvertMCPServers(t *testing.T) {
	converted := convertMCPServers([]acpsdk.McpServer{
		{
			Stdio: &acpsdk.McpServerStdio{
				Name:    "flowgentic",
				Command: "agentctl",
				Args:    nil,
				Env: []acpsdk.EnvVariable{
					{Name: "AGENTCTL_AGENT_RUN_ID", Value: "sess-1"},
				},
			},
		},
		{
			Http: &acpsdk.McpServerHttp{
				Name: "remote-http",
				Url:  "https://example.com/mcp",
				Headers: []acpsdk.HttpHeader{
					{Name: "Authorization", Value: "Bearer token"},
				},
			},
		},
		{
			Sse: &acpsdk.McpServerSse{
				Name: "remote-sse",
				Url:  "https://example.com/sse",
			},
		},
	})

	require.Len(t, converted, 3)

	stdio, ok := converted["flowgentic"].(*claudecode.McpStdioServerConfig)
	require.True(t, ok)
	assert.Equal(t, "agentctl", stdio.Command)
	assert.Nil(t, stdio.Args)
	assert.Equal(t, map[string]string{"AGENTCTL_AGENT_RUN_ID": "sess-1"}, stdio.Env)

	httpCfg, ok := converted["remote-http"].(*claudecode.McpHTTPServerConfig)
	require.True(t, ok)
	assert.Equal(t, "https://example.com/mcp", httpCfg.URL)
	assert.Equal(t, map[string]string{"Authorization": "Bearer token"}, httpCfg.Headers)

	sseCfg, ok := converted["remote-sse"].(*claudecode.McpSSEServerConfig)
	require.True(t, ok)
	assert.Equal(t, "https://example.com/sse", sseCfg.URL)
}

func TestBuildSDKOptions_IncludesMCPServers(t *testing.T) {
	a, _ := newTestAdapter()
	a.mcpServers = map[string]claudecode.McpServerConfig{
		"flowgentic": &claudecode.McpStdioServerConfig{
			Type:    claudecode.McpServerTypeStdio,
			Command: "agentctl",
			Args:    nil,
		},
	}

	opts := claudecode.Options{}
	for _, opt := range a.buildSDKOptions() {
		opt(&opts)
	}

	require.Len(t, opts.McpServers, 1)
	_, ok := opts.McpServers["flowgentic"].(*claudecode.McpStdioServerConfig)
	assert.True(t, ok)
}

func TestIsAllowedInFlowgenticPlanMode(t *testing.T) {
	assert.True(t, isAllowedInFlowgenticPlanMode("mcp__flowgentic__set_topic"))
	assert.True(t, isAllowedInFlowgenticPlanMode("Write"))
	assert.True(t, isAllowedInFlowgenticPlanMode("Read"))

	assert.False(t, isAllowedInFlowgenticPlanMode("Bash"))
	assert.False(t, isAllowedInFlowgenticPlanMode("Task"))
	assert.False(t, isAllowedInFlowgenticPlanMode("AskUserQuestion"))
}

func TestToolCallLifecycle_ToolResultBlock(t *testing.T) {
	a, fake := newTestAdapter()
	ctx := context.Background()

	// Pre-track the tool as active (simulating it was started earlier).
	a.activeTools = map[string]string{"t1": "Read"}

	a.normalizeAndSend(ctx, testSessionID, &claudecode.AssistantMessage{
		MessageType: "assistant",
		Content: []claudecode.ContentBlock{
			&claudecode.ToolResultBlock{
				MessageType: "tool_result",
				ToolUseID:   "t1",
				Content:     "file contents here",
				IsError:     nil,
			},
		},
	})

	updates := fake.allUpdates()
	require.GreaterOrEqual(t, len(updates), 2, "expected at least 2 updates")

	// First update: completeActiveTools completes t1 (no output).
	u0 := updates[0].Update
	require.NotNil(t, u0.ToolCallUpdate)
	assert.Equal(t, acpsdk.ToolCallId("t1"), u0.ToolCallUpdate.ToolCallId)
	require.NotNil(t, u0.ToolCallUpdate.Status)
	assert.Equal(t, acpsdk.ToolCallStatusCompleted, *u0.ToolCallUpdate.Status)

	// Second update: ToolResultBlock sends completed with raw output.
	u1 := updates[1].Update
	require.NotNil(t, u1.ToolCallUpdate)
	assert.Equal(t, acpsdk.ToolCallId("t1"), u1.ToolCallUpdate.ToolCallId)
	require.NotNil(t, u1.ToolCallUpdate.Status)
	assert.Equal(t, acpsdk.ToolCallStatusCompleted, *u1.ToolCallUpdate.Status)
	assert.NotNil(t, u1.ToolCallUpdate.RawOutput, "ToolResultBlock update should include raw output")
}

func TestToolCallLifecycle_FailedToolResult(t *testing.T) {
	a, fake := newTestAdapter()
	ctx := context.Background()

	isError := true
	a.normalizeAndSend(ctx, testSessionID, &claudecode.AssistantMessage{
		MessageType: "assistant",
		Content: []claudecode.ContentBlock{
			&claudecode.ToolResultBlock{
				MessageType: "tool_result",
				ToolUseID:   "t1",
				Content:     "error: file not found",
				IsError:     &isError,
			},
		},
	})

	updates := fake.allUpdates()

	var found bool
	for _, u := range updates {
		if u.Update.ToolCallUpdate != nil &&
			u.Update.ToolCallUpdate.ToolCallId == "t1" &&
			u.Update.ToolCallUpdate.Status != nil &&
			*u.Update.ToolCallUpdate.Status == acpsdk.ToolCallStatusFailed {
			found = true
		}
	}
	assert.True(t, found, "should find a failed update for t1")
}

func TestStreamEvent_TextDelta(t *testing.T) {
	a, fake := newTestAdapter()
	ctx := context.Background()

	a.normalizeAndSend(ctx, testSessionID, &claudecode.StreamEvent{
		Event: map[string]any{
			"type": "content_block_delta",
			"delta": map[string]any{
				"type": "text_delta",
				"text": "hello ",
			},
		},
	})

	updates := fake.allUpdates()
	require.Len(t, updates, 1)
	require.NotNil(t, updates[0].Update.AgentMessageChunk, "should send AgentMessageChunk for text_delta")
}

func TestStreamEvent_ThinkingDelta(t *testing.T) {
	a, fake := newTestAdapter()
	ctx := context.Background()

	a.normalizeAndSend(ctx, testSessionID, &claudecode.StreamEvent{
		Event: map[string]any{
			"type": "content_block_delta",
			"delta": map[string]any{
				"type": "thinking_delta",
				"text": "I should consider...",
			},
		},
	})

	updates := fake.allUpdates()
	require.Len(t, updates, 1)
	require.NotNil(t, updates[0].Update.AgentThoughtChunk, "should send AgentThoughtChunk for thinking_delta")
}

func TestStreamEvent_ThinkingStart(t *testing.T) {
	a, fake := newTestAdapter()
	ctx := context.Background()

	a.normalizeAndSend(ctx, testSessionID, &claudecode.StreamEvent{
		Event: map[string]any{
			"type": "content_block_start",
			"content_block": map[string]any{
				"type":     "thinking",
				"thinking": "Initial thought...",
			},
		},
	})

	updates := fake.allUpdates()
	require.Len(t, updates, 1)
	require.NotNil(t, updates[0].Update.AgentThoughtChunk, "should send AgentThoughtChunk for thinking content_block_start")
}

func TestStreamEvent_MessageStop_CompletesActiveTools(t *testing.T) {
	a, fake := newTestAdapter()
	ctx := context.Background()

	a.activeTools = map[string]string{"t1": "mcp__flowgentic__set_topic"}

	a.normalizeAndSend(ctx, testSessionID, &claudecode.StreamEvent{
		Event: map[string]any{
			"type": "message_stop",
		},
	})

	updates := fake.allUpdates()
	require.Len(t, updates, 1)
	require.NotNil(t, updates[0].Update.ToolCallUpdate)
	require.NotNil(t, updates[0].Update.ToolCallUpdate.Status)
	assert.Equal(t, acpsdk.ToolCallStatusCompleted, *updates[0].Update.ToolCallUpdate.Status)
	assert.Equal(t, acpsdk.ToolCallId("t1"), updates[0].Update.ToolCallUpdate.ToolCallId)
	assert.Nil(t, a.activeTools, "active tools should be cleared after message_stop")
}

func TestNormalizeAndSend_StreamEventMessageStop_ReturnsFalse(t *testing.T) {
	a, _ := newTestAdapter()
	ctx := context.Background()

	result := a.normalizeAndSend(ctx, testSessionID, &claudecode.StreamEvent{
		Event: map[string]any{
			"type": "message_stop",
		},
	})

	assert.False(t, result, "message_stop should not signal turn completion")
}

func TestStreamEvent_ContentBlockStop_CompletesMatchingTool(t *testing.T) {
	a, fake := newTestAdapter()
	ctx := context.Background()

	a.activeTools = map[string]string{"t1": "mcp__flowgentic__set_topic", "t2": "Read"}

	a.normalizeAndSend(ctx, testSessionID, &claudecode.StreamEvent{
		Event: map[string]any{
			"type": "content_block_stop",
			"id":   "t1",
		},
	})

	updates := fake.allUpdates()
	require.Len(t, updates, 1)
	require.NotNil(t, updates[0].Update.ToolCallUpdate)
	require.NotNil(t, updates[0].Update.ToolCallUpdate.Status)
	assert.Equal(t, acpsdk.ToolCallStatusCompleted, *updates[0].Update.ToolCallUpdate.Status)
	assert.Equal(t, acpsdk.ToolCallId("t1"), updates[0].Update.ToolCallUpdate.ToolCallId)
	assert.Contains(t, a.activeTools, "t2", "other active tools should remain tracked")
	assert.NotContains(t, a.activeTools, "t1", "completed tool should be removed")
}

func TestSystemMessage_ExtractsText(t *testing.T) {
	a, fake := newTestAdapter()
	ctx := context.Background()

	a.normalizeAndSend(ctx, testSessionID, &claudecode.SystemMessage{
		MessageType: "system",
		Subtype:     "init",
		Data: map[string]any{
			"message": map[string]any{
				"content": []any{
					map[string]any{
						"type": "text",
						"text": "System initialized",
					},
					map[string]any{
						"type": "text",
						"text": " successfully",
					},
				},
			},
		},
	})

	updates := fake.allUpdates()
	require.Len(t, updates, 1)
	require.NotNil(t, updates[0].Update.AgentMessageChunk, "should send AgentMessageChunk for system message text")
}

func TestSystemMessage_AvailableCommandsRootLevel(t *testing.T) {
	a, fake := newTestAdapter()
	ctx := context.Background()

	a.normalizeAndSend(ctx, testSessionID, &claudecode.SystemMessage{
		MessageType: "system",
		Subtype:     "init",
		Data: map[string]any{
			"availableCommands": []any{
				map[string]any{"name": "init", "description": "create/update AGENTS.md"},
				map[string]any{"name": "review", "description": "review changes"},
			},
		},
	})

	updates := fake.allUpdates()
	require.Len(t, updates, 1)
	require.NotNil(t, updates[0].Update.AvailableCommandsUpdate)
	require.Len(t, updates[0].Update.AvailableCommandsUpdate.AvailableCommands, 2)
	assert.Equal(t, "init", updates[0].Update.AvailableCommandsUpdate.AvailableCommands[0].Name)
	assert.Equal(t, "review changes", updates[0].Update.AvailableCommandsUpdate.AvailableCommands[1].Description)
}

func TestSystemMessage_AvailableCommandsNestedUnderMessage(t *testing.T) {
	a, fake := newTestAdapter()
	ctx := context.Background()

	a.normalizeAndSend(ctx, testSessionID, &claudecode.SystemMessage{
		MessageType: "system",
		Subtype:     "init",
		Data: map[string]any{
			"message": map[string]any{
				"availableCommands": []any{
					map[string]any{"name": "compact", "description": "compact the session"},
				},
			},
		},
	})

	updates := fake.allUpdates()
	require.Len(t, updates, 1)
	require.NotNil(t, updates[0].Update.AvailableCommandsUpdate)
	require.Len(t, updates[0].Update.AvailableCommandsUpdate.AvailableCommands, 1)
	assert.Equal(t, "compact", updates[0].Update.AvailableCommandsUpdate.AvailableCommands[0].Name)
}

func TestSystemMessage_AvailableCommandsNestedUnderData(t *testing.T) {
	a, fake := newTestAdapter()
	ctx := context.Background()

	a.normalizeAndSend(ctx, testSessionID, &claudecode.SystemMessage{
		MessageType: "system",
		Subtype:     "init",
		Data: map[string]any{
			"type":    "system",
			"subtype": "init",
			"data": map[string]any{
				"availableCommands": []any{
					map[string]any{"name": "review", "description": "review changes"},
				},
			},
		},
	})

	updates := fake.allUpdates()
	require.Len(t, updates, 1)
	require.NotNil(t, updates[0].Update.AvailableCommandsUpdate)
	require.Len(t, updates[0].Update.AvailableCommandsUpdate.AvailableCommands, 1)
	assert.Equal(t, "review", updates[0].Update.AvailableCommandsUpdate.AvailableCommands[0].Name)
}

func TestSystemMessage_AvailableCommandsSnakeCaseUnderData(t *testing.T) {
	a, fake := newTestAdapter()
	ctx := context.Background()

	a.normalizeAndSend(ctx, testSessionID, &claudecode.SystemMessage{
		MessageType: "system",
		Subtype:     "init",
		Data: map[string]any{
			"type":    "system",
			"subtype": "init",
			"data": map[string]any{
				"available_commands": []any{
					map[string]any{"name": "compact", "description": "compact the session"},
				},
			},
		},
	})

	updates := fake.allUpdates()
	require.Len(t, updates, 1)
	require.NotNil(t, updates[0].Update.AvailableCommandsUpdate)
	require.Len(t, updates[0].Update.AvailableCommandsUpdate.AvailableCommands, 1)
	assert.Equal(t, "compact", updates[0].Update.AvailableCommandsUpdate.AvailableCommands[0].Name)
}

func TestNilConn_NoOp(t *testing.T) {
	// Adapter with neither conn nor updater set.
	a := &Adapter{
		log: testLogger(),
	}
	ctx := context.Background()

	t.Run("AssistantMessage", func(t *testing.T) {
		result := a.normalizeAndSend(ctx, testSessionID, &claudecode.AssistantMessage{
			MessageType: "assistant",
			Content: []claudecode.ContentBlock{
				&claudecode.TextBlock{MessageType: "text", Text: "hello"},
			},
		})
		assert.False(t, result, "should return false with nil sender")
	})

	t.Run("ResultMessage", func(t *testing.T) {
		result := a.normalizeAndSend(ctx, testSessionID, &claudecode.ResultMessage{
			MessageType: "result",
		})
		assert.False(t, result, "should return false with nil sender")
	})

	t.Run("StreamEvent", func(t *testing.T) {
		result := a.normalizeAndSend(ctx, testSessionID, &claudecode.StreamEvent{
			Event: map[string]any{
				"type": "content_block_delta",
				"delta": map[string]any{
					"type": "text_delta",
					"text": "chunk",
				},
			},
		})
		assert.False(t, result, "should return false with nil sender")
	})
}

// --- New tests for enhancements ---

func TestCancel_InvokesPromptCancel(t *testing.T) {
	a, _ := newTestAdapter()
	ctx := context.Background()

	cancelled := false
	a.mu.Lock()
	a.promptCancel = func() { cancelled = true }
	a.mu.Unlock()

	err := a.Cancel(ctx, acpsdk.CancelNotification{})
	require.NoError(t, err)
	assert.True(t, cancelled, "Cancel should invoke promptCancel")
}

func TestCancel_NoOpWhenNoPrompt(t *testing.T) {
	a, _ := newTestAdapter()
	err := a.Cancel(context.Background(), acpsdk.CancelNotification{})
	require.NoError(t, err, "Cancel with no active prompt should not error")
}

func TestToolCallMetadata_BashWithDescription(t *testing.T) {
	a, fake := newTestAdapter()
	ctx := context.Background()

	a.normalizeAndSend(ctx, testSessionID, &claudecode.AssistantMessage{
		MessageType: "assistant",
		Content: []claudecode.ContentBlock{
			&claudecode.ToolUseBlock{
				MessageType: "tool_use",
				ToolUseID:   "t1",
				Name:        "Bash",
				Input: map[string]any{
					"command":     "make build",
					"description": "Build the project",
				},
			},
		},
	})

	updates := fake.allUpdates()
	require.GreaterOrEqual(t, len(updates), 1)

	u0 := updates[0].Update
	require.NotNil(t, u0.ToolCall)
	assert.Equal(t, "Build the project", u0.ToolCall.Title)
	assert.Equal(t, acpsdk.ToolKindExecute, u0.ToolCall.Kind)
	assert.NotNil(t, u0.ToolCall.Meta)
	// Verify meta structure
	meta, ok := u0.ToolCall.Meta.(claudeCodeMeta)
	require.True(t, ok)
	assert.Equal(t, "Bash", meta.ClaudeCode.ToolName)
}

func TestToolCallMetadata_EditWithDiffContent(t *testing.T) {
	a, fake := newTestAdapter()
	ctx := context.Background()

	a.normalizeAndSend(ctx, testSessionID, &claudecode.AssistantMessage{
		MessageType: "assistant",
		Content: []claudecode.ContentBlock{
			&claudecode.ToolUseBlock{
				MessageType: "tool_use",
				ToolUseID:   "t1",
				Name:        "Edit",
				Input: map[string]any{
					"file_path":  "src/main.go",
					"old_string": "foo",
					"new_string": "bar",
				},
			},
		},
	})

	updates := fake.allUpdates()
	require.GreaterOrEqual(t, len(updates), 1)

	u0 := updates[0].Update
	require.NotNil(t, u0.ToolCall)
	assert.Equal(t, "Edit `src/main.go`", u0.ToolCall.Title)
	assert.Equal(t, acpsdk.ToolKindEdit, u0.ToolCall.Kind)
	require.Len(t, u0.ToolCall.Content, 1)
	require.NotNil(t, u0.ToolCall.Content[0].Diff)
	assert.Equal(t, "src/main.go", u0.ToolCall.Content[0].Diff.Path)
	assert.Equal(t, "bar", u0.ToolCall.Content[0].Diff.NewText)
	require.NotNil(t, u0.ToolCall.Content[0].Diff.OldText)
	assert.Equal(t, "foo", *u0.ToolCall.Content[0].Diff.OldText)
	require.Len(t, u0.ToolCall.Locations, 1)
	assert.Equal(t, "src/main.go", u0.ToolCall.Locations[0].Path)
}

func TestToolCallMetadata_GrepSearch(t *testing.T) {
	a, fake := newTestAdapter()
	ctx := context.Background()

	a.normalizeAndSend(ctx, testSessionID, &claudecode.AssistantMessage{
		MessageType: "assistant",
		Content: []claudecode.ContentBlock{
			&claudecode.ToolUseBlock{
				MessageType: "tool_use",
				ToolUseID:   "t1",
				Name:        "Grep",
				Input: map[string]any{
					"pattern": "TODO",
					"path":    "src/",
				},
			},
		},
	})

	updates := fake.allUpdates()
	require.GreaterOrEqual(t, len(updates), 1)

	u0 := updates[0].Update
	require.NotNil(t, u0.ToolCall)
	assert.Equal(t, `grep "TODO" src/`, u0.ToolCall.Title)
	assert.Equal(t, acpsdk.ToolKindSearch, u0.ToolCall.Kind)
}

func TestToolCallMetadata_StreamUpgradeEnrichesMetadata(t *testing.T) {
	a, fake := newTestAdapter()
	ctx := context.Background()

	// Stream event starts the tool (no input yet).
	a.normalizeAndSend(ctx, testSessionID, &claudecode.StreamEvent{
		Event: map[string]any{
			"type": "content_block_start",
			"content_block": map[string]any{
				"type": "tool_use",
				"id":   "t1",
				"name": "Edit",
			},
		},
	})

	// AssistantMessage upgrades it with input.
	a.normalizeAndSend(ctx, testSessionID, &claudecode.AssistantMessage{
		MessageType: "assistant",
		Content: []claudecode.ContentBlock{
			&claudecode.ToolUseBlock{
				MessageType: "tool_use",
				ToolUseID:   "t1",
				Name:        "Edit",
				Input: map[string]any{
					"file_path":  "src/app.go",
					"old_string": "a",
					"new_string": "b",
				},
			},
		},
	})

	updates := fake.allUpdates()
	require.Len(t, updates, 2)

	// Start: generic title (no input known yet)
	u0 := updates[0].Update
	require.NotNil(t, u0.ToolCall)
	assert.Equal(t, "Edit", u0.ToolCall.Title) // no file_path yet
	assert.Equal(t, acpsdk.ToolKindEdit, u0.ToolCall.Kind)

	// Upgrade: enriched title & content
	u1 := updates[1].Update
	require.NotNil(t, u1.ToolCallUpdate)
	require.NotNil(t, u1.ToolCallUpdate.Title)
	assert.Equal(t, "Edit `src/app.go`", *u1.ToolCallUpdate.Title)
	require.NotNil(t, u1.ToolCallUpdate.Kind)
	assert.Equal(t, acpsdk.ToolKindEdit, *u1.ToolCallUpdate.Kind)
	require.Len(t, u1.ToolCallUpdate.Content, 1)
	require.NotNil(t, u1.ToolCallUpdate.Content[0].Diff)
}

func TestAdapterImplementsAgentExperimental(t *testing.T) {
	// Compile-time check is at bottom of adapter.go, but also verify at runtime.
	var a any = &Adapter{log: testLogger()}
	_, ok := a.(acpsdk.AgentExperimental)
	assert.True(t, ok, "Adapter should implement AgentExperimental")
}

func TestSetSessionModel_NoClient(t *testing.T) {
	a, _ := newTestAdapter()
	_, err := a.SetSessionModel(context.Background(), acpsdk.SetSessionModelRequest{
		ModelId: "claude-sonnet",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active session")
}
