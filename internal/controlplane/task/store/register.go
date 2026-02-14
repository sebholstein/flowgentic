package store

import (
	"database/sql"

	"github.com/sebastianm/flowgentic/internal/controlplane/task"
)

func init() {
	task.RegisterStoreFactory(func(db *sql.DB) task.Store {
		return NewSQLiteStore(db)
	})
}
