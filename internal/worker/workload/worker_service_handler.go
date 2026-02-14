package workload

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	acp "github.com/coder/acp-go-sdk"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
	v2 "github.com/sebastianm/flowgentic/internal/worker/driver/v2"
)

// workerServiceHandler implements workerv1connect.WorkerServiceHandler.
type workerServiceHandler struct {
	log *slog.Logger
	svc *WorkloadService
}

var statusToProto = map[v2.SessionStatus]workerv1.SessionStatus{
	v2.SessionStatusStarting: workerv1.SessionStatus_SESSION_STATUS_STARTING,
	v2.SessionStatusRunning:  workerv1.SessionStatus_SESSION_STATUS_RUNNING,
	v2.SessionStatusIdle:     workerv1.SessionStatus_SESSION_STATUS_IDLE,
	v2.SessionStatusStopped:  workerv1.SessionStatus_SESSION_STATUS_STOPPED,
	v2.SessionStatusErrored:  workerv1.SessionStatus_SESSION_STATUS_ERRORED,
}

func (h *workerServiceHandler) ListSessions(
	ctx context.Context,
	_ *connect.Request[workerv1.ListSessionsRequest],
) (*connect.Response[workerv1.ListSessionsResponse], error) {
	entries := h.svc.ListSessions(ctx)
	sessions := make([]*workerv1.SessionInfo, 0, len(entries))
	for _, e := range entries {
		sessions = append(sessions, &workerv1.SessionInfo{
			SessionId:      e.SessionID,
			Agent:          driver.AgentType(e.Info.AgentID).ProtoAgent(),
			Status:         statusToProto[e.Info.Status],
			Mode:           workerv1.SessionMode_SESSION_MODE_HEADLESS,
			AgentSessionId: e.Info.AgentSessionID,
			Model:          e.Info.CurrentModel,
		})
	}
	return connect.NewResponse(&workerv1.ListSessionsResponse{
		Sessions: sessions,
	}), nil
}

func (h *workerServiceHandler) StateSync(
	ctx context.Context,
	stream *connect.BidiStream[workerv1.StateSyncRequest, workerv1.StateSyncResponse],
) error {
	h.log.Info("StateSync stream handler invoked")

	// Send initial full snapshot.
	snapshot := h.svc.GetStateSnapshot()
	states := make([]*workerv1.SessionState, 0, len(snapshot))
	for _, s := range snapshot {
		states = append(states, sessionSnapshotToProto(s))
	}
	if err := stream.Send(&workerv1.StateSyncResponse{
		Update: &workerv1.StateSyncResponse_Snapshot{
			Snapshot: &workerv1.SessionStateSnapshot{Sessions: states},
		},
	}); err != nil {
		h.log.Error("StateSync snapshot send failed", "error", err)
		return err
	}
	h.log.Info("StateSync snapshot sent", "sessions", len(states))

	// Send pending events from queue for all sessions.
	allPending := h.svc.AllPendingEvents()
	for _, events := range allPending {
		for _, e := range events {
			if err := stream.Send(&workerv1.StateSyncResponse{
				Update: &workerv1.StateSyncResponse_SessionEvent{
					SessionEvent: e,
				},
			}); err != nil {
				return err
			}
		}
	}

	// Subscribe to state changes and events.
	stateCh := h.svc.Subscribe()
	defer h.svc.Unsubscribe(stateCh)

	eventCh := h.svc.SubscribeEvents()
	defer h.svc.UnsubscribeEvents(eventCh)

	// Handle incoming ACKs from CP in a background goroutine.
	ackDone := make(chan struct{})
	go func() {
		defer close(ackDone)
		for {
			req, err := stream.Receive()
			if err != nil {
				return
			}
			if req.AckSessionId != "" {
				h.svc.AckEvents(req.AckSessionId, req.AckSequence)
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ackDone:
			return nil
		case event := <-stateCh:
			switch event.Type {
			case StateEventUpdate:
				if err := stream.Send(&workerv1.StateSyncResponse{
					Update: &workerv1.StateSyncResponse_SessionUpdate{
						SessionUpdate: sessionSnapshotToProto(*event.Snapshot),
					},
				}); err != nil {
					return err
				}
			case StateEventRemoved:
				if err := stream.Send(&workerv1.StateSyncResponse{
					Update: &workerv1.StateSyncResponse_SessionRemoved{
						SessionRemoved: &workerv1.SessionRemoved{
							SessionId:   event.SessionID,
							FinalStatus: "completed",
						},
					},
				}); err != nil {
					return err
				}
			}
		case evt := <-eventCh:
			if err := stream.Send(&workerv1.StateSyncResponse{
				Update: &workerv1.StateSyncResponse_SessionEvent{
					SessionEvent: evt.Event,
				},
			}); err != nil {
				return err
			}
		}
	}
}

func sessionSnapshotToProto(s SessionSnapshot) *workerv1.SessionState {
	return &workerv1.SessionState{
		SessionId:      s.SessionID,
		Agent:          driver.AgentType(s.Info.AgentID).ProtoAgent(),
		Status:         statusToProto[s.Info.Status],
		Mode:           workerv1.SessionMode_SESSION_MODE_HEADLESS,
		AgentSessionId: s.Info.AgentSessionID,
		Topic:          s.Topic,
	}
}

func (h *workerServiceHandler) SetSessionMode(
	ctx context.Context,
	req *connect.Request[workerv1.SetSessionModeRequest],
) (*connect.Response[workerv1.SetSessionModeResponse], error) {
	mode, err := driver.ParseSessionMode(req.Msg.ModeId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := h.svc.SetSessionMode(ctx, req.Msg.SessionId, mode); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&workerv1.SetSessionModeResponse{}), nil
}

func (h *workerServiceHandler) PromptSession(
	ctx context.Context,
	req *connect.Request[workerv1.PromptSessionRequest],
) (*connect.Response[workerv1.PromptSessionResponse], error) {
	// Convert proto ContentBlocks to ACP ContentBlocks.
	var blocks []acp.ContentBlock
	for _, b := range req.Msg.ContentBlocks {
		blocks = append(blocks, acp.TextBlock(b.Text))
	}

	resp, err := h.svc.Prompt(ctx, req.Msg.SessionId, blocks)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&workerv1.PromptSessionResponse{
		StopReason: string(resp.StopReason),
	}), nil
}

func (h *workerServiceHandler) CancelSession(
	ctx context.Context,
	req *connect.Request[workerv1.CancelSessionRequest],
) (*connect.Response[workerv1.CancelSessionResponse], error) {
	if err := h.svc.Cancel(ctx, req.Msg.SessionId); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&workerv1.CancelSessionResponse{}), nil
}

func (h *workerServiceHandler) CheckSessionResumable(
	_ context.Context,
	req *connect.Request[workerv1.CheckSessionResumableRequest],
) (*connect.Response[workerv1.CheckSessionResumableResponse], error) {
	resumable, reason := h.svc.CheckSessionResumable(req.Msg.Agent, req.Msg.AgentSessionId, req.Msg.Cwd)
	return connect.NewResponse(&workerv1.CheckSessionResumableResponse{
		Resumable: resumable,
		Reason:    reason,
	}), nil
}

func (h *workerServiceHandler) NewSession(
	ctx context.Context,
	req *connect.Request[workerv1.NewSessionRequest],
) (*connect.Response[workerv1.NewSessionResponse], error) {
	msg := req.Msg
	h.log.Info("NewSession called",
		"session_id", msg.SessionId,
		"agent", msg.Agent,
		"mode", msg.Mode,
		"cwd", msg.Cwd,
		"session_mode", msg.SessionMode,
		"allowed_tools", msg.AllowedTools,
	)

	agentType, err := driver.AgentTypeFromProto(msg.Agent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if msg.Mode != "" && msg.Mode != "headless" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unsupported mode %q: only headless mode is supported", msg.Mode))
	}

	opts := v2.LaunchOpts{
		Prompt:          msg.Prompt,
		SystemPrompt:    msg.SystemPrompt,
		Model:           msg.Model,
		Cwd:             msg.Cwd,
		ResumeSessionID: msg.AgentSessionId,
		SessionMode:     msg.SessionMode,
		AllowedTools:    msg.AllowedTools,
	}

	result, err := h.svc.Schedule(ctx, msg.SessionId, string(agentType), opts)
	if err != nil {
		h.log.Error("NewSession internal error", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	h.log.Info("NewSession result",
		"accepted", result.Accepted,
		"session_id", result.SessionID,
		"agent_id", result.AgentID,
		"agent_session_id", result.AgentSessionID,
		"status", result.Status,
		"message", result.Message,
	)

	return connect.NewResponse(&workerv1.NewSessionResponse{
		Accepted:       result.Accepted,
		Message:        result.Message,
		SessionId:      result.SessionID,
		Agent:          driver.AgentType(result.AgentID).ProtoAgent(),
		Status:         result.Status,
		Mode:           "headless",
		AgentSessionId: result.AgentSessionID,
	}), nil
}
