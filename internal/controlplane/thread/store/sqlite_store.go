package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/sebastianm/flowgentic/internal/controlplane/thread"
)

const timeFormat = "2006-01-02T15:04:05.000Z"

// SQLiteStore implements thread.Store using sqlc-generated queries.
type SQLiteStore struct {
	q *Queries
}

// NewSQLiteStore creates a SQLiteStore.
func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{q: New(db)}
}

func (s *SQLiteStore) ListThreads(ctx context.Context, projectID string) ([]thread.Thread, error) {
	rows, err := s.q.ListThreads(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("listing threads: %w", err)
	}

	threads := make([]thread.Thread, len(rows))
	for i, r := range rows {
		threads[i] = threadFromRow(r)
	}
	return threads, nil
}

func (s *SQLiteStore) GetThread(ctx context.Context, id string) (thread.Thread, error) {
	row, err := s.q.GetThread(ctx, id)
	if err != nil {
		return thread.Thread{}, fmt.Errorf("getting thread %q: %w", id, err)
	}
	return threadFromRow(row), nil
}

func (s *SQLiteStore) CreateThread(ctx context.Context, t thread.Thread) (thread.Thread, error) {
	now := time.Now().UTC()
	t.CreatedAt = now
	t.UpdatedAt = now

	nowStr := now.Format(timeFormat)
	err := s.q.CreateThread(ctx, CreateThreadParams{
		ID:        t.ID,
		ProjectID: t.ProjectID,
		Agent:     t.Agent,
		Model:     t.Model,
		Mode:      t.Mode,
		CreatedAt: nowStr,
		UpdatedAt: nowStr,
	})
	if err != nil {
		return thread.Thread{}, fmt.Errorf("inserting thread: %w", err)
	}
	return t, nil
}

func (s *SQLiteStore) UpdateThread(ctx context.Context, t thread.Thread) (thread.Thread, error) {
	now := time.Now().UTC()
	t.UpdatedAt = now

	res, err := s.q.UpdateThread(ctx, UpdateThreadParams{
		Agent:     t.Agent,
		Model:     t.Model,
		UpdatedAt: now.Format(timeFormat),
		ID:        t.ID,
	})
	if err != nil {
		return thread.Thread{}, fmt.Errorf("updating thread: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return thread.Thread{}, fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return thread.Thread{}, fmt.Errorf("thread %q not found", t.ID)
	}

	return s.GetThread(ctx, t.ID)
}

func (s *SQLiteStore) DeleteThread(ctx context.Context, id string) error {
	res, err := s.q.DeleteThread(ctx, id)
	if err != nil {
		return fmt.Errorf("deleting thread: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("thread %q not found", id)
	}
	return nil
}

func threadFromRow(r Thread) thread.Thread {
	createdAt, _ := time.Parse(timeFormat, r.CreatedAt)
	updatedAt, _ := time.Parse(timeFormat, r.UpdatedAt)
	return thread.Thread{
		ID:        r.ID,
		ProjectID: r.ProjectID,
		Agent:     r.Agent,
		Model:     r.Model,
		Mode:      r.Mode,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}
