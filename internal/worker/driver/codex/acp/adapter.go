package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/google/uuid"
)

// Adapter implements acp.Agent by wrapping the Codex app-server JSON-RPC protocol.
type Adapter struct {
	log  *slog.Logger
	conn *acpsdk.AgentSideConnection

	mu     sync.Mutex
	server *bridge
	// Per-session state.
	threadID string
	turnID   string
	cwd      string

	turnDoneCh chan struct{} // closed when turn/completed received

	pendingPermissions   map[string]pendingPermission
	pendingPermissionsMu sync.Mutex
}

type pendingPermission struct {
	serverRequestID int64
	ch              chan permissionResponse
}

type permissionResponse struct {
	Allow bool
}

// NewAdapter creates a new Codex ACP adapter.
func NewAdapter(log *slog.Logger) acpsdk.Agent {
	return &Adapter{
		log:                log.With("adapter", "codex"),
		pendingPermissions: make(map[string]pendingPermission),
	}
}

// SetConnection stores the agent-side connection for sending notifications.
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
			Name:    "codex",
			Version: "1.0.0",
		},
	}, nil
}

func (a *Adapter) Cancel(_ context.Context, _ acpsdk.CancelNotification) error {
	a.mu.Lock()
	srv := a.server
	threadID := a.threadID
	turnID := a.turnID
	a.mu.Unlock()

	if srv != nil && turnID != "" {
		_ = srv.turnInterrupt(threadID, turnID)
	}
	return nil
}

func (a *Adapter) NewSession(ctx context.Context, req acpsdk.NewSessionRequest) (acpsdk.NewSessionResponse, error) {
	a.cwd = req.Cwd

	// Parse _meta for adapter-specific options.
	var model, systemPrompt, sessionMode string
	var envVars map[string]string
	if meta, ok := req.Meta.(map[string]any); ok {
		model, _ = meta["model"].(string)
		systemPrompt, _ = meta["systemPrompt"].(string)
		sessionMode, _ = meta["sessionMode"].(string)
		if ev, ok := meta["envVars"].(map[string]any); ok {
			envVars = make(map[string]string, len(ev))
			for k, v := range ev {
				if s, ok := v.(string); ok {
					envVars[k] = s
				}
			}
		}
	}

	// Start the shared bridge (app-server process).
	b := newBridge(a.log, a.dispatchNotification)
	if err := b.start(ctx, envVars); err != nil {
		return acpsdk.NewSessionResponse{}, fmt.Errorf("start app-server: %w", err)
	}

	a.mu.Lock()
	a.server = b
	a.mu.Unlock()

	// Create thread.
	threadID, err := b.threadStart(model, req.Cwd, systemPrompt, sessionMode, req.McpServers)
	if err != nil {
		b.close()
		return acpsdk.NewSessionResponse{}, fmt.Errorf("thread/start: %w", err)
	}

	a.mu.Lock()
	a.threadID = threadID
	a.mu.Unlock()

	resp := acpsdk.NewSessionResponse{
		SessionId: acpsdk.SessionId(threadID),
	}
	if b.modelState != nil {
		cloned := *b.modelState
		cloned.AvailableModels = append([]acpsdk.ModelInfo(nil), b.modelState.AvailableModels...)
		resp.Models = &cloned
	}
	return resp, nil
}

func (a *Adapter) Prompt(ctx context.Context, req acpsdk.PromptRequest) (acpsdk.PromptResponse, error) {
	var promptText string
	for _, block := range req.Prompt {
		if block.Text != nil {
			promptText += block.Text.Text
		}
	}

	a.mu.Lock()
	srv := a.server
	threadID := a.threadID
	a.mu.Unlock()

	if srv == nil {
		return acpsdk.PromptResponse{}, fmt.Errorf("no app-server running")
	}

	// Parse sessionMode from session meta if available.
	var sessionMode string
	if meta, ok := req.Meta.(map[string]any); ok {
		sessionMode, _ = meta["sessionMode"].(string)
	}

	turnID, err := srv.turnStart(threadID, promptText, a.cwd, sessionMode)
	if err != nil {
		return acpsdk.PromptResponse{}, fmt.Errorf("turn/start: %w", err)
	}

	a.mu.Lock()
	a.turnID = turnID
	a.mu.Unlock()

	// Wait for turn completion (signaled by dispatchNotification setting turnDone).
	turnDone := make(chan struct{})
	a.mu.Lock()
	a.turnDoneCh = turnDone
	a.mu.Unlock()

	select {
	case <-turnDone:
		return acpsdk.PromptResponse{StopReason: acpsdk.StopReasonEndTurn}, nil
	case <-ctx.Done():
		return acpsdk.PromptResponse{StopReason: acpsdk.StopReasonCancelled}, nil
	case <-srv.done:
		return acpsdk.PromptResponse{StopReason: acpsdk.StopReasonEndTurn}, nil
	}
}

func (a *Adapter) SetSessionMode(_ context.Context, _ acpsdk.SetSessionModeRequest) (acpsdk.SetSessionModeResponse, error) {
	return acpsdk.SetSessionModeResponse{}, nil
}

// dispatchNotification routes app-server JSON-RPC notifications to ACP session updates.
func (a *Adapter) dispatchNotification(threadID string, method string, params json.RawMessage, serverRequestID *int64) {
	if threadID == "" {
		a.mu.Lock()
		threadID = a.threadID
		a.mu.Unlock()
	}
	sessionID := acpsdk.SessionId(threadID)

	// Handle permission requests.
	if method == "item/commandExecution/requestApproval" && serverRequestID != nil {
		a.handleApprovalRequest(sessionID, params, *serverRequestID)
		return
	}

	// Normalize notification to ACP updates.
	updates := a.normalizeNotification(method, params)
	for _, update := range updates {
		a.sendUpdate(context.Background(), sessionID, update)
	}

	// Signal turn completion.
	if method == "turn/completed" {
		a.mu.Lock()
		ch := a.turnDoneCh
		a.turnDoneCh = nil
		a.mu.Unlock()
		if ch != nil {
			close(ch)
		}
	}
}

func (a *Adapter) handleApprovalRequest(sessionID acpsdk.SessionId, params json.RawMessage, serverRequestID int64) {
	if a.conn == nil {
		return
	}

	requestID := uuid.New().String()

	var approvalParams struct {
		Command string `json:"command"`
	}
	_ = json.Unmarshal(params, &approvalParams)

	ch := make(chan permissionResponse, 1)
	a.pendingPermissionsMu.Lock()
	a.pendingPermissions[requestID] = pendingPermission{
		serverRequestID: serverRequestID,
		ch:              ch,
	}
	a.pendingPermissionsMu.Unlock()

	// Request permission from the client.
	go func() {
		resp, err := a.conn.RequestPermission(context.Background(), acpsdk.RequestPermissionRequest{
			SessionId: sessionID,
			Options: []acpsdk.PermissionOption{
				{OptionId: "allow", Name: "Allow", Kind: acpsdk.PermissionOptionKindAllowOnce},
				{OptionId: "deny", Name: "Deny", Kind: acpsdk.PermissionOptionKindRejectOnce},
			},
			ToolCall: acpsdk.RequestPermissionToolCall{
				ToolCallId: acpsdk.ToolCallId(requestID),
				Title:      acpsdk.Ptr(approvalParams.Command),
				Kind:       acpsdk.Ptr(acpsdk.ToolKindExecute),
				RawInput:   map[string]string{"command": approvalParams.Command},
			},
		})

		a.pendingPermissionsMu.Lock()
		delete(a.pendingPermissions, requestID)
		a.pendingPermissionsMu.Unlock()

		if err != nil {
			return
		}

		a.mu.Lock()
		srv := a.server
		a.mu.Unlock()

		if srv != nil {
			decision := "deny"
			if resp.Outcome.Selected != nil && resp.Outcome.Selected.OptionId == "allow" {
				decision = "accept"
			}
			srv.respondToServerRequest(serverRequestID, map[string]string{"decision": decision})
		}
	}()
}

func (a *Adapter) normalizeNotification(method string, params json.RawMessage) []acpsdk.SessionUpdate {
	switch method {
	case "item/agentMessage/delta":
		var p struct {
			Delta string `json:"delta"`
		}
		if json.Unmarshal(params, &p) != nil {
			return nil
		}
		return []acpsdk.SessionUpdate{acpsdk.UpdateAgentMessageText(p.Delta)}

	case "item/reasoning/textDelta":
		var p struct {
			Delta string `json:"delta"`
		}
		if json.Unmarshal(params, &p) != nil {
			return nil
		}
		return []acpsdk.SessionUpdate{acpsdk.UpdateAgentThoughtText(p.Delta)}

	case "item/started":
		var p struct {
			Item struct {
				ID        string `json:"id"`
				Type      string `json:"type"`
				Command   string `json:"command,omitempty"`
				Server    string `json:"server,omitempty"`
				ToolName  string `json:"toolName,omitempty"`
				Name      string `json:"name,omitempty"`
				Arguments any    `json:"arguments,omitempty"`
			} `json:"item"`
		}
		if json.Unmarshal(params, &p) != nil {
			return nil
		}
		if p.Item.Type == "commandExecution" {
			return []acpsdk.SessionUpdate{
				acpsdk.StartToolCall(
					acpsdk.ToolCallId(p.Item.ID),
					p.Item.Command,
					acpsdk.WithStartKind(acpsdk.ToolKindExecute),
					acpsdk.WithStartStatus(acpsdk.ToolCallStatusInProgress),
				),
			}
		}
		if p.Item.Type == "mcpToolCall" {
			title := p.Item.ToolName
			if title == "" {
				title = p.Item.Name
			}
			if title == "" {
				title = "MCP tool call"
			}
			if p.Item.Server != "" {
				title = p.Item.Server + "." + title
			}
			opts := []acpsdk.ToolCallStartOpt{
				acpsdk.WithStartKind(acpsdk.ToolKindOther),
				acpsdk.WithStartStatus(acpsdk.ToolCallStatusInProgress),
			}
			if p.Item.Arguments != nil {
				opts = append(opts, acpsdk.WithStartRawInput(p.Item.Arguments))
			}
			return []acpsdk.SessionUpdate{
				acpsdk.StartToolCall(acpsdk.ToolCallId(p.Item.ID), title, opts...),
			}
		}

	case "item/mcpToolCall/progress":
		var p struct {
			ItemID   string `json:"itemId,omitempty"`
			ID       string `json:"id,omitempty"`
			Progress string `json:"progress,omitempty"`
			Delta    string `json:"delta,omitempty"`
			Message  string `json:"message,omitempty"`
			Output   string `json:"output,omitempty"`
		}
		if json.Unmarshal(params, &p) != nil {
			return nil
		}
		id := p.ItemID
		if id == "" {
			id = p.ID
		}
		if id == "" {
			return nil
		}
		progress := p.Progress
		if progress == "" {
			progress = p.Message
		}
		if progress == "" {
			progress = p.Delta
		}
		if progress == "" {
			progress = p.Output
		}
		opts := []acpsdk.ToolCallUpdateOpt{
			acpsdk.WithUpdateStatus(acpsdk.ToolCallStatusInProgress),
		}
		if progress != "" {
			opts = append(opts, acpsdk.WithUpdateRawOutput(progress))
		}
		return []acpsdk.SessionUpdate{
			acpsdk.UpdateToolCall(acpsdk.ToolCallId(id), opts...),
		}

	case "item/completed":
		var p struct {
			Item struct {
				ID               string `json:"id"`
				Type             string `json:"type"`
				Text             string `json:"text,omitempty"`
				Command          string `json:"command,omitempty"`
				AggregatedOutput string `json:"aggregatedOutput,omitempty"`
				ExitCode         *int   `json:"exitCode,omitempty"`
				Error            string `json:"error,omitempty"`
				IsError          bool   `json:"isError,omitempty"`
				Result           any    `json:"result,omitempty"`
				Output           any    `json:"output,omitempty"`
				Status           string `json:"status,omitempty"`
				Changes          []struct {
					Path string `json:"path"`
					Diff string `json:"diff"`
				} `json:"changes,omitempty"`
			} `json:"item"`
		}
		if json.Unmarshal(params, &p) != nil {
			return nil
		}
		switch p.Item.Type {
		case "agentMessage":
			// Text already sent via item/agentMessage/delta events; don't re-send.
			return nil
		case "reasoning":
			// Thought text already sent via item/reasoning/delta events; don't re-send.
			return nil
		case "commandExecution":
			status := acpsdk.ToolCallStatusCompleted
			if p.Item.ExitCode != nil && *p.Item.ExitCode != 0 {
				status = acpsdk.ToolCallStatusFailed
			}
			return []acpsdk.SessionUpdate{
				acpsdk.UpdateToolCall(
					acpsdk.ToolCallId(p.Item.ID),
					acpsdk.WithUpdateStatus(status),
					acpsdk.WithUpdateRawOutput(p.Item.AggregatedOutput),
				),
			}
		case "fileChange":
			var content []acpsdk.ToolCallContent
			for _, c := range p.Item.Changes {
				content = append(content, acpsdk.ToolDiffContent(c.Path, c.Diff))
			}
			return []acpsdk.SessionUpdate{
				acpsdk.UpdateToolCall(
					acpsdk.ToolCallId(p.Item.ID),
					acpsdk.WithUpdateStatus(acpsdk.ToolCallStatusCompleted),
					acpsdk.WithUpdateContent(content),
				),
			}
		case "mcpToolCall":
			status := acpsdk.ToolCallStatusCompleted
			if p.Item.IsError || p.Item.Error != "" || p.Item.Status == "failed" || p.Item.Status == "error" {
				status = acpsdk.ToolCallStatusFailed
			}
			output := p.Item.Result
			if output == nil {
				output = p.Item.Output
			}
			opts := []acpsdk.ToolCallUpdateOpt{
				acpsdk.WithUpdateStatus(status),
			}
			if output != nil {
				opts = append(opts, acpsdk.WithUpdateRawOutput(output))
			} else if p.Item.Error != "" {
				opts = append(opts, acpsdk.WithUpdateRawOutput(p.Item.Error))
			}
			return []acpsdk.SessionUpdate{
				acpsdk.UpdateToolCall(acpsdk.ToolCallId(p.Item.ID), opts...),
			}
		}

	case "codex/event/mcp_startup_update", "codex/event/mcp_startup_complete":
		var p map[string]any
		if json.Unmarshal(params, &p) != nil {
			return nil
		}
		text := formatMCPStartupUpdate(method, p)
		if text == "" {
			return nil
		}
		return []acpsdk.SessionUpdate{acpsdk.UpdateAgentThoughtText(text)}
	}

	return nil
}

func (a *Adapter) sendUpdate(ctx context.Context, sessionID acpsdk.SessionId, update acpsdk.SessionUpdate) {
	if a.conn == nil {
		return
	}
	if err := a.conn.SessionUpdate(ctx, acpsdk.SessionNotification{
		SessionId: sessionID,
		Update:    update,
	}); err != nil {
		a.log.Debug("failed to send session update", "error", err)
	}
}

var _ acpsdk.Agent = (*Adapter)(nil)

func formatMCPStartupUpdate(method string, params map[string]any) string {
	var parts []string
	if server, ok := params["server"].(string); ok && server != "" {
		parts = append(parts, server)
	}
	for _, key := range []string{"message", "status", "detail", "error"} {
		if s, ok := params[key].(string); ok && s != "" {
			parts = append(parts, s)
			break
		}
	}
	if len(parts) == 0 {
		return ""
	}
	prefix := "[mcp startup]"
	if method == "codex/event/mcp_startup_complete" {
		prefix = "[mcp ready]"
	}
	return prefix + " " + strings.Join(parts, " - ")
}
