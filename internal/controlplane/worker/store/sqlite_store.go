package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/sebastianm/flowgentic/internal/controlplane/worker"
)

const timeFormat = "2006-01-02T15:04:05.000Z"

// SQLiteStore implements worker.Store using sqlc-generated queries.
type SQLiteStore struct {
	q *Queries
}

// NewSQLiteStore creates a SQLiteStore.
func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{q: New(db)}
}

func (s *SQLiteStore) ListWorkers(ctx context.Context) ([]worker.Worker, error) {
	rows, err := s.q.ListWorkers(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing workers: %w", err)
	}

	workers := make([]worker.Worker, len(rows))
	for i, r := range rows {
		workers[i] = workerFromRow(r)
	}
	return workers, nil
}

func (s *SQLiteStore) CreateWorker(ctx context.Context, w worker.Worker) (worker.Worker, error) {
	now := time.Now().UTC()
	w.CreatedAt = now
	w.UpdatedAt = now

	nowStr := now.Format(timeFormat)
	err := s.q.CreateWorker(ctx, CreateWorkerParams{
		ID:        w.ID,
		Name:      w.Name,
		Url:       w.URL,
		Secret:    w.Secret,
		CreatedAt: nowStr,
		UpdatedAt: nowStr,
	})
	if err != nil {
		return worker.Worker{}, fmt.Errorf("inserting worker: %w", err)
	}
	return w, nil
}

func (s *SQLiteStore) UpdateWorker(ctx context.Context, w worker.Worker) (worker.Worker, error) {
	now := time.Now().UTC()
	w.UpdatedAt = now

	res, err := s.q.UpdateWorker(ctx, UpdateWorkerParams{
		Name:      w.Name,
		Url:       w.URL,
		Secret:    w.Secret,
		UpdatedAt: now.Format(timeFormat),
		ID:        w.ID,
	})
	if err != nil {
		return worker.Worker{}, fmt.Errorf("updating worker: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return worker.Worker{}, fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return worker.Worker{}, fmt.Errorf("worker %q not found", w.ID)
	}

	return s.GetWorker(ctx, w.ID)
}

func (s *SQLiteStore) DeleteWorker(ctx context.Context, id string) error {
	res, err := s.q.DeleteWorker(ctx, id)
	if err != nil {
		return fmt.Errorf("deleting worker: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("worker %q not found", id)
	}
	return nil
}

func (s *SQLiteStore) GetWorker(ctx context.Context, id string) (worker.Worker, error) {
	row, err := s.q.GetWorker(ctx, id)
	if err != nil {
		return worker.Worker{}, fmt.Errorf("getting worker %q: %w", id, err)
	}
	return workerFromRow(row), nil
}

func workerFromRow(r Worker) worker.Worker {
	createdAt, _ := time.Parse(timeFormat, r.CreatedAt)
	updatedAt, _ := time.Parse(timeFormat, r.UpdatedAt)
	return worker.Worker{
		ID:        r.ID,
		Name:      r.Name,
		URL:       r.Url,
		Secret:    r.Secret,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}
