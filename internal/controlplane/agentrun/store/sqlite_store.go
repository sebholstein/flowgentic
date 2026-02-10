package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/sebastianm/flowgentic/internal/controlplane/agentrun"
)

const timeFormat = "2006-01-02T15:04:05.000Z"

// SQLiteStore implements agentrun.Store using sqlc-generated queries.
type SQLiteStore struct {
	q *Queries
}

// NewSQLiteStore creates a SQLiteStore.
func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{q: New(db)}
}

func (s *SQLiteStore) CreateAgentRun(ctx context.Context, r agentrun.AgentRun) error {
	var yolo int64
	if r.Yolo {
		yolo = 1
	}

	return s.q.CreateAgentRun(ctx, CreateAgentRunParams{
		ID:        r.ID,
		ThreadID:  r.ThreadID,
		WorkerID:  r.WorkerID,
		Prompt:    r.Prompt,
		Status:    r.Status,
		Agent:     r.Agent,
		Model:     r.Model,
		Mode:      r.Mode,
		Yolo:      yolo,
		CreatedAt: r.CreatedAt.Format(timeFormat),
		UpdatedAt: r.UpdatedAt.Format(timeFormat),
	})
}

func (s *SQLiteStore) GetAgentRun(ctx context.Context, id string) (agentrun.AgentRun, error) {
	row, err := s.q.GetAgentRun(ctx, id)
	if err != nil {
		return agentrun.AgentRun{}, fmt.Errorf("getting agent run %q: %w", id, err)
	}
	return agentRunFromRow(row), nil
}

func (s *SQLiteStore) ListAgentRunsByThread(ctx context.Context, threadID string) ([]agentrun.AgentRun, error) {
	rows, err := s.q.ListAgentRunsByThread(ctx, threadID)
	if err != nil {
		return nil, fmt.Errorf("listing agent runs for thread %q: %w", threadID, err)
	}

	runs := make([]agentrun.AgentRun, len(rows))
	for i, r := range rows {
		runs[i] = agentRunFromRow(r)
	}
	return runs, nil
}

func (s *SQLiteStore) ListPendingAgentRuns(ctx context.Context, limit int64) ([]agentrun.AgentRun, error) {
	rows, err := s.q.ListPendingAgentRuns(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("listing pending agent runs: %w", err)
	}

	runs := make([]agentrun.AgentRun, len(rows))
	for i, r := range rows {
		runs[i] = agentRunFromRow(r)
	}
	return runs, nil
}

func (s *SQLiteStore) UpdateAgentRunStatus(ctx context.Context, id, status, sessionID string) error {
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	res, err := s.q.UpdateAgentRunStatus(ctx, UpdateAgentRunStatusParams{
		Status:    status,
		SessionID: sessionID,
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return fmt.Errorf("updating agent run status: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("agent run %q not found", id)
	}
	return nil
}

func agentRunFromRow(r AgentRun) agentrun.AgentRun {
	createdAt, _ := time.Parse(timeFormat, r.CreatedAt)
	updatedAt, _ := time.Parse(timeFormat, r.UpdatedAt)
	return agentrun.AgentRun{
		ID:        r.ID,
		ThreadID:  r.ThreadID,
		WorkerID:  r.WorkerID,
		Prompt:    r.Prompt,
		Status:    r.Status,
		Agent:     r.Agent,
		Model:     r.Model,
		Mode:      r.Mode,
		Yolo:      r.Yolo != 0,
		SessionID: r.SessionID,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}
