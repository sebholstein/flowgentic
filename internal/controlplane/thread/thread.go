package thread

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/sebastianm/flowgentic/internal/proto/gen/controlplane/v1/controlplanev1connect"
)

// storeFactory is registered by the store subpackage via RegisterStoreFactory.
var storeFactory func(db *sql.DB) Store

// RegisterStoreFactory is called by the store subpackage to provide a Store
// constructor, avoiding an import cycle.
func RegisterStoreFactory(f func(db *sql.DB) Store) {
	storeFactory = f
}

// StartDeps holds the dependencies needed by the thread feature.
type StartDeps struct {
	Mux             *http.ServeMux
	Log             *slog.Logger
	DB              *sql.DB
	AgentRunCreator AgentRunCreator
}

// Start registers the ThreadService RPC handler on the mux.
func Start(d StartDeps) {
	st := storeFactory(d.DB)
	svc := NewThreadService(st)
	h := &threadServiceHandler{log: d.Log, svc: svc, agentRunCreator: d.AgentRunCreator}
	d.Mux.Handle(controlplanev1connect.NewThreadServiceHandler(h))
}
