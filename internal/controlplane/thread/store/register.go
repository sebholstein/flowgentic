package store

import (
	"database/sql"

	"github.com/sebastianm/flowgentic/internal/controlplane/thread"
)

func init() {
	thread.RegisterStoreFactory(func(db *sql.DB) thread.Store {
		return NewSQLiteStore(db)
	})
}
