package worker

import (
	"log/slog"
	"net/http"

	"github.com/sebastianm/flowgentic/internal/proto/gen/controlplane/v1/controlplanev1connect"
)

// StartDeps holds the dependencies needed by the worker management feature.
type StartDeps struct {
	Mux          *http.ServeMux
	Log          *slog.Logger
	Store        Store
	Registry     RegistryUpdater
	PingRegistry WorkerRegistry
}

// Start registers the WorkerManagementService RPC handler on the mux.
func Start(d StartDeps) {
	svc := NewWorkerService(d.Store, d.Registry)
	pingSvc := NewPingService(d.PingRegistry)
	h := &workerManagementServiceHandler{log: d.Log, svc: svc, pingSvc: pingSvc}
	d.Mux.Handle(controlplanev1connect.NewWorkerManagementServiceHandler(h))
}
