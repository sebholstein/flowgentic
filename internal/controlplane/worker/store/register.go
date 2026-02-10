package store

import (
	"database/sql"

	"github.com/sebastianm/flowgentic/internal/controlplane/worker"
)

func init() {
	worker.RegisterStoreFactory(func(db *sql.DB) worker.Store {
		return NewSQLiteStore(db)
	})
}
