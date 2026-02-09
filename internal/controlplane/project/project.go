package project

import (
	"log/slog"
	"net/http"

	"github.com/sebastianm/flowgentic/internal/proto/gen/controlplane/v1/controlplanev1connect"
)

// StartDeps holds the dependencies needed by the project management feature.
type StartDeps struct {
	Mux   *http.ServeMux
	Log   *slog.Logger
	Store Store
}

// Start registers the ProjectManagementService RPC handler on the mux.
func Start(d StartDeps) {
	svc := NewProjectService(d.Store)
	h := &projectManagementServiceHandler{log: d.Log, svc: svc}
	d.Mux.Handle(controlplanev1connect.NewProjectManagementServiceHandler(h))
}
