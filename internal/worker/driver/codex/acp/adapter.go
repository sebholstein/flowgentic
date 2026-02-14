package acp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/google/uuid"
)

const (
	methodItemStarted         = "item/started"
	methodItemCompleted       = "item/completed"
	methodTurnCompleted       = "turn/completed"
	methodItemCommandApproval = "item/commandExecution/requestApproval"
	methodAgentMessageDelta   = "item/agentMessage/delta"
	methodReasoningTextDelta  = "item/reasoning/textDelta"
	methodMCPToolCallProgress = "item/mcpToolCall/progress"
	methodMCPStartupUpdate    = "codex/event/mcp_startup_update"
	methodMCPStartupComplete  = "codex/event/mcp_startup_complete"
	methodSessionConfigured   = "sessionConfigured"
	methodSessionConfiguredV2 = "session/configured"
	methodCommandsUpdated     = "codex/event/available_commands_update"
	methodSkillsUpdated       = "codex/event/skills_update_available"
)

type agentMessageDeltaParams struct {
	Delta string `json:"delta"`
}

type reasoningTextDeltaParams struct {
	Delta string `json:"delta"`
}

type itemStartedParams struct {
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

type mcpToolCallProgressParams struct {
	ItemID   string `json:"itemId,omitempty"`
	ID       string `json:"id,omitempty"`
	Progress string `json:"progress,omitempty"`
	Delta    string `json:"delta,omitempty"`
	Message  string `json:"message,omitempty"`
	Output   string `json:"output,omitempty"`
}

type fileChange struct {
	Path string `json:"path"`
	Diff string `json:"diff"`
}

type itemCompletedParams struct {
	Item struct {
		ID               string       `json:"id"`
		Type             string       `json:"type"`
		Text             string       `json:"text,omitempty"`
		Command          string       `json:"command,omitempty"`
		AggregatedOutput string       `json:"aggregatedOutput,omitempty"`
		ExitCode         *int         `json:"exitCode,omitempty"`
		Error            string       `json:"error,omitempty"`
		IsError          bool         `json:"isError,omitempty"`
		Result           any          `json:"result,omitempty"`
		Output           any          `json:"output,omitempty"`
		Status           string       `json:"status,omitempty"`
		Changes          []fileChange `json:"changes,omitempty"`
	} `json:"item"`
}

type commandApprovalParams struct {
	Command string `json:"command"`
}

type pendingPermission struct {
	serverRequestID int64
	ch              chan bool
}

type updateSender interface {
	SessionUpdate(ctx context.Context, n acpsdk.SessionNotification) error
}

type bridgeClient interface {
	start(ctx context.Context, envVars map[string]string) error
	threadStart(model, cwd, systemPrompt, sessionMode string, mcpServers []acpsdk.McpServer) (string, error)
	turnStart(threadID, prompt, cwd, sessionMode string) (string, error)
	turnInterrupt(threadID, turnID string) error
	respondToServerRequest(id int64, result any)
	request(method string, params any) (json.RawMessage, error)
	modelSnapshot() *acpsdk.SessionModelState
	availableCommandsSnapshot() []acpsdk.AvailableCommand
	doneChan() <-chan struct{}
	close()
}

type notificationHandler func(a *Adapter, params json.RawMessage) []acpsdk.SessionUpdate

var notificationHandlers = map[string]notificationHandler{
	methodAgentMessageDelta:   (*Adapter).handleAgentMessageDelta,
	methodReasoningTextDelta:  (*Adapter).handleReasoningDelta,
	methodItemStarted:         (*Adapter).handleItemStarted,
	methodMCPToolCallProgress: (*Adapter).handleMCPToolCallProgress,
	methodItemCompleted:       (*Adapter).handleItemCompleted,
	methodMCPStartupUpdate:    (*Adapter).handleMCPStartupUpdate,
	methodMCPStartupComplete:  (*Adapter).handleMCPStartupUpdate,
	methodSessionConfigured:   (*Adapter).handleAvailableCommandsUpdate,
	methodSessionConfiguredV2: (*Adapter).handleAvailableCommandsUpdate,
	methodCommandsUpdated:     (*Adapter).handleAvailableCommandsUpdate,
	methodSkillsUpdated:       (*Adapter).handleAvailableCommandsUpdate,
}

type Adapter struct {
	log  *slog.Logger
	conn atomic.Pointer[acpsdk.AgentSideConnection]

	updater updateSender

	mu     sync.Mutex
	server bridgeClient
	ctx    context.Context
	cancel context.CancelFunc

	threadID string
	turnID   string
	cwd      string

	latestAvailableCommands []acpsdk.AvailableCommand
	turnDoneCh              chan struct{}

	pendingPermissions   map[string]pendingPermission
	pendingPermissionsMu sync.Mutex

	bridgeFactory func(log *slog.Logger, dispatch func(threadID string, method string, params json.RawMessage, serverRequestID *int64)) bridgeClient
}

func NewAdapter(log *slog.Logger) acpsdk.Agent {
	return &Adapter{
		log:                log.With("adapter", "codex"),
		pendingPermissions: make(map[string]pendingPermission),
		bridgeFactory: func(log *slog.Logger, dispatch func(threadID string, method string, params json.RawMessage, serverRequestID *int64)) bridgeClient {
			return newBridge(log, dispatch)
		},
	}
}

func (a *Adapter) SetConnection(conn *acpsdk.AgentSideConnection) {
	a.conn.Store(conn)
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

	a.mu.Lock()
	a.ctx, a.cancel = context.WithCancel(ctx)
	a.mu.Unlock()

	b := a.bridgeFactory(a.log, a.dispatchNotification)
	if err := b.start(ctx, envVars); err != nil {
		return acpsdk.NewSessionResponse{}, fmt.Errorf("start app-server: %w", err)
	}

	a.mu.Lock()
	a.server = b
	a.mu.Unlock()

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
	if modelState := b.modelSnapshot(); modelState != nil {
		resp.Models = modelState
	}
	if cmds := b.availableCommandsSnapshot(); len(cmds) > 0 {
		a.setLatestAvailableCommands(cmds)
		a.sendUpdate(ctx, resp.SessionId, acpsdk.SessionUpdate{
			AvailableCommandsUpdate: &acpsdk.SessionAvailableCommandsUpdate{
				AvailableCommands: cmds,
			},
		})
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
	adapterCtx := a.ctx
	a.mu.Unlock()

	if srv == nil {
		return acpsdk.PromptResponse{}, fmt.Errorf("no app-server running")
	}

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

	turnDone := make(chan struct{})
	a.mu.Lock()
	a.turnDoneCh = turnDone
	a.mu.Unlock()

	select {
	case <-turnDone:
		return acpsdk.PromptResponse{StopReason: acpsdk.StopReasonEndTurn}, nil
	case <-ctx.Done():
		return acpsdk.PromptResponse{StopReason: acpsdk.StopReasonCancelled}, nil
	case <-adapterCtx.Done():
		return acpsdk.PromptResponse{StopReason: acpsdk.StopReasonEndTurn}, nil
	case <-srv.doneChan():
		return acpsdk.PromptResponse{StopReason: acpsdk.StopReasonEndTurn}, nil
	}
}

func (a *Adapter) SetSessionMode(_ context.Context, _ acpsdk.SetSessionModeRequest) (acpsdk.SetSessionModeResponse, error) {
	return acpsdk.SetSessionModeResponse{}, nil
}

func (a *Adapter) Close() error {
	a.mu.Lock()
	if a.cancel != nil {
		a.cancel()
	}
	srv := a.server
	a.mu.Unlock()

	if srv != nil {
		srv.close()
	}

	a.pendingPermissionsMu.Lock()
	for id, pp := range a.pendingPermissions {
		select {
		case pp.ch <- false:
		default:
		}
		delete(a.pendingPermissions, id)
	}
	a.pendingPermissionsMu.Unlock()

	return nil
}

func (a *Adapter) dispatchNotification(threadID string, method string, params json.RawMessage, serverRequestID *int64) {
	if threadID == "" {
		a.mu.Lock()
		threadID = a.threadID
		a.mu.Unlock()
	}
	sessionID := acpsdk.SessionId(threadID)

	if method == methodItemCommandApproval && serverRequestID != nil {
		a.handleApprovalRequest(sessionID, params, *serverRequestID)
		return
	}

	var updates []acpsdk.SessionUpdate
	if handler, ok := notificationHandlers[method]; ok {
		updates = handler(a, params)
		for _, update := range updates {
			a.sendUpdate(context.Background(), sessionID, update)
		}
	}

	if method == methodSkillsUpdated && len(updates) == 0 {
		a.refreshSkillsSnapshot(context.Background(), sessionID)
	}

	if method == methodTurnCompleted {
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
	conn := a.conn.Load()
	if conn == nil {
		return
	}

	a.mu.Lock()
	adapterCtx := a.ctx
	a.mu.Unlock()
	if adapterCtx == nil {
		adapterCtx = context.Background()
	}

	requestID := uuid.New().String()

	var p commandApprovalParams
	if err := json.Unmarshal(params, &p); err != nil {
		a.log.Debug("failed to unmarshal approval params", "error", err)
		return
	}

	ch := make(chan bool, 1)
	a.pendingPermissionsMu.Lock()
	a.pendingPermissions[requestID] = pendingPermission{
		serverRequestID: serverRequestID,
		ch:              ch,
	}
	a.pendingPermissionsMu.Unlock()

	go func() {
		ctx, cancel := context.WithCancel(adapterCtx)
		defer cancel()

		resp, err := conn.RequestPermission(ctx, acpsdk.RequestPermissionRequest{
			SessionId: sessionID,
			Options: []acpsdk.PermissionOption{
				{OptionId: "allow", Name: "Allow", Kind: acpsdk.PermissionOptionKindAllowOnce},
				{OptionId: "deny", Name: "Deny", Kind: acpsdk.PermissionOptionKindRejectOnce},
			},
			ToolCall: acpsdk.RequestPermissionToolCall{
				ToolCallId: acpsdk.ToolCallId(requestID),
				Title:      acpsdk.Ptr(p.Command),
				Kind:       acpsdk.Ptr(acpsdk.ToolKindExecute),
				RawInput:   map[string]string{"command": p.Command},
			},
		})

		a.pendingPermissionsMu.Lock()
		delete(a.pendingPermissions, requestID)
		a.pendingPermissionsMu.Unlock()

		if err != nil {
			a.log.Debug("permission request failed", "error", err)
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

func (a *Adapter) handleAgentMessageDelta(params json.RawMessage) []acpsdk.SessionUpdate {
	var p agentMessageDeltaParams
	if err := json.Unmarshal(params, &p); err != nil {
		a.log.Debug("failed to unmarshal agentMessageDelta", "error", err)
		return nil
	}
	return []acpsdk.SessionUpdate{acpsdk.UpdateAgentMessageText(p.Delta)}
}

func (a *Adapter) handleReasoningDelta(params json.RawMessage) []acpsdk.SessionUpdate {
	var p reasoningTextDeltaParams
	if err := json.Unmarshal(params, &p); err != nil {
		a.log.Debug("failed to unmarshal reasoningTextDelta", "error", err)
		return nil
	}
	return []acpsdk.SessionUpdate{acpsdk.UpdateAgentThoughtText(p.Delta)}
}

func (a *Adapter) handleItemStarted(params json.RawMessage) []acpsdk.SessionUpdate {
	var p itemStartedParams
	if err := json.Unmarshal(params, &p); err != nil {
		a.log.Debug("failed to unmarshal itemStarted", "error", err)
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
	return nil
}

func (a *Adapter) handleMCPToolCallProgress(params json.RawMessage) []acpsdk.SessionUpdate {
	var p mcpToolCallProgressParams
	if err := json.Unmarshal(params, &p); err != nil {
		a.log.Debug("failed to unmarshal mcpToolCallProgress", "error", err)
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
}

func (a *Adapter) handleItemCompleted(params json.RawMessage) []acpsdk.SessionUpdate {
	var p itemCompletedParams
	if err := json.Unmarshal(params, &p); err != nil {
		a.log.Debug("failed to unmarshal itemCompleted", "error", err)
		return nil
	}
	switch p.Item.Type {
	case "agentMessage":
		return nil
	case "reasoning":
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
	return nil
}

func (a *Adapter) handleMCPStartupUpdate(params json.RawMessage) []acpsdk.SessionUpdate {
	var p map[string]any
	if err := json.Unmarshal(params, &p); err != nil {
		a.log.Debug("failed to unmarshal mcpStartupUpdate", "error", err)
		return nil
	}
	text := formatMCPStartupUpdate("", p)
	if text == "" {
		return nil
	}
	return []acpsdk.SessionUpdate{acpsdk.UpdateAgentThoughtText(text)}
}

func (a *Adapter) handleAvailableCommandsUpdate(params json.RawMessage) []acpsdk.SessionUpdate {
	cmds := parseAvailableCommands(params)
	if len(cmds) == 0 {
		return nil
	}
	a.setLatestAvailableCommands(cmds)
	return []acpsdk.SessionUpdate{
		{
			AvailableCommandsUpdate: &acpsdk.SessionAvailableCommandsUpdate{
				AvailableCommands: cmds,
			},
		},
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

func (a *Adapter) sender() updateSender {
	if a.updater != nil {
		return a.updater
	}
	conn := a.conn.Load()
	if conn == nil {
		return nil
	}
	return conn
}

func (a *Adapter) setLatestAvailableCommands(cmds []acpsdk.AvailableCommand) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.latestAvailableCommands = append([]acpsdk.AvailableCommand(nil), cmds...)
}

func (a *Adapter) refreshSkillsSnapshot(ctx context.Context, sessionID acpsdk.SessionId) {
	a.mu.Lock()
	srv := a.server
	a.mu.Unlock()
	if srv == nil {
		return
	}

	// Ask Codex for a fresh skills list when it signals that the set changed.
	raw, err := srv.request("skills/list", map[string]any{
		"forceReload": true,
	})
	if err != nil {
		a.log.Debug("failed to refresh skills list", "error", err)
		return
	}
	cmds := parseAvailableCommands(raw)
	if len(cmds) == 0 {
		return
	}
	a.setLatestAvailableCommands(cmds)
	a.sendUpdate(ctx, sessionID, acpsdk.SessionUpdate{
		AvailableCommandsUpdate: &acpsdk.SessionAvailableCommandsUpdate{
			AvailableCommands: cmds,
		},
	})
}

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
	if method == methodMCPStartupComplete {
		prefix = "[mcp ready]"
	}
	return prefix + " " + strings.Join(parts, " - ")
}

var _ acpsdk.Agent = (*Adapter)(nil)
