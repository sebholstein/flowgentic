package driver

import (
	"context"
	"time"
)

// SessionStatus represents the lifecycle state of a session.
type SessionStatus string

const (
	SessionStatusStarting SessionStatus = "starting"
	SessionStatusRunning  SessionStatus = "running"
	SessionStatusIdle     SessionStatus = "idle"
	SessionStatusStopping SessionStatus = "stopping"
	SessionStatusStopped  SessionStatus = "stopped"
	SessionStatusErrored  SessionStatus = "errored"
)

// SessionInfo describes a running or completed session.
type SessionInfo struct {
	ID             string        `json:"id"`
	AgentID        string        `json:"agent_id"`
	AgentSessionID string        `json:"agent_session_id,omitempty"`
	Status         SessionStatus `json:"status"`
	Mode           SessionMode   `json:"mode"`
	Cwd            string        `json:"cwd"`
	StartedAt      time.Time     `json:"started_at"`
}

// Session represents a running agent session.
type Session interface {
	// Info returns the session's current state.
	Info() SessionInfo
	// Stop gracefully stops the agent session.
	Stop(ctx context.Context) error
	// Wait blocks until the session completes.
	Wait(ctx context.Context) error
}

// AgentSessionIDSetter is implemented by sessions that support setting
// the agent session ID after launch (used with SessionResolver drivers).
type AgentSessionIDSetter interface {
	SetAgentSessionID(id string)
}
