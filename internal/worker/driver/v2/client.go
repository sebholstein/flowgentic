package v2

import (
	"context"
	"fmt"
	"sync"

	acp "github.com/coder/acp-go-sdk"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
)

// flowgenticClient implements acp.Client.
// It forwards session updates to the EventCallback and handles permission requests
// by blocking until RespondToPermission is called.
type flowgenticClient struct {
	onEvent     EventCallback
	handlers    *ClientHandlers
	sessionMode driver.SessionMode

	mu          sync.Mutex
	permissions map[string]chan bool // requestID -> response channel
}

func newFlowgenticClient(onEvent EventCallback, handlers *ClientHandlers, sessionMode string) *flowgenticClient {
	mode := driver.SessionModeAsk
	if parsed, err := driver.ParseSessionMode(sessionMode); err == nil {
		mode = parsed
	}
	return &flowgenticClient{
		onEvent:      onEvent,
		handlers:     handlers,
		sessionMode:  mode,
		permissions:  make(map[string]chan bool),
	}
}

func (c *flowgenticClient) SessionUpdate(_ context.Context, n acp.SessionNotification) error {
	if c.onEvent != nil {
		c.onEvent(n)
	}
	return nil
}

func (c *flowgenticClient) RequestPermission(ctx context.Context, p acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	allowOptionID := findAllowOptionID(p.Options)

	// Emit permission request as a session update so the caller knows to prompt the user.
	requestID := string(p.ToolCall.ToolCallId)
	if c.onEvent != nil {
		title := ""
		if p.ToolCall.Title != nil {
			title = *p.ToolCall.Title
		}
		c.onEvent(acp.SessionNotification{
			SessionId: p.SessionId,
			Update: acp.SessionUpdate{
				ToolCall: &acp.SessionUpdateToolCall{
					ToolCallId:    p.ToolCall.ToolCallId,
					Title:         title,
					Kind:          derefToolKind(p.ToolCall.Kind),
					Status:        acp.ToolCallStatusPending,
					SessionUpdate: "tool_call",
					RawInput: map[string]any{
						"_permissionRequest": true,
						"requestId":          requestID,
						"options":            p.Options,
					},
				},
			},
		})
	}
	if c.shouldAutoApprovePermission() && allowOptionID != "" {
		if c.onEvent != nil {
			status := acp.ToolCallStatusCompleted
			c.onEvent(acp.SessionNotification{
				SessionId: p.SessionId,
				Update: acp.SessionUpdate{
					ToolCallUpdate: &acp.SessionToolCallUpdate{
						ToolCallId:    p.ToolCall.ToolCallId,
						Status:        &status,
						SessionUpdate: "tool_call_update",
					},
				},
			})
		}
		return acp.RequestPermissionResponse{
			Outcome: acp.NewRequestPermissionOutcomeSelected(allowOptionID),
		}, nil
	}

	// Create a channel and block until RespondToPermission resolves it.
	ch := make(chan bool, 1)
	c.mu.Lock()
	c.permissions[requestID] = ch
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.permissions, requestID)
		c.mu.Unlock()
	}()

	select {
	case <-ctx.Done():
		return acp.RequestPermissionResponse{
			Outcome: acp.NewRequestPermissionOutcomeCancelled(),
		}, nil
	case allowed := <-ch:
		if allowed && allowOptionID != "" {
			return acp.RequestPermissionResponse{
				Outcome: acp.NewRequestPermissionOutcomeSelected(allowOptionID),
			}, nil
		}
		return acp.RequestPermissionResponse{
			Outcome: acp.NewRequestPermissionOutcomeCancelled(),
		}, nil
	}
}

func findAllowOptionID(options []acp.PermissionOption) acp.PermissionOptionId {
	var allowAlwaysOptionID acp.PermissionOptionId
	for _, opt := range options {
		switch opt.Kind {
		case acp.PermissionOptionKindAllowOnce:
			return opt.OptionId
		case acp.PermissionOptionKindAllowAlways:
			if allowAlwaysOptionID == "" {
				allowAlwaysOptionID = opt.OptionId
			}
		}
	}
	return allowAlwaysOptionID
}

func (c *flowgenticClient) shouldAutoApprovePermission() bool {
	return c.sessionMode == driver.SessionModeArchitect || c.sessionMode == driver.SessionModeCode
}

// resolvePermission unblocks a pending RequestPermission call.
func (c *flowgenticClient) resolvePermission(requestID string, allow bool) error {
	c.mu.Lock()
	ch, ok := c.permissions[requestID]
	c.mu.Unlock()
	if !ok {
		return fmt.Errorf("no pending permission request: %s", requestID)
	}
	select {
	case ch <- allow:
	default:
	}
	return nil
}

// closePendingPermissions unblocks all pending permission requests (used on session stop).
func (c *flowgenticClient) closePendingPermissions() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, ch := range c.permissions {
		close(ch)
		delete(c.permissions, id)
	}
}

// Client capabilities — delegate to handlers when available, otherwise return errors.

func (c *flowgenticClient) ReadTextFile(ctx context.Context, req acp.ReadTextFileRequest) (acp.ReadTextFileResponse, error) {
	if c.handlers != nil && c.handlers.FS != nil {
		return c.handlers.FS.ReadTextFile(ctx, req)
	}
	return acp.ReadTextFileResponse{}, fmt.Errorf("fs.readTextFile not supported")
}

func (c *flowgenticClient) WriteTextFile(ctx context.Context, req acp.WriteTextFileRequest) (acp.WriteTextFileResponse, error) {
	if c.handlers != nil && c.handlers.FS != nil {
		return c.handlers.FS.WriteTextFile(ctx, req)
	}
	return acp.WriteTextFileResponse{}, fmt.Errorf("fs.writeTextFile not supported")
}

func (c *flowgenticClient) CreateTerminal(ctx context.Context, req acp.CreateTerminalRequest) (acp.CreateTerminalResponse, error) {
	if c.handlers != nil && c.handlers.Terminal != nil {
		return c.handlers.Terminal.CreateTerminal(ctx, req)
	}
	return acp.CreateTerminalResponse{}, fmt.Errorf("terminal not supported")
}

func (c *flowgenticClient) KillTerminalCommand(ctx context.Context, req acp.KillTerminalCommandRequest) (acp.KillTerminalCommandResponse, error) {
	if c.handlers != nil && c.handlers.Terminal != nil {
		return c.handlers.Terminal.KillTerminalCommand(ctx, req)
	}
	return acp.KillTerminalCommandResponse{}, fmt.Errorf("terminal not supported")
}

func (c *flowgenticClient) TerminalOutput(ctx context.Context, req acp.TerminalOutputRequest) (acp.TerminalOutputResponse, error) {
	if c.handlers != nil && c.handlers.Terminal != nil {
		return c.handlers.Terminal.TerminalOutput(ctx, req)
	}
	return acp.TerminalOutputResponse{}, fmt.Errorf("terminal not supported")
}

func (c *flowgenticClient) ReleaseTerminal(ctx context.Context, req acp.ReleaseTerminalRequest) (acp.ReleaseTerminalResponse, error) {
	if c.handlers != nil && c.handlers.Terminal != nil {
		return c.handlers.Terminal.ReleaseTerminal(ctx, req)
	}
	return acp.ReleaseTerminalResponse{}, fmt.Errorf("terminal not supported")
}

func (c *flowgenticClient) WaitForTerminalExit(ctx context.Context, req acp.WaitForTerminalExitRequest) (acp.WaitForTerminalExitResponse, error) {
	if c.handlers != nil && c.handlers.Terminal != nil {
		return c.handlers.Terminal.WaitForTerminalExit(ctx, req)
	}
	return acp.WaitForTerminalExitResponse{}, fmt.Errorf("terminal not supported")
}

func derefToolKind(k *acp.ToolKind) acp.ToolKind {
	if k != nil {
		return *k
	}
	return ""
}
