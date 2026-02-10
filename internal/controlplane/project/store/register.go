package store

import (
	"database/sql"

	"github.com/sebastianm/flowgentic/internal/controlplane/project"
)

func init() {
	project.RegisterStoreFactory(func(db *sql.DB) project.Store {
		return NewSQLiteStore(db)
	})
}
