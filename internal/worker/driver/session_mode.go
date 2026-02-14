package driver

import "fmt"

// SessionMode represents a standardised session permission mode,
// aligned with the ACP session-mode IDs.
type SessionMode string

const (
	SessionModeAsk       SessionMode = "ask"       // Permission required before changes.
	SessionModeArchitect SessionMode = "architect"  // Plan-only, no implementation.
	SessionModeCode      SessionMode = "code"       // Full tool access, auto-approve.
)

// ParseSessionMode validates and returns a SessionMode from a string.
func ParseSessionMode(s string) (SessionMode, error) {
	switch SessionMode(s) {
	case SessionModeAsk, SessionModeArchitect, SessionModeCode:
		return SessionMode(s), nil
	default:
		return "", fmt.Errorf("unknown session mode: %q", s)
	}
}
