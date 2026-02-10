package agentrun

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
	"github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1/workerv1connect"
)

// WorkerRegistry looks up worker connection details by ID.
type WorkerRegistry interface {
	Lookup(workerID string) (url string, secret string, ok bool)
}

// Reconciler watches for pending agent runs and dispatches them to workers.
type Reconciler struct {
	log      *slog.Logger
	store    Store
	registry WorkerRegistry
	notify   chan struct{}
}

// NewReconciler creates a new Reconciler.
func NewReconciler(log *slog.Logger, store Store, registry WorkerRegistry) *Reconciler {
	return &Reconciler{
		log:      log,
		store:    store,
		registry: registry,
		notify:   make(chan struct{}, 1),
	}
}

// Notify wakes the reconciler to process pending runs.
func (r *Reconciler) Notify() {
	select {
	case r.notify <- struct{}{}:
	default: // already pending
	}
}

// Run starts the reconciler loop. It blocks until ctx is cancelled.
func (r *Reconciler) Run(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.notify:
		case <-ticker.C:
		}
		r.reconcileOnce(ctx)
	}
}

func (r *Reconciler) reconcileOnce(ctx context.Context) {
	runs, err := r.store.ListPendingAgentRuns(ctx, 10)
	if err != nil {
		r.log.Error("reconciler: failed to list pending runs", "error", err)
		return
	}

	for _, run := range runs {
		r.dispatchRun(ctx, run)
	}
}

func (r *Reconciler) dispatchRun(ctx context.Context, run AgentRun) {
	workerURL, secret, ok := r.registry.Lookup(run.WorkerID)
	if !ok {
		r.log.Error("reconciler: unknown worker", "worker_id", run.WorkerID, "agent_run_id", run.ID)
		if err := r.store.UpdateAgentRunStatus(ctx, run.ID, "failed", ""); err != nil {
			r.log.Error("reconciler: failed to mark run as failed", "agent_run_id", run.ID, "error", err)
		}
		return
	}

	// Mark as scheduling.
	if err := r.store.UpdateAgentRunStatus(ctx, run.ID, "scheduling", ""); err != nil {
		r.log.Error("reconciler: failed to mark run as scheduling", "agent_run_id", run.ID, "error", err)
		return
	}

	client := workerv1connect.NewWorkerServiceClient(
		http.DefaultClient,
		workerURL,
		connect.WithInterceptors(secretInterceptor(secret)),
	)

	resp, err := client.NewAgentRun(ctx, connect.NewRequest(&workerv1.NewAgentRunRequest{
		AgentRunId: run.ID,
		Agent:      agentToProto(run.Agent),
		Mode:       "headless",
		Prompt:     run.Prompt,
		Model:      run.Model,
		Yolo:       run.Yolo,
	}))
	if err != nil {
		r.log.Error("reconciler: NewAgentRun RPC failed", "agent_run_id", run.ID, "error", err)
		if err := r.store.UpdateAgentRunStatus(ctx, run.ID, "failed", ""); err != nil {
			r.log.Error("reconciler: failed to mark run as failed", "agent_run_id", run.ID, "error", err)
		}
		return
	}

	if !resp.Msg.Accepted {
		r.log.Warn("reconciler: worker rejected agent run", "agent_run_id", run.ID, "message", resp.Msg.Message)
		if err := r.store.UpdateAgentRunStatus(ctx, run.ID, "failed", ""); err != nil {
			r.log.Error("reconciler: failed to mark run as failed", "agent_run_id", run.ID, "error", err)
		}
		return
	}

	sessionID := resp.Msg.SessionId
	if err := r.store.UpdateAgentRunStatus(ctx, run.ID, "running", sessionID); err != nil {
		r.log.Error("reconciler: failed to mark run as running", "agent_run_id", run.ID, "error", err)
		return
	}

	r.log.Info("reconciler: dispatched agent run", "agent_run_id", run.ID, "session_id", sessionID)
}

// secretInterceptor injects the Authorization header into outgoing requests.
func secretInterceptor(secret string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+secret)
			return next(ctx, req)
		}
	}
}

// agentToProto converts a string agent name to the proto enum.
// Accepts proto enum names like "AGENT_CLAUDE_CODE".
func agentToProto(agent string) workerv1.Agent {
	if v, ok := workerv1.Agent_value[agent]; ok {
		return workerv1.Agent(v)
	}
	return workerv1.Agent_AGENT_UNSPECIFIED
}
