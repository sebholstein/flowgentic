package agentrun

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AgentRun is the domain representation of an agent run.
type AgentRun struct {
	ID        string
	ThreadID  string
	WorkerID  string
	Prompt    string
	Status    string
	Agent     string
	Model     string
	Mode      string
	Yolo      bool
	SessionID string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Store persists agent run records.
type Store interface {
	CreateAgentRun(ctx context.Context, r AgentRun) error
	GetAgentRun(ctx context.Context, id string) (AgentRun, error)
	ListAgentRunsByThread(ctx context.Context, threadID string) ([]AgentRun, error)
	ListPendingAgentRuns(ctx context.Context, limit int64) ([]AgentRun, error)
	UpdateAgentRunStatus(ctx context.Context, id, status, sessionID string) error
}

// AgentRunService implements the business logic for agent runs.
type AgentRunService struct {
	store      Store
	reconciler *Reconciler
}

// NewAgentRunService creates an AgentRunService.
func NewAgentRunService(store Store, reconciler *Reconciler) *AgentRunService {
	return &AgentRunService{store: store, reconciler: reconciler}
}

// CreateAgentRunForThread creates a new pending agent run and notifies the reconciler.
func (s *AgentRunService) CreateAgentRunForThread(ctx context.Context, threadID, workerID, prompt, agent, model, mode string, yolo bool) (string, error) {
	id := uuid.Must(uuid.NewV7()).String()
	now := time.Now().UTC()

	r := AgentRun{
		ID:        id,
		ThreadID:  threadID,
		WorkerID:  workerID,
		Prompt:    prompt,
		Status:    "pending",
		Agent:     agent,
		Model:     model,
		Mode:      mode,
		Yolo:      yolo,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.CreateAgentRun(ctx, r); err != nil {
		return "", fmt.Errorf("creating agent run: %w", err)
	}

	s.reconciler.Notify()
	return id, nil
}

// GetAgentRun returns a single agent run by ID.
func (s *AgentRunService) GetAgentRun(ctx context.Context, id string) (AgentRun, error) {
	return s.store.GetAgentRun(ctx, id)
}

// ListAgentRuns returns all agent runs for a thread.
func (s *AgentRunService) ListAgentRuns(ctx context.Context, threadID string) ([]AgentRun, error) {
	return s.store.ListAgentRunsByThread(ctx, threadID)
}
