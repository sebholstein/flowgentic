package store

import (
	"database/sql"

	"github.com/sebastianm/flowgentic/internal/controlplane/agentrun"
)

func init() {
	agentrun.RegisterStoreFactory(func(db *sql.DB) agentrun.Store {
		return NewSQLiteStore(db)
	})
}
