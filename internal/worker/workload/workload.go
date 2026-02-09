package workload

import (
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1/workerv1connect"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
)

// StartDeps holds the dependencies needed by the workload feature.
type StartDeps struct {
	Mux          *http.ServeMux
	Log          *slog.Logger
	Interceptors connect.HandlerOption
	Drivers      []driver.Driver
	CtlURL       string
	CtlSecret    string
}

// Start registers the WorkerService RPC handler on the mux and creates
// the WorkloadManager. It returns the manager so the caller can pass it
// to agentctl as the EventHandler.
func Start(d StartDeps) *WorkloadManager {
	mgr := NewWorkloadManager(d.Log, d.CtlURL, d.CtlSecret, d.Drivers...)
	svc := NewWorkloadService(mgr)
	h := &workerServiceHandler{log: d.Log, svc: svc}
	d.Mux.Handle(workerv1connect.NewWorkerServiceHandler(h, d.Interceptors))

	return mgr
}
