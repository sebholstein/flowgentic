package systeminfo

import (
	"log/slog"
	"net/http"

	"connectrpc.com/connect"
	"github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1/workerv1connect"
	"github.com/sebastianm/flowgentic/internal/worker/systeminfo/agentinfo"
)

// StartDeps holds the dependencies needed by the systeminfo feature.
type StartDeps struct {
	Mux          *http.ServeMux
	Log          *slog.Logger
	Interceptors connect.HandlerOption
	Agents       agentinfo.AgentInfo
}

// Start registers the SystemService RPC handler on the mux.
func Start(d StartDeps) {
	svc := NewSystemInfoService(d.Agents)
	h := &systemServiceHandler{log: d.Log, svc: svc}
	d.Mux.Handle(workerv1connect.NewSystemServiceHandler(h, d.Interceptors))
}
