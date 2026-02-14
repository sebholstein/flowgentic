package workload

import (
	"context"
	"fmt"

	acp "github.com/coder/acp-go-sdk"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
	v2 "github.com/sebastianm/flowgentic/internal/worker/driver/v2"
)

// Subscribe returns a channel that receives a StateEvent on every state change.
func (s *WorkloadService) Subscribe() chan StateEvent {
	return s.mgr.Subscribe()
}

// Unsubscribe removes a subscriber channel.
func (s *WorkloadService) Unsubscribe(ch chan StateEvent) {
	s.mgr.Unsubscribe(ch)
}

// GetStateSnapshot returns the current state of all sessions.
func (s *WorkloadService) GetStateSnapshot() []SessionSnapshot {
	return s.mgr.GetStateSnapshot()
}

// HandleSetTopic updates the topic for the given session.
func (s *WorkloadService) HandleSetTopic(ctx context.Context, sessionID, topic string) error {
	return s.mgr.HandleSetTopic(ctx, sessionID, topic)
}

// SetSessionMode changes the permission mode of a running session.
func (s *WorkloadService) SetSessionMode(ctx context.Context, sessionID string, mode driver.SessionMode) error {
	return s.mgr.SetSessionMode(ctx, sessionID, mode)
}

// Prompt sends a follow-up prompt to a running session.
func (s *WorkloadService) Prompt(ctx context.Context, sessionID string, blocks []acp.ContentBlock) (*acp.PromptResponse, error) {
	return s.mgr.Prompt(ctx, sessionID, blocks)
}

// Cancel cancels the active prompt on a running session.
func (s *WorkloadService) Cancel(ctx context.Context, sessionID string) error {
	return s.mgr.Cancel(ctx, sessionID)
}

// SubscribeEvents returns a channel that receives session events.
func (s *WorkloadService) SubscribeEvents() chan SessionEventUpdate {
	return s.mgr.SubscribeEvents()
}

// UnsubscribeEvents removes an event subscriber channel.
func (s *WorkloadService) UnsubscribeEvents(ch chan SessionEventUpdate) {
	s.mgr.UnsubscribeEvents(ch)
}

// AllPendingEvents returns all pending events across all sessions.
func (s *WorkloadService) AllPendingEvents() map[string][]*workerv1.SessionEvent {
	return s.mgr.AllPendingEvents()
}

// AckEvents drops all events up to the given sequence for a session.
func (s *WorkloadService) AckEvents(sessionID string, sequence int64) {
	s.mgr.AckEvents(sessionID, sequence)
}

// CheckSessionResumable checks if a session can be resumed.
func (s *WorkloadService) CheckSessionResumable(agentID, agentSessionID, cwd string) (bool, string) {
	return s.mgr.CheckSessionResumable(agentID, agentSessionID, cwd)
}

// LaunchResult describes the outcome of launching a workload.
type LaunchResult struct {
	Accepted       bool
	Message        string
	SessionID      string
	AgentID        string
	AgentSessionID string
	Status         string
}

// WorkloadService orchestrates workload scheduling on the worker.
type WorkloadService struct {
	mgr *SessionManager
}

func NewWorkloadService(mgr *SessionManager) *WorkloadService {
	return &WorkloadService{mgr: mgr}
}

// ListSessions returns info for all active sessions.
func (s *WorkloadService) ListSessions(_ context.Context) []SessionListEntry {
	return s.mgr.ListSessions()
}

// Schedule launches a workload via the SessionManager.
func (s *WorkloadService) Schedule(ctx context.Context, sessionID, agentID string, opts v2.LaunchOpts) (LaunchResult, error) {
	sess, err := s.mgr.Launch(ctx, sessionID, agentID, opts, nil)
	if err != nil {
		return LaunchResult{
			Accepted: false,
			Message:  fmt.Sprintf("launch failed: %v", err),
		}, nil
	}

	info := sess.Info()
	return LaunchResult{
		Accepted:       true,
		Message:        fmt.Sprintf("session launched on agent %s", agentID),
		SessionID:      sessionID,
		AgentID:        info.AgentID,
		AgentSessionID: info.AgentSessionID,
		Status:         string(info.Status),
	}, nil
}
