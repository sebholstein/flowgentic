package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/sebastianm/flowgentic/internal/controlplane/session"
)

const timeFormat = "2006-01-02T15:04:05.000Z"

type SQLiteStore struct {
	db *sql.DB
	q  *Queries
}

func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db, q: New(db)}
}

func (s *SQLiteStore) CreateSession(ctx context.Context, sess session.Session) error {
	return s.q.CreateSession(ctx, CreateSessionParams{
		ID:          sess.ID,
		ThreadID:    sess.ThreadID,
		WorkerID:    sess.WorkerID,
		Prompt:      sess.Prompt,
		Status:      sess.Status,
		Agent:       sess.Agent,
		Model:       sess.Model,
		Mode:        sess.Mode,
		SessionMode: sess.SessionMode,
		CreatedAt:   sess.CreatedAt.Format(timeFormat),
		UpdatedAt:   sess.UpdatedAt.Format(timeFormat),
	})
}

func (s *SQLiteStore) GetSession(ctx context.Context, id string) (session.Session, error) {
	row, err := s.q.GetSession(ctx, id)
	if err != nil {
		return session.Session{}, fmt.Errorf("getting session %q: %w", id, err)
	}
	return sessionFromRow(row), nil
}

func (s *SQLiteStore) ListSessionsByThread(ctx context.Context, threadID string) ([]session.Session, error) {
	rows, err := s.q.ListSessionsByThread(ctx, threadID)
	if err != nil {
		return nil, fmt.Errorf("listing sessions for thread %q: %w", threadID, err)
	}

	sessions := make([]session.Session, len(rows))
	for i, r := range rows {
		sessions[i] = sessionFromRow(r)
	}
	return sessions, nil
}

func (s *SQLiteStore) ListPendingSessions(ctx context.Context, limit int64) ([]session.Session, error) {
	rows, err := s.q.ListPendingSessions(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("listing pending sessions: %w", err)
	}

	sessions := make([]session.Session, len(rows))
	for i, r := range rows {
		sessions[i] = sessionFromRow(r)
	}
	return sessions, nil
}

func (s *SQLiteStore) UpdateSessionStatus(ctx context.Context, id, status, sessionID string) error {
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	res, err := s.q.UpdateSessionStatus(ctx, UpdateSessionStatusParams{
		Status:    status,
		SessionID: sessionID,
		UpdatedAt: now,
		ID:        id,
	})
	if err != nil {
		return fmt.Errorf("updating session status: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("session %q not found", id)
	}
	return nil
}

func (s *SQLiteStore) GetCwdForSession(ctx context.Context, sessionID string) (string, error) {
	sess, err := s.q.GetSession(ctx, sessionID)
	if err != nil {
		return "", fmt.Errorf("getting session %q: %w", sessionID, err)
	}

	var projectID string
	err = s.db.QueryRowContext(ctx,
		"SELECT project_id FROM threads WHERE id = ?", sess.ThreadID,
	).Scan(&projectID)
	if err != nil {
		return "", fmt.Errorf("getting project for thread %q: %w", sess.ThreadID, err)
	}

	path, err := s.q.GetWorkerProjectPath(ctx, GetWorkerProjectPathParams{
		ProjectID: projectID,
		WorkerID:  sess.WorkerID,
	})
	if err == nil && path != "" {
		return path, nil
	}

	return s.q.GetEmbeddedWorkerPathForSession(ctx, sessionID)
}

func (s *SQLiteStore) InsertSessionEvent(ctx context.Context, evt session.SessionEvent) error {
	return s.q.InsertSessionEvent(ctx, InsertSessionEventParams{
		SessionID: evt.SessionID,
		Sequence:  evt.Sequence,
		EventType: evt.EventType,
		Payload:   evt.Payload,
		CreatedAt: evt.CreatedAt.Format(timeFormat),
	})
}

func (s *SQLiteStore) ListSessionEventsBySession(ctx context.Context, sessionID string) ([]session.SessionEvent, error) {
	rows, err := s.q.ListSessionEventsBySession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("listing events for session %q: %w", sessionID, err)
	}
	return sessionEventsFromRows(rows), nil
}

func (s *SQLiteStore) ListSessionEventsByThread(ctx context.Context, threadID string) ([]session.SessionEvent, error) {
	rows, err := s.q.ListSessionEventsByThread(ctx, threadID)
	if err != nil {
		return nil, fmt.Errorf("listing events for thread %q: %w", threadID, err)
	}
	return sessionEventsFromRows(rows), nil
}

func (s *SQLiteStore) ListSessionEventsByTask(ctx context.Context, taskID sql.NullString) ([]session.SessionEvent, error) {
	rows, err := s.q.ListSessionEventsByTask(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("listing events for task: %w", err)
	}
	return sessionEventsFromRows(rows), nil
}

func sessionEventsFromRows(rows []SessionEvent) []session.SessionEvent {
	evts := make([]session.SessionEvent, len(rows))
	for i, r := range rows {
		createdAt, _ := time.Parse(timeFormat, r.CreatedAt)
		evts[i] = session.SessionEvent{
			SessionID: r.SessionID,
			Sequence:  r.Sequence,
			EventType: r.EventType,
			Payload:   r.Payload,
			CreatedAt: createdAt,
		}
	}
	return evts
}

func sessionFromRow(r Session) session.Session {
	createdAt, _ := time.Parse(timeFormat, r.CreatedAt)
	updatedAt, _ := time.Parse(timeFormat, r.UpdatedAt)
	return session.Session{
		ID:          r.ID,
		ThreadID:    r.ThreadID,
		WorkerID:    r.WorkerID,
		Prompt:      r.Prompt,
		Status:      r.Status,
		Agent:       r.Agent,
		Model:       r.Model,
		Mode:        r.Mode,
		SessionMode: r.SessionMode,
		SessionID:   r.SessionID,
		TaskID:      r.TaskID.String,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}
