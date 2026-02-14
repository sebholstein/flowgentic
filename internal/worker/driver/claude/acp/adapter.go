package acp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/google/uuid"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
	claudecode "github.com/severity1/claude-agent-sdk-go"
)

// updateSender abstracts sending session updates, enabling test injection.
type updateSender interface {
	SessionUpdate(ctx context.Context, n acpsdk.SessionNotification) error
}

type modelStateProvider interface {
	SessionModelState(ctx context.Context) (*acpsdk.SessionModelState, error)
}

// Adapter implements acp.Agent by wrapping the claude-agent-sdk-go library.
type Adapter struct {
	log  *slog.Logger
	conn *acpsdk.AgentSideConnection

	// updater sends ACP session updates. Defaults to a.conn when nil.
	// Tests inject a fake to capture updates without a real connection.
	updater updateSender

	// Per-session state (single session per adapter instance).
	cwd          string
	systemPrompt string
	model        string
	sessionMode  string
	allowedTools []string
	sessionID    string
	envVars      map[string]string
	mcpServers   map[string]claudecode.McpServerConfig
	planModeMCP  bool

	// Persistent Claude SDK client — lives across Prompt() calls so
	// multi-turn conversations share the same subprocess and history.
	mu      sync.Mutex
	client  claudecode.Client // lazy-initialized on first Prompt()
	msgChan <-chan claudecode.Message

	// sessionCtx/sessionCancel control the Claude subprocess lifetime.
	// They outlive individual Prompt() calls so the subprocess persists.
	sessionCtx    context.Context
	sessionCancel context.CancelFunc

	// promptCancel cancels the in-flight Prompt() context when Cancel() is called.
	promptCancel context.CancelFunc

	// activeTools tracks tool calls that have been started but not yet completed.
	// Maps toolCallId → tool name. Used to deduplicate starts (stream vs batch)
	// and synthesize completion events when the next assistant turn begins.
	activeTools map[string]string

	modelProvider modelStateProvider
}

// NewAdapter creates a new Claude ACP adapter.
func NewAdapter(log *slog.Logger) acpsdk.Agent {
	return &Adapter{log: log.With("adapter", "claude-code")}
}

// SetConnection is called after the agent-side connection is created,
// so the adapter can send notifications back to the client.
func (a *Adapter) SetConnection(conn *acpsdk.AgentSideConnection) {
	a.conn = conn
}

func (a *Adapter) Authenticate(_ context.Context, _ acpsdk.AuthenticateRequest) (acpsdk.AuthenticateResponse, error) {
	return acpsdk.AuthenticateResponse{}, nil
}

func (a *Adapter) Initialize(_ context.Context, _ acpsdk.InitializeRequest) (acpsdk.InitializeResponse, error) {
	return acpsdk.InitializeResponse{
		ProtocolVersion: acpsdk.ProtocolVersion(acpsdk.ProtocolVersionNumber),
		AgentInfo: &acpsdk.Implementation{
			Name:    "claude-code",
			Version: "1.0.0",
		},
	}, nil
}

func (a *Adapter) Cancel(_ context.Context, _ acpsdk.CancelNotification) error {
	a.mu.Lock()
	cancel := a.promptCancel
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

func (a *Adapter) NewSession(_ context.Context, req acpsdk.NewSessionRequest) (acpsdk.NewSessionResponse, error) {
	a.cwd = req.Cwd
	a.sessionID = uuid.New().String()
	a.mcpServers = convertMCPServers(req.McpServers)
	a.log.Info(
		"claude new session",
		"cwd", req.Cwd,
		"mcp_servers", len(a.mcpServers),
		"mcp_server_names", mapKeys(a.mcpServers),
		"mcp_server_summaries", summarizeMCPServers(a.mcpServers),
	)

	// Parse _meta for adapter-specific options.
	if meta, ok := req.Meta.(map[string]any); ok {
		if sp, ok := meta["systemPrompt"].(string); ok {
			a.systemPrompt = sp
		}
		if m, ok := meta["model"].(string); ok {
			a.model = m
		}
		if sm, ok := meta["sessionMode"].(string); ok {
			a.sessionMode = sm
		}
		if tools, ok := meta["allowedTools"].([]any); ok {
			for _, t := range tools {
				if s, ok := t.(string); ok {
					a.allowedTools = append(a.allowedTools, s)
				}
			}
		}
		if env, ok := meta["envVars"].(map[string]any); ok {
			a.envVars = make(map[string]string, len(env))
			for k, v := range env {
				if s, ok := v.(string); ok {
					a.envVars[k] = s
				}
			}
		}
	}
	a.planModeMCP = strings.Contains(a.systemPrompt, "## Flowgentic MCP") && len(a.mcpServers) > 0

	resp := acpsdk.NewSessionResponse{
		SessionId: acpsdk.SessionId(a.sessionID),
	}
	if a.modelProvider != nil {
		state, err := a.modelProvider.SessionModelState(context.Background())
		if err != nil {
			return acpsdk.NewSessionResponse{}, fmt.Errorf("model state: %w", err)
		}
		if state != nil {
			cloned := *state
			cloned.AvailableModels = append([]acpsdk.ModelInfo(nil), state.AvailableModels...)
			resp.Models = &cloned
		}
	}
	return resp, nil
}

func (a *Adapter) Prompt(ctx context.Context, req acpsdk.PromptRequest) (acpsdk.PromptResponse, error) {
	// Wrap context so Cancel() can abort the in-flight prompt.
	ctx, cancel := context.WithCancel(ctx)
	a.mu.Lock()
	a.promptCancel = cancel
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		a.promptCancel = nil
		a.mu.Unlock()
		cancel()
	}()

	// Extract text from prompt content blocks.
	var promptText string
	for _, block := range req.Prompt {
		if block.Text != nil {
			promptText += block.Text.Text
		}
	}

	// Lazy-init: create and connect client on first Prompt() call.
	// The client persists across calls so multi-turn conversations
	// share the same subprocess and conversation history.
	a.mu.Lock()
	if a.client == nil {
		// Use a session-scoped context for Connect/ReceiveMessages so the
		// subprocess outlives individual Prompt() calls.
		sessionCtx, sessionCancel := context.WithCancel(context.Background())
		a.sessionCtx = sessionCtx
		a.sessionCancel = sessionCancel

		sdkOpts := a.buildSDKOptions()
		sdkOpts = append(sdkOpts, claudecode.WithCanUseTool(func(toolCtx context.Context, toolName string, input map[string]any, _ claudecode.ToolPermissionContext) (claudecode.PermissionResult, error) {
			return a.handlePermission(toolCtx, req.SessionId, toolName, input)
		}))

		client := claudecode.NewClient(sdkOpts...)
		if err := client.Connect(sessionCtx); err != nil {
			sessionCancel()
			a.mu.Unlock()
			return acpsdk.PromptResponse{}, fmt.Errorf("connect: %w", err)
		}
		a.client = client
		a.msgChan = client.ReceiveMessages(sessionCtx)
	}
	a.mu.Unlock()

	// Send prompt on the persistent session.
	if promptText != "" {
		if err := a.client.QueryWithSession(ctx, promptText, a.sessionID); err != nil {
			return acpsdk.PromptResponse{}, fmt.Errorf("query: %w", err)
		}
	}

	// Drain messages until ResultMessage signals the turn is complete.
	var finalStopReason acpsdk.StopReason = acpsdk.StopReasonEndTurn
	for {
		select {
		case msg, ok := <-a.msgChan:
			if !ok || msg == nil {
				return acpsdk.PromptResponse{StopReason: finalStopReason}, nil
			}
			if a.normalizeAndSend(ctx, req.SessionId, msg) {
				return acpsdk.PromptResponse{StopReason: finalStopReason}, nil
			}
		case <-ctx.Done():
			finalStopReason = acpsdk.StopReasonCancelled
			return acpsdk.PromptResponse{StopReason: finalStopReason}, nil
		}
	}
}

func (a *Adapter) SetSessionMode(ctx context.Context, req acpsdk.SetSessionModeRequest) (acpsdk.SetSessionModeResponse, error) {
	mode, err := driver.ParseSessionMode(string(req.ModeId))
	if err != nil {
		return acpsdk.SetSessionModeResponse{}, fmt.Errorf("set session mode: %w", err)
	}

	permMode, err := sessionModeToPermission(mode)
	if err != nil {
		return acpsdk.SetSessionModeResponse{}, err
	}

	a.mu.Lock()
	client := a.client
	a.mu.Unlock()

	if client == nil {
		return acpsdk.SetSessionModeResponse{}, fmt.Errorf("no active streaming session")
	}

	if err := client.SetPermissionMode(ctx, permMode); err != nil {
		return acpsdk.SetSessionModeResponse{}, fmt.Errorf("set permission mode: %w", err)
	}

	// Notify the client of the mode change.
	a.sendUpdate(ctx, acpsdk.SessionId(a.sessionID), acpsdk.SessionUpdate{
		CurrentModeUpdate: &acpsdk.SessionCurrentModeUpdate{
			CurrentModeId: req.ModeId,
		},
	})

	return acpsdk.SetSessionModeResponse{}, nil
}

// SetSessionModel implements the experimental AgentExperimental interface.
func (a *Adapter) SetSessionModel(ctx context.Context, req acpsdk.SetSessionModelRequest) (acpsdk.SetSessionModelResponse, error) {
	a.mu.Lock()
	client := a.client
	a.mu.Unlock()

	if client == nil {
		return acpsdk.SetSessionModelResponse{}, errors.New("no active session")
	}

	modelStr := string(req.ModelId)
	if err := client.SetModel(ctx, &modelStr); err != nil {
		return acpsdk.SetSessionModelResponse{}, fmt.Errorf("set model: %w", err)
	}
	a.model = modelStr

	return acpsdk.SetSessionModelResponse{}, nil
}

// sessionModeToPermission maps a driver.SessionMode to a Claude SDK PermissionMode.
func sessionModeToPermission(mode driver.SessionMode) (claudecode.PermissionMode, error) {
	switch mode {
	case driver.SessionModeAsk:
		return claudecode.PermissionModeDefault, nil
	case driver.SessionModeArchitect:
		return claudecode.PermissionModePlan, nil
	case driver.SessionModeCode:
		return claudecode.PermissionModeBypassPermissions, nil
	default:
		return "", fmt.Errorf("unsupported session mode: %q", mode)
	}
}

// handlePermission delegates to the ACP client's RequestPermission.
func (a *Adapter) handlePermission(ctx context.Context, sessionID acpsdk.SessionId, toolName string, input map[string]any) (claudecode.PermissionResult, error) {
	if a.planModeMCP && !isAllowedInFlowgenticPlanMode(toolName) {
		a.log.Warn("denying tool outside Flowgentic plan mode allowlist", "tool", toolName)
		return claudecode.NewPermissionResultDeny("tool is not allowed in Flowgentic plan mode"), nil
	}
	if isDisallowedTool(toolName) {
		a.log.Warn("denying disallowed tool call", "tool", toolName)
		// Do not interrupt the turn; allow the model to continue with plain-text
		// questions or proceed directly to planning in the same response.
		return claudecode.NewPermissionResultDeny("tool is not allowed in this session"), nil
	}

	if a.conn == nil {
		return claudecode.NewPermissionResultDeny("no ACP connection"), nil
	}

	info := toolInfoFromToolUse(toolName, input)
	meta := newClaudeCodeMeta(toolName)

	// Build permission options — ExitPlanMode gets special options.
	options := []acpsdk.PermissionOption{
		{OptionId: "allow_always", Name: "Always Allow", Kind: acpsdk.PermissionOptionKindAllowAlways},
		{OptionId: "allow", Name: "Allow", Kind: acpsdk.PermissionOptionKindAllowOnce},
		{OptionId: "reject", Name: "Reject", Kind: acpsdk.PermissionOptionKindRejectOnce},
	}
	if toolName == "ExitPlanMode" {
		options = []acpsdk.PermissionOption{
			{OptionId: "acceptEdits", Name: "Yes, and auto-accept edits", Kind: acpsdk.PermissionOptionKindAllowAlways},
			{OptionId: "default", Name: "Yes, and manually approve edits", Kind: acpsdk.PermissionOptionKindAllowOnce},
			{OptionId: "plan", Name: "No, keep planning", Kind: acpsdk.PermissionOptionKindRejectOnce},
		}
	}

	resp, err := a.conn.RequestPermission(ctx, acpsdk.RequestPermissionRequest{
		SessionId: sessionID,
		Options:   options,
		ToolCall: acpsdk.RequestPermissionToolCall{
			ToolCallId: acpsdk.ToolCallId(toolName),
			Title:      &info.Title,
			Kind:       &info.Kind,
			RawInput:   input,
			Content:    info.Content,
			Locations:  info.Locations,
			Meta:       meta,
		},
	})
	if err != nil {
		return claudecode.NewPermissionResultDeny("permission request failed"), nil
	}

	if resp.Outcome.Selected == nil {
		return claudecode.NewPermissionResultDeny("no option selected"), nil
	}

	switch resp.Outcome.Selected.OptionId {
	case "allow", "default", "acceptEdits", "allow_always":
		result := claudecode.NewPermissionResultAllow()
		return result, nil
	case "plan":
		deny := claudecode.NewPermissionResultDeny("user chose to keep planning")
		deny.Interrupt = true
		return deny, nil
	default:
		return claudecode.NewPermissionResultDeny("user denied"), nil
	}
}

func isDisallowedTool(toolName string) bool {
	switch toolName {
	case "AskUserQuestion":
		return true
	default:
		return false
	}
}

func isAllowedInFlowgenticPlanMode(toolName string) bool {
	if strings.HasPrefix(toolName, "mcp__flowgentic__") {
		return true
	}
	switch toolName {
	case "Read", "Write", "Edit", "MultiEdit", "Glob", "Grep", "LS":
		return true
	default:
		return false
	}
}

// normalizeAndSend converts SDK messages to ACP SessionUpdate notifications.
// Returns true only when a ResultMessage is received, signaling turn completion.
func (a *Adapter) normalizeAndSend(ctx context.Context, sessionID acpsdk.SessionId, msg claudecode.Message) bool {
	if a.sender() == nil {
		return false
	}

	switch m := msg.(type) {
	case *claudecode.AssistantMessage:
		a.normalizeAssistantMessage(ctx, sessionID, m)
	case *claudecode.ResultMessage:
		a.normalizeResultMessage(ctx, sessionID, m)
		return true
	case *claudecode.StreamEvent:
		return a.normalizeStreamEvent(ctx, sessionID, m)
	case *claudecode.SystemMessage:
		a.normalizeSystemMessage(ctx, sessionID, m)
	}
	return false
}

// toolStartOpts builds the StartToolCall options for a given tool, including
// rich metadata from toolInfoFromToolUse.
func toolStartOpts(name string, input map[string]any, status acpsdk.ToolCallStatus) (string, []acpsdk.ToolCallStartOpt) {
	info := toolInfoFromToolUse(name, input)
	meta := newClaudeCodeMeta(name)

	opts := []acpsdk.ToolCallStartOpt{
		acpsdk.WithStartKind(info.Kind),
		acpsdk.WithStartStatus(status),
	}
	if input != nil {
		opts = append(opts, acpsdk.WithStartRawInput(input))
	}
	if len(info.Content) > 0 {
		opts = append(opts, acpsdk.WithStartContent(info.Content))
	}
	if len(info.Locations) > 0 {
		opts = append(opts, acpsdk.WithStartLocations(info.Locations))
	}
	// Set _meta via a custom opt since there's no WithStartMeta helper.
	opts = append(opts, func(tc *acpsdk.SessionUpdateToolCall) {
		tc.Meta = meta
	})

	return info.Title, opts
}

func (a *Adapter) normalizeAssistantMessage(ctx context.Context, sessionID acpsdk.SessionId, msg *claudecode.AssistantMessage) {
	// A new assistant message means any previously active tools have completed,
	// EXCEPT tools that appear in this message (they're being upgraded from
	// pending → in_progress). Collect those IDs first to avoid premature completion.
	keep := make(map[string]bool)
	for _, block := range msg.Content {
		if b, ok := block.(*claudecode.ToolUseBlock); ok {
			keep[b.ToolUseID] = true
		}
	}
	a.completeActiveToolsExcept(ctx, sessionID, keep)

	for _, block := range msg.Content {
		switch b := block.(type) {
		case *claudecode.TextBlock:
			// Skip — already streamed via text_delta stream events.
		case *claudecode.ThinkingBlock:
			// Skip — already streamed via thinking_delta stream events.
		case *claudecode.ToolUseBlock:
			id := b.ToolUseID
			if _, already := a.activeTools[id]; already {
				// Already started via stream event — upgrade to in_progress with input.
				info := toolInfoFromToolUse(b.Name, b.Input)
				updateOpts := []acpsdk.ToolCallUpdateOpt{
					acpsdk.WithUpdateStatus(acpsdk.ToolCallStatusInProgress),
					acpsdk.WithUpdateRawInput(b.Input),
					acpsdk.WithUpdateTitle(info.Title),
					acpsdk.WithUpdateKind(info.Kind),
				}
				if len(info.Content) > 0 {
					updateOpts = append(updateOpts, acpsdk.WithUpdateContent(info.Content))
				}
				a.sendUpdate(ctx, sessionID, acpsdk.UpdateToolCall(
					acpsdk.ToolCallId(id),
					updateOpts...,
				))
			} else {
				// No stream event preceded this — send full StartToolCall.
				title, opts := toolStartOpts(b.Name, b.Input, acpsdk.ToolCallStatusInProgress)
				a.sendUpdate(ctx, sessionID, acpsdk.StartToolCall(
					acpsdk.ToolCallId(id),
					title,
					opts...,
				))
				if a.activeTools == nil {
					a.activeTools = make(map[string]string)
				}
				a.activeTools[id] = b.Name
			}
		case *claudecode.ToolResultBlock:
			status := acpsdk.ToolCallStatusCompleted
			if b.IsError != nil && *b.IsError {
				status = acpsdk.ToolCallStatusFailed
			}
			raw, _ := json.Marshal(b.Content)
			a.sendUpdate(ctx, sessionID, acpsdk.UpdateToolCall(
				acpsdk.ToolCallId(b.ToolUseID),
				acpsdk.WithUpdateStatus(status),
				acpsdk.WithUpdateRawOutput(json.RawMessage(raw)),
			))
			delete(a.activeTools, b.ToolUseID)
		}
	}
}

func (a *Adapter) normalizeResultMessage(ctx context.Context, sessionID acpsdk.SessionId, _ *claudecode.ResultMessage) {
	// Result message signals conversation completion — complete any remaining tools.
	a.completeActiveTools(ctx, sessionID)
}

// completeActiveTools sends completion updates for all tracked tool calls
// and clears the active set.
func (a *Adapter) completeActiveTools(ctx context.Context, sessionID acpsdk.SessionId) {
	a.completeActiveToolsExcept(ctx, sessionID, nil)
}

// completeActiveToolsExcept sends completion updates for tracked tool calls,
// skipping any IDs in the keep set. Completed tools are removed from activeTools.
func (a *Adapter) completeActiveToolsExcept(ctx context.Context, sessionID acpsdk.SessionId, keep map[string]bool) {
	for id := range a.activeTools {
		if keep[id] {
			continue
		}
		a.sendUpdate(ctx, sessionID, acpsdk.UpdateToolCall(
			acpsdk.ToolCallId(id),
			acpsdk.WithUpdateStatus(acpsdk.ToolCallStatusCompleted),
		))
		delete(a.activeTools, id)
	}
	if len(a.activeTools) == 0 {
		a.activeTools = nil
	}
}

func (a *Adapter) normalizeStreamEvent(ctx context.Context, sessionID acpsdk.SessionId, msg *claudecode.StreamEvent) bool {
	if msg.Event == nil {
		return false
	}

	eventType, _ := msg.Event["type"].(string)

	switch eventType {
	case "message_stop":
		// Some SDK/client combinations can end a turn with stream boundary
		// events between assistant chunks. Ensure no tool card is left
		// in-progress, but do not treat this as full turn completion.
		a.completeActiveTools(ctx, sessionID)
		return false

	case "content_block_start":
		cb, ok := msg.Event["content_block"].(map[string]any)
		if !ok {
			return false
		}
		cbType, _ := cb["type"].(string)
		switch cbType {
		case "tool_use":
			name, _ := cb["name"].(string)
			id, _ := cb["id"].(string)
			if a.activeTools == nil {
				a.activeTools = make(map[string]string)
			}
			a.activeTools[id] = name
			// Stream events don't have input yet, so we pass nil — metadata
			// will be enriched when the AssistantMessage arrives with input.
			title, opts := toolStartOpts(name, nil, acpsdk.ToolCallStatusPending)
			a.sendUpdate(ctx, sessionID, acpsdk.StartToolCall(
				acpsdk.ToolCallId(id),
				title,
				opts...,
			))
		case "text":
			// Text streaming starting means the model is responding to tool
			// results — any active tools have finished executing.
			a.completeActiveTools(ctx, sessionID)
		case "thinking":
			thinking, _ := cb["thinking"].(string)
			if thinking != "" {
				a.sendUpdate(ctx, sessionID, acpsdk.UpdateAgentThoughtText(thinking))
			}
		}

	case "content_block_stop":
		// Be defensive: if the stream includes a tool-use block end, synthesize
		// completion for that specific tool ID.
		var toolID string
		if id, ok := msg.Event["id"].(string); ok {
			toolID = id
		}
		if toolID == "" {
			if cb, ok := msg.Event["content_block"].(map[string]any); ok {
				if id, ok := cb["id"].(string); ok {
					toolID = id
				}
			}
		}
		if toolID != "" {
			if _, active := a.activeTools[toolID]; active {
				a.sendUpdate(ctx, sessionID, acpsdk.UpdateToolCall(
					acpsdk.ToolCallId(toolID),
					acpsdk.WithUpdateStatus(acpsdk.ToolCallStatusCompleted),
				))
				delete(a.activeTools, toolID)
				if len(a.activeTools) == 0 {
					a.activeTools = nil
				}
			}
		}

	case "content_block_delta":
		delta, ok := msg.Event["delta"].(map[string]any)
		if !ok {
			return false
		}
		deltaType, _ := delta["type"].(string)
		switch deltaType {
		case "text_delta":
			text, _ := delta["text"].(string)
			a.sendUpdate(ctx, sessionID, acpsdk.UpdateAgentMessageText(text))
		case "thinking_delta":
			text, _ := delta["text"].(string)
			a.sendUpdate(ctx, sessionID, acpsdk.UpdateAgentThoughtText(text))
		}
	}
	return false
}

func (a *Adapter) normalizeSystemMessage(ctx context.Context, sessionID acpsdk.SessionId, msg *claudecode.SystemMessage) {
	if msg.Data == nil {
		return
	}
	msgData, ok := msg.Data["message"]
	if !ok {
		return
	}
	msgMap, ok := msgData.(map[string]any)
	if !ok {
		return
	}
	contentRaw, ok := msgMap["content"]
	if !ok {
		return
	}
	contentList, ok := contentRaw.([]any)
	if !ok {
		return
	}

	var text string
	for _, item := range contentList {
		itemMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if itemMap["type"] == "text" {
			if t, ok := itemMap["text"].(string); ok {
				text += t
			}
		}
	}

	if text != "" {
		a.sendUpdate(ctx, sessionID, acpsdk.UpdateAgentMessageText(text))
	}
}

func (a *Adapter) sendUpdate(ctx context.Context, sessionID acpsdk.SessionId, update acpsdk.SessionUpdate) {
	sender := a.sender()
	if sender == nil {
		return
	}
	if err := sender.SessionUpdate(ctx, acpsdk.SessionNotification{
		SessionId: sessionID,
		Update:    update,
	}); err != nil {
		a.log.Debug("failed to send session update", "error", err)
	}
}

// sender returns the updateSender to use. It prefers the injected updater
// (used in tests), falling back to the real connection.
func (a *Adapter) sender() updateSender {
	if a.updater != nil {
		return a.updater
	}
	if a.conn != nil {
		return a.conn
	}
	return nil
}

func (a *Adapter) buildSDKOptions() []claudecode.Option {
	var sdkOpts []claudecode.Option

	if a.model != "" {
		sdkOpts = append(sdkOpts, claudecode.WithModel(a.model))
	}
	if a.systemPrompt != "" {
		sdkOpts = append(sdkOpts, claudecode.WithSystemPrompt(a.systemPrompt))
	}
	if a.cwd != "" {
		sdkOpts = append(sdkOpts, claudecode.WithCwd(a.cwd))
	}
	if len(a.allowedTools) > 0 {
		sdkOpts = append(sdkOpts, claudecode.WithAllowedTools(a.allowedTools...))
	}
	if sm, err := driver.ParseSessionMode(a.sessionMode); err == nil {
		if perm, err := sessionModeToPermission(sm); err == nil {
			sdkOpts = append(sdkOpts, claudecode.WithPermissionMode(perm))
		}
	}
	// Note: WithResume is only for resuming an existing Claude Code session.
	// Our ACP sessionID is internal and not known to Claude Code.
	if len(a.envVars) > 0 {
		sdkOpts = append(sdkOpts, claudecode.WithEnv(a.envVars))
	}
	if len(a.mcpServers) > 0 {
		sdkOpts = append(sdkOpts, claudecode.WithMcpServers(a.mcpServers))
		sdkOpts = append(sdkOpts, claudecode.WithStderrCallback(func(line string) {
			l := strings.TrimSpace(line)
			if l == "" {
				return
			}
			if strings.Contains(strings.ToLower(l), "mcp") {
				a.log.Warn("claude stderr (mcp)", "line", l)
				return
			}
			a.log.Debug("claude stderr", "line", l)
		}))
	}

	sdkOpts = append(sdkOpts, claudecode.WithPartialStreaming())
	sdkOpts = append(sdkOpts, claudecode.WithDebugWriter(io.Discard))

	return sdkOpts
}

// Ensure compile-time interface compliance.
var (
	_ acpsdk.Agent             = (*Adapter)(nil)
	_ acpsdk.AgentExperimental = (*Adapter)(nil)
)

func mapKeys[K comparable, V any](m map[K]V) []K {
	if len(m) == 0 {
		return nil
	}
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func summarizeMCPServers(servers map[string]claudecode.McpServerConfig) []string {
	if len(servers) == 0 {
		return nil
	}
	out := make([]string, 0, len(servers))
	for name, cfg := range servers {
		switch c := cfg.(type) {
		case *claudecode.McpStdioServerConfig:
			out = append(out, fmt.Sprintf("%s:stdio cmd=%s args=%d env=%d", name, c.Command, len(c.Args), len(c.Env)))
		case *claudecode.McpSSEServerConfig:
			out = append(out, fmt.Sprintf("%s:sse url=%s headers=%d", name, c.URL, len(c.Headers)))
		case *claudecode.McpHTTPServerConfig:
			out = append(out, fmt.Sprintf("%s:http url=%s headers=%d", name, c.URL, len(c.Headers)))
		default:
			out = append(out, fmt.Sprintf("%s:unknown", name))
		}
	}
	return out
}

func convertMCPServers(servers []acpsdk.McpServer) map[string]claudecode.McpServerConfig {
	if len(servers) == 0 {
		return nil
	}

	out := make(map[string]claudecode.McpServerConfig, len(servers))
	for i, server := range servers {
		switch {
		case server.Stdio != nil:
			name := server.Stdio.Name
			if name == "" {
				name = fmt.Sprintf("mcp-%d", i+1)
			}
			env := map[string]string{}
			for _, kv := range server.Stdio.Env {
				if kv.Name == "" {
					continue
				}
				env[kv.Name] = kv.Value
			}
			out[name] = &claudecode.McpStdioServerConfig{
				Type:    claudecode.McpServerTypeStdio,
				Command: server.Stdio.Command,
				Args:    append([]string(nil), server.Stdio.Args...),
				Env:     env,
			}

		case server.Sse != nil:
			name := server.Sse.Name
			if name == "" {
				name = fmt.Sprintf("mcp-%d", i+1)
			}
			headers := map[string]string{}
			for _, h := range server.Sse.Headers {
				if h.Name == "" {
					continue
				}
				headers[h.Name] = h.Value
			}
			out[name] = &claudecode.McpSSEServerConfig{
				Type:    claudecode.McpServerTypeSSE,
				URL:     server.Sse.Url,
				Headers: headers,
			}

		case server.Http != nil:
			name := server.Http.Name
			if name == "" {
				name = fmt.Sprintf("mcp-%d", i+1)
			}
			headers := map[string]string{}
			for _, h := range server.Http.Headers {
				if h.Name == "" {
					continue
				}
				headers[h.Name] = h.Value
			}
			out[name] = &claudecode.McpHTTPServerConfig{
				Type:    claudecode.McpServerTypeHTTP,
				URL:     server.Http.Url,
				Headers: headers,
			}
		}
	}

	if len(out) == 0 {
		return nil
	}
	return out
}
