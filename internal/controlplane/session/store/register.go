package store

import (
	"database/sql"

	"github.com/sebastianm/flowgentic/internal/controlplane/session"
)

func init() {
	session.RegisterStoreFactory(func(db *sql.DB) session.Store {
		return NewSQLiteStore(db)
	})
}
