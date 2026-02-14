package project

import (
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1/workerv1connect"
)

// StartDeps holds the dependencies needed by the project feature.
type StartDeps struct {
	Mux          *http.ServeMux
	Log          *slog.Logger
	Interceptors connect.HandlerOption
}

// Start registers the ProjectService RPC handler on the mux.
func Start(d StartDeps) {
	svc := NewProjectService()
	h := &projectServiceHandler{log: d.Log, svc: svc}
	d.Mux.Handle(workerv1connect.NewProjectServiceHandler(h, d.Interceptors))
}
