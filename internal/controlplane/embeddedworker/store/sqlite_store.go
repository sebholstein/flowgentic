package store

import (
	"context"
	"database/sql"
)

// SQLiteStore implements embeddedworker.EmbeddedWorkerConfigStore.
type SQLiteStore struct {
	q *Queries
}

// NewSQLiteStore creates a SQLiteStore.
func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{q: New(db)}
}

func (s *SQLiteStore) GetSecret(ctx context.Context) (string, error) {
	return s.q.GetSecret(ctx)
}

func (s *SQLiteStore) UpsertSecret(ctx context.Context, secret string) error {
	return s.q.UpsertSecret(ctx, secret)
}
