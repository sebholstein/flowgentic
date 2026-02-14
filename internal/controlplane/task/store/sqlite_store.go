package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/sebastianm/flowgentic/internal/controlplane/task"
)

const timeFormat = "2006-01-02T15:04:05.000Z"

type SQLiteStore struct {
	db *sql.DB
	q  *Queries
}

func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db, q: New(db)}
}

func (s *SQLiteStore) CreateTask(ctx context.Context, t task.Task) error {
	return s.q.CreateTask(ctx, CreateTaskParams{
		ID:          t.ID,
		ThreadID:    t.ThreadID,
		Description: t.Description,
		Subtasks:    task.MarshalSubtasks(t.Subtasks),
		Memory:      t.Memory,
		Status:      t.Status,
		SortIndex:   int64(t.SortIndex),
		CreatedAt:   t.CreatedAt.Format(timeFormat),
		UpdatedAt:   t.UpdatedAt.Format(timeFormat),
	})
}

func (s *SQLiteStore) GetTask(ctx context.Context, id string) (task.Task, error) {
	row, err := s.q.GetTask(ctx, id)
	if err != nil {
		return task.Task{}, fmt.Errorf("getting task %q: %w", id, err)
	}
	return taskFromRow(row), nil
}

func (s *SQLiteStore) ListTasksByThread(ctx context.Context, threadID string) ([]task.Task, error) {
	rows, err := s.q.ListTasksByThread(ctx, threadID)
	if err != nil {
		return nil, fmt.Errorf("listing tasks for thread %q: %w", threadID, err)
	}
	tasks := make([]task.Task, len(rows))
	for i, r := range rows {
		tasks[i] = taskFromRow(r)
	}
	return tasks, nil
}

func (s *SQLiteStore) UpdateTask(ctx context.Context, t task.Task) error {
	res, err := s.q.UpdateTask(ctx, UpdateTaskParams{
		Description: t.Description,
		Subtasks:    task.MarshalSubtasks(t.Subtasks),
		Memory:      t.Memory,
		Status:      t.Status,
		SortIndex:   int64(t.SortIndex),
		UpdatedAt:   t.UpdatedAt.Format(timeFormat),
		ID:          t.ID,
	})
	if err != nil {
		return fmt.Errorf("updating task: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("task %q not found", t.ID)
	}
	return nil
}

func (s *SQLiteStore) DeleteTask(ctx context.Context, id string) error {
	return s.q.DeleteTask(ctx, id)
}

func taskFromRow(r Task) task.Task {
	createdAt, _ := time.Parse(timeFormat, r.CreatedAt)
	updatedAt, _ := time.Parse(timeFormat, r.UpdatedAt)
	return task.Task{
		ID:          r.ID,
		ThreadID:    r.ThreadID,
		Description: r.Description,
		Subtasks:    task.UnmarshalSubtasks(r.Subtasks),
		Memory:      r.Memory,
		Status:      r.Status,
		SortIndex:   int32(r.SortIndex),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}
