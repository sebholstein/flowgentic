package session

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	"github.com/sebastianm/flowgentic/internal/controlplane/systemprompts"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
	"github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1/workerv1connect"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
)

// WorkerRegistry resolves a worker ID to its connection details.
type WorkerRegistry interface {
	Lookup(workerID string) (url string, secret string, ok bool)
}

// Reconciler watches for pending sessions and dispatches them to workers.
//
// It polls the store on a 5-second interval and can also be woken up
// immediately via Notify. For each pending session it:
//  1. Looks up the target worker in the registry.
//  2. Transitions the session to "scheduling".
//  3. Sends a NewSession RPC to the worker with the session's prompt,
//     system prompt, model, and allowed tools.
//  4. On success, transitions the session to "running" and records the
//     agent-side session ID. On failure (unknown worker, RPC error, or
//     worker rejection), marks the session as "failed".
type Reconciler struct {
	log      *slog.Logger
	store    Store
	registry WorkerRegistry
	notify   chan struct{}
}

// NewReconciler creates a reconciler that dispatches pending sessions to
// workers discovered through the given registry.
func NewReconciler(log *slog.Logger, store Store, registry WorkerRegistry) *Reconciler {
	return &Reconciler{
		log:      log,
		store:    store,
		registry: registry,
		notify:   make(chan struct{}, 1),
	}
}

// Notify wakes the reconciler to process pending sessions immediately.
// Non-blocking; coalesces multiple signals into one reconciliation pass.
func (r *Reconciler) Notify() {
	select {
	case r.notify <- struct{}{}:
	default:
	}
}

// Run starts the reconciliation loop. It blocks until ctx is cancelled.
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
	sessions, err := r.store.ListPendingSessions(ctx, 10)
	if err != nil {
		r.log.Error("reconciler: failed to list pending sessions", "error", err)
		return
	}

	for _, sess := range sessions {
		r.dispatchSession(ctx, sess)
	}
}

func (r *Reconciler) dispatchSession(ctx context.Context, sess Session) {
	workerURL, secret, ok := r.registry.Lookup(sess.WorkerID)
	if !ok {
		r.log.Error("reconciler: unknown worker", "worker_id", sess.WorkerID, "session_id", sess.ID)
		if err := r.store.UpdateSessionStatus(ctx, sess.ID, "failed", ""); err != nil {
			r.log.Error("reconciler: failed to mark session as failed", "session_id", sess.ID, "error", err)
		}
		return
	}

	if err := r.store.UpdateSessionStatus(ctx, sess.ID, "scheduling", ""); err != nil {
		r.log.Error("reconciler: failed to mark session as scheduling", "session_id", sess.ID, "error", err)
		return
	}

	cwd, err := r.store.GetCwdForSession(ctx, sess.ID)
	if err != nil {
		r.log.Error("reconciler: failed to resolve cwd", "session_id", sess.ID, "error", err)
	}

	client := workerv1connect.NewWorkerServiceClient(
		http.DefaultClient,
		workerURL,
		connect.WithInterceptors(secretInterceptor(secret)),
	)

	resp, err := client.NewSession(ctx, connect.NewRequest(&workerv1.NewSessionRequest{
		SessionId:    sess.ID,
		Agent:        agentToProto(sess.Agent),
		Mode:         "headless",
		Prompt:       sess.Prompt,
		SystemPrompt: renderSystemPrompt(r.log, sess),
		Model:        sess.Model,
		Cwd:          cwd,
		SessionMode:  orchestratedPlanSessionMode(sess.SessionMode),
		AllowedTools: flowgenticPlanAllowedTools(),
	}))
	if err != nil {
		r.log.Error("reconciler: NewSession RPC failed", "session_id", sess.ID, "error", err)
		if err := r.store.UpdateSessionStatus(ctx, sess.ID, "failed", ""); err != nil {
			r.log.Error("reconciler: failed to mark session as failed", "session_id", sess.ID, "error", err)
		}
		return
	}

	if !resp.Msg.Accepted {
		r.log.Warn("reconciler: worker rejected session", "session_id", sess.ID, "message", resp.Msg.Message)
		if err := r.store.UpdateSessionStatus(ctx, sess.ID, "failed", ""); err != nil {
			r.log.Error("reconciler: failed to mark session as failed", "session_id", sess.ID, "error", err)
		}
		return
	}

	agentSessionID := resp.Msg.AgentSessionId
	if err := r.store.UpdateSessionStatus(ctx, sess.ID, "running", agentSessionID); err != nil {
		r.log.Error("reconciler: failed to mark session as running", "session_id", sess.ID, "error", err)
		return
	}

	// Persist the initial prompt as the first UserMessage event so it appears in history.
	r.persistInitialPrompt(ctx, sess)

	r.log.Info("reconciler: dispatched session", "session_id", sess.ID, "agent_session_id", agentSessionID)
}

// persistInitialPrompt stores the session's initial prompt as a UserMessage event
// with sequence=1, so it appears as the first item in history replay.
func (r *Reconciler) persistInitialPrompt(ctx context.Context, sess Session) {
	if sess.Prompt == "" {
		return
	}

	now := time.Now().UTC()
	event := &workerv1.SessionEvent{
		SessionId: sess.ID,
		Sequence:  0, // sequence 0 = before any agent events (which start at 1)
		Timestamp: now.Format(time.RFC3339Nano),
		Payload: &workerv1.SessionEvent_UserMessage{
			UserMessage: &workerv1.UserMessage{Text: sess.Prompt},
		},
	}

	payload, err := proto.Marshal(event)
	if err != nil {
		r.log.Error("reconciler: failed to marshal initial prompt event", "session_id", sess.ID, "error", err)
		return
	}

	evt := SessionEvent{
		SessionID: sess.ID,
		Sequence:  0,
		EventType: "user_message",
		Payload:   payload,
		CreatedAt: now,
	}
	if err := r.store.InsertSessionEvent(ctx, evt); err != nil {
		r.log.Error("reconciler: failed to persist initial prompt", "session_id", sess.ID, "error", err)
	}
}

func renderSystemPrompt(log *slog.Logger, sess Session) string {
	prompt, err := systemprompts.RenderOrchestratedPlanMode(systemprompts.OrchestratedPlanModeData{
		CurrentPlanDir: systemprompts.DefaultPlanDirForSession(sess.ID),
		// Additional plan dirs are optional and injected only when known.
		AdditionalPlanDirs: nil,
	})
	if err != nil {
		log.Error("reconciler: failed to render orchestrated plan prompt", "session_id", sess.ID, "error", err)
		return ""
	}
	log.Info("reconciler: rendered system prompt", "session_id", sess.ID, "system_prompt", prompt)
	return prompt
}

func flowgenticPlanAllowedTools() []string {
	return []string{
		"mcp__flowgentic__set_topic",
		"mcp__flowgentic__ask_question",
		"mcp__flowgentic__plan_get_current_dir",
		"mcp__flowgentic__plan_request_thread_dir",
		"mcp__flowgentic__plan_remove_thread",
		"mcp__flowgentic__plan_clear_current",
		"mcp__flowgentic__plan_commit",
		"Read",
		"Write",
		"Edit",
		"MultiEdit",
		"Glob",
		"Grep",
		"LS",
	}
}

func orchestratedPlanSessionMode(_ string) string {
	// Orchestrated plan sessions should not run in "code" bypass mode because
	// that allows unrestricted tool usage and bypasses plan workflow controls.
	return "architect"
}

func secretInterceptor(secret string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+secret)
			return next(ctx, req)
		}
	}
}

func agentToProto(agent string) workerv1.Agent {
	v, err := driver.ParseProtoAgent(agent)
	if err != nil {
		return workerv1.Agent_AGENT_UNSPECIFIED
	}
	return v
}
