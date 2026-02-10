package agentrun

import (
	"context"
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

// StartDeps holds the dependencies needed by the agentrun feature.
type StartDeps struct {
	Mux      *http.ServeMux
	Log      *slog.Logger
	DB       *sql.DB
	Registry WorkerRegistry
}

// Start registers the AgentRunService RPC handler on the mux, starts the
// reconciler goroutine, and returns the AgentRunService so it can be passed
// to other features (e.g. thread) as AgentRunCreator.
func Start(ctx context.Context, d StartDeps) *AgentRunService {
	st := storeFactory(d.DB)
	reconciler := NewReconciler(d.Log, st, d.Registry)
	svc := NewAgentRunService(st, reconciler)
	h := &agentRunServiceHandler{log: d.Log, svc: svc}
	d.Mux.Handle(controlplanev1connect.NewAgentRunServiceHandler(h))

	go reconciler.Run(ctx)

	return svc
}
