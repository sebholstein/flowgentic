package thread

import (
	"log/slog"
	"net/http"

	"github.com/sebastianm/flowgentic/internal/proto/gen/controlplane/v1/controlplanev1connect"
)

// StartDeps holds the dependencies needed by the thread management feature.
type StartDeps struct {
	Mux   *http.ServeMux
	Log   *slog.Logger
	Store Store
}

// Start registers the ThreadManagementService RPC handler on the mux.
func Start(d StartDeps) {
	svc := NewThreadService(d.Store)
	h := &threadManagementServiceHandler{log: d.Log, svc: svc}
	d.Mux.Handle(controlplanev1connect.NewThreadManagementServiceHandler(h))
}
