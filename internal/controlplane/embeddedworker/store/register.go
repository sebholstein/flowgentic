package store

import (
	"database/sql"

	"github.com/sebastianm/flowgentic/internal/controlplane/embeddedworker"
)

func init() {
	embeddedworker.RegisterEmbeddedWorkerConfigStoreFactory(func(db *sql.DB) embeddedworker.EmbeddedWorkerConfigStore {
		return NewSQLiteStore(db)
	})
}
