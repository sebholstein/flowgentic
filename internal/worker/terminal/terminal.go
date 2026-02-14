package terminal

import (
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1/workerv1connect"
)

// StartDeps holds the dependencies needed by the terminal feature.
type StartDeps struct {
	Mux          *http.ServeMux
	Log          *slog.Logger
	Interceptors connect.HandlerOption
}

// Start registers the TerminalService RPC handler on the mux.
func Start(d StartDeps) {
	svc := NewTerminalService(d.Log)
	h := &terminalServiceHandler{log: d.Log, svc: svc}
	d.Mux.Handle(workerv1connect.NewTerminalServiceHandler(h, d.Interceptors))
}
