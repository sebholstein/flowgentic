package agentctl

import (
	"context"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	workerv1connect "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1/workerv1connect"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
)

// EventHandler defines the interface that the AgentRunManager satisfies,
// allowing agentctl RPC handlers to dispatch events without knowing the
// concrete manager type.
type EventHandler interface {
	HandleHookEvent(ctx context.Context, event driver.HookEvent) error
	HandleStatusReport(ctx context.Context, sessionID, agent, status string) error
	HandlePlanSubmission(ctx context.Context, sessionID, agent string, plan []byte) error
}

// StartDeps are the dependencies for starting the agentctl feature.
type StartDeps struct {
	Mux          *http.ServeMux
	Log          *slog.Logger
	Interceptors connect.HandlerOption
	Handler      EventHandler
}

// Start registers the agent-facing RPC handlers on the mux.
func Start(d StartDeps) {
	d.Mux.Handle(workerv1connect.NewHookCtlServiceHandler(
		&hookCtlServiceHandler{log: d.Log, handler: d.Handler}, d.Interceptors))
	d.Mux.Handle(workerv1connect.NewAgentCtlServiceHandler(
		&agentCtlServiceHandler{log: d.Log, handler: d.Handler}, d.Interceptors))
}
