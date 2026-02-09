package driver

import "context"

// SessionMode specifies how the agent session runs.
type SessionMode string

const (
	SessionModeHeadless SessionMode = "headless"
)

// LaunchOpts configures a new agent session.
type LaunchOpts struct {
	Mode         SessionMode
	Prompt       string
	SystemPrompt string
	Model        string
	Cwd          string
	SessionID    string // empty = new session, non-empty = resume
	Yolo         bool
	AllowedTools []string
	EnvVars      map[string]string
}

// Driver launches agent sessions in headless mode.
type Driver interface {
	// ID returns the driver identifier (e.g. "claude-code", "codex").
	Agent() string
	// Capabilities returns the driver's capability descriptor.
	Capabilities() Capabilities
	// Launch starts a new agent session.
	Launch(ctx context.Context, opts LaunchOpts, onEvent EventCallback) (Session, error)
	// HandleHookEvent processes a raw hook event from the agent process.
	HandleHookEvent(ctx context.Context, sessionID string, event HookEvent) error
}

// SessionResolver discovers the agent session ID after launch.
// Drivers that set the ID upfront (e.g. Claude, OpenCode) don't need this.
// Drivers that cannot control the session ID (e.g. Codex, Gemini) implement
// this so the WorkloadManager can resolve the ID post-launch.
type SessionResolver interface {
	ResolveSessionID(ctx context.Context, cwd string) (string, error)
}
