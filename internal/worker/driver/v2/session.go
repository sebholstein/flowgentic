package v2

import (
	"context"
	"fmt"
	"sync"
	"time"

	acp "github.com/coder/acp-go-sdk"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
)

// SessionStatus represents the lifecycle state of a session.
type SessionStatus string

const (
	SessionStatusStarting SessionStatus = "starting"
	SessionStatusRunning  SessionStatus = "running"
	SessionStatusIdle     SessionStatus = "idle"
	SessionStatusStopped  SessionStatus = "stopped"
	SessionStatusErrored  SessionStatus = "errored"
)

// SessionInfo describes a running or completed session.
type SessionInfo struct {
	ID             string        `json:"id"`
	AgentID        string        `json:"agent_id"`
	AgentSessionID string        `json:"agent_session_id,omitempty"`
	Status         SessionStatus `json:"status"`
	Cwd            string        `json:"cwd"`
	StartedAt      time.Time     `json:"started_at"`
	Modes          []string      `json:"modes,omitempty"`  // available session modes
	Models         []string      `json:"models,omitempty"` // available models
	CurrentModel   string        `json:"current_model,omitempty"`
}

// Session represents a running ACP agent session.
type Session interface {
	Info() SessionInfo
	Prompt(ctx context.Context, blocks []acp.ContentBlock) (*acp.PromptResponse, error)
	Cancel(ctx context.Context) error
	Stop(ctx context.Context) error
	Wait(ctx context.Context) error
	RespondToPermission(ctx context.Context, requestID string, allow bool, reason string) error
	SetSessionMode(ctx context.Context, mode driver.SessionMode) error
}

// promptRequest is sent over promptCh to request a new prompt turn.
type promptRequest struct {
	blocks   []acp.ContentBlock
	resultCh chan promptResult
}

// promptResult carries the response from a prompt turn.
type promptResult struct {
	resp *acp.PromptResponse
	err  error
}

// acpSession implements Session over an ACP connection.
type acpSession struct {
	info     SessionInfo
	conn     *acp.ClientSideConnection
	client   *flowgenticClient
	cancel   context.CancelFunc
	done     chan struct{}
	statusCh chan<- SessionStatus // optional push-based status notifications

	promptCh chan promptRequest
	cancelCh chan struct{}

	mu sync.Mutex
}

func (s *acpSession) Info() SessionInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.info
}

func (s *acpSession) setStatus(status SessionStatus) {
	s.mu.Lock()
	prev := s.info.Status
	s.info.Status = status
	ch := s.statusCh
	s.mu.Unlock()
	if status != prev && ch != nil {
		select {
		case ch <- status:
		default:
		}
	}
}

func (s *acpSession) closeStatusCh() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.statusCh != nil {
		close(s.statusCh)
		s.statusCh = nil
	}
}

func (s *acpSession) Prompt(ctx context.Context, blocks []acp.ContentBlock) (*acp.PromptResponse, error) {
	req := promptRequest{
		blocks:   blocks,
		resultCh: make(chan promptResult, 1),
	}
	select {
	case s.promptCh <- req:
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.done:
		return nil, fmt.Errorf("session closed")
	}
	select {
	case res := <-req.resultCh:
		return res.resp, res.err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.done:
		return nil, fmt.Errorf("session closed")
	}
}

func (s *acpSession) Cancel(_ context.Context) error {
	select {
	case s.cancelCh <- struct{}{}:
	default:
	}
	return nil
}

func (s *acpSession) Stop(_ context.Context) error {
	s.cancel()
	<-s.done
	return nil
}

func (s *acpSession) Wait(_ context.Context) error {
	<-s.done
	return nil
}

func (s *acpSession) RespondToPermission(_ context.Context, requestID string, allow bool, _ string) error {
	return s.client.resolvePermission(requestID, allow)
}

func (s *acpSession) SetSessionMode(ctx context.Context, mode driver.SessionMode) error {
	s.mu.Lock()
	conn := s.conn
	sessionID := s.info.AgentSessionID
	s.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("session not connected")
	}

	_, err := conn.SetSessionMode(ctx, acp.SetSessionModeRequest{
		SessionId: acp.SessionId(sessionID),
		ModeId:    acp.SessionModeId(mode),
	})
	return err
}
