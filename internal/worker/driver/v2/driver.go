package v2

import (
	"context"

	acp "github.com/coder/acp-go-sdk"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
)

// EventCallback receives ACP session notifications.
type EventCallback func(acp.SessionNotification)

// ModelInventory is authoritative model metadata from an ACP agent.
type ModelInventory struct {
	Models       []string
	DefaultModel string
}

// LaunchOpts configures a new agent session.
type LaunchOpts struct {
	Prompt          string
	SystemPrompt    string
	Model           string
	Cwd             string
	ResumeSessionID string // ACP agent session ID to resume; empty = new session
	SessionMode     string // "ask", "architect", "code"
	AllowedTools    []string
	MCPServers      []acp.McpServer
	EnvVars         map[string]string
	Handlers        *ClientHandlers
	StatusCh        chan<- SessionStatus // optional: receives status transitions (non-blocking send)
}

// Driver launches and manages ACP agent sessions.
type Driver interface {
	Agent() string
	Capabilities() driver.Capabilities
	Launch(ctx context.Context, opts LaunchOpts, onEvent EventCallback) (Session, error)
	DiscoverModels(ctx context.Context, cwd string) (ModelInventory, error)
}
