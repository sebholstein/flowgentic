package session

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"connectrpc.com/connect"

	controlplanev1 "github.com/sebastianm/flowgentic/internal/proto/gen/controlplane/v1"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
	"github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1/workerv1connect"
)

// ThreadTopicUpdater updates a thread's topic (used after session creation).
type ThreadTopicUpdater interface {
	UpdateTopic(ctx context.Context, id, topic string) error
}

type sessionServiceHandler struct {
	log                *slog.Logger
	svc                *SessionService
	store              Store
	threadTopicUpdater ThreadTopicUpdater
}

func (h *sessionServiceHandler) CreateSession(
	ctx context.Context,
	req *connect.Request[controlplanev1.CreateSessionRequest],
) (*connect.Response[controlplanev1.CreateSessionResponse], error) {
	msg := req.Msg
	if msg.ThreadId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("thread_id is required"))
	}
	if msg.WorkerId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("worker_id is required"))
	}
	if msg.Prompt == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("prompt is required"))
	}
	if msg.Agent == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("agent is required"))
	}

	sessionID, err := h.svc.CreateSessionForThread(ctx, msg.ThreadId, msg.WorkerId, msg.Prompt, msg.Agent, msg.Model, msg.Mode, msg.SessionMode)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("creating session: %w", err))
	}

	// Derive topic from prompt and update thread.
	if topic := deriveInitialTopic(msg.Prompt); topic != "" {
		if err := h.threadTopicUpdater.UpdateTopic(ctx, msg.ThreadId, topic); err != nil {
			h.log.Error("failed to update thread topic", "thread_id", msg.ThreadId, "error", err)
		}
	}

	sess, err := h.svc.GetSession(ctx, sessionID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("fetching created session: %w", err))
	}

	return connect.NewResponse(&controlplanev1.CreateSessionResponse{
		Session: sessionToProto(sess),
	}), nil
}

func (h *sessionServiceHandler) GetSession(
	ctx context.Context,
	req *connect.Request[controlplanev1.GetSessionRequest],
) (*connect.Response[controlplanev1.GetSessionResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	s, err := h.svc.GetSession(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&controlplanev1.GetSessionResponse{
		Session: sessionToProto(s),
	}), nil
}

func (h *sessionServiceHandler) ListSessions(
	ctx context.Context,
	req *connect.Request[controlplanev1.ListSessionsRequest],
) (*connect.Response[controlplanev1.ListSessionsResponse], error) {
	if req.Msg.ThreadId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	sessions, err := h.svc.ListSessions(ctx, req.Msg.ThreadId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	pbSessions := make([]*controlplanev1.SessionConfig, len(sessions))
	for i, s := range sessions {
		pbSessions[i] = sessionToProto(s)
	}

	return connect.NewResponse(&controlplanev1.ListSessionsResponse{
		Sessions: pbSessions,
	}), nil
}

func (h *sessionServiceHandler) SetSessionMode(
	ctx context.Context,
	req *connect.Request[controlplanev1.SetSessionModeRequest],
) (*connect.Response[controlplanev1.SetSessionModeResponse], error) {
	sessionID := req.Msg.SessionId
	if sessionID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("session_id is required"))
	}
	modeID := req.Msg.ModeId
	if modeID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("mode_id is required"))
	}

	s, err := h.svc.GetSession(ctx, sessionID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("session not found: %w", err))
	}

	workerURL, secret, ok := h.svc.LookupWorker(s.WorkerID)
	if !ok {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("worker %s not reachable", s.WorkerID))
	}

	client := workerv1connect.NewWorkerServiceClient(
		http.DefaultClient,
		workerURL,
		connect.WithInterceptors(secretInterceptor(secret)),
	)
	_, err = client.SetSessionMode(ctx, connect.NewRequest(&workerv1.SetSessionModeRequest{
		SessionId: sessionID,
		ModeId:    modeID,
	}))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("forward to worker: %w", err))
	}

	return connect.NewResponse(&controlplanev1.SetSessionModeResponse{}), nil
}

func (h *sessionServiceHandler) SendUserMessage(
	ctx context.Context,
	req *connect.Request[controlplanev1.SendUserMessageRequest],
) (*connect.Response[controlplanev1.SendUserMessageResponse], error) {
	threadID := req.Msg.ThreadId
	if threadID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("thread_id is required"))
	}
	text := req.Msg.Text
	if text == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("text is required"))
	}

	sess, err := h.svc.FindActiveSessionForThread(ctx, threadID)
	if err != nil {
		h.log.Error("SendUserMessage: no active session", "thread_id", threadID, "error", err)
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("no active session: %w", err))
	}

	h.log.Info("SendUserMessage: forwarding to worker", "thread_id", threadID, "session_id", sess.ID, "worker_id", sess.WorkerID, "session_status", sess.Status)

	// Forward to the worker.
	workerURL, secret, ok := h.svc.LookupWorker(sess.WorkerID)
	if !ok {
		h.log.Error("SendUserMessage: worker not reachable", "worker_id", sess.WorkerID)
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("worker %s not reachable", sess.WorkerID))
	}

	client := workerv1connect.NewWorkerServiceClient(
		http.DefaultClient,
		workerURL,
		connect.WithInterceptors(secretInterceptor(secret)),
	)
	_, err = client.SendUserMessage(ctx, connect.NewRequest(&workerv1.SendUserMessageRequest{
		SessionId: sess.ID,
		ContentBlocks: []*workerv1.ContentBlock{
			{Type: "text", Text: text},
		},
	}))
	if err != nil {
		h.log.Error("SendUserMessage: forward to worker failed", "session_id", sess.ID, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("forward to worker: %w", err))
	}
	h.log.Info("SendUserMessage: completed", "session_id", sess.ID)

	return connect.NewResponse(&controlplanev1.SendUserMessageResponse{}), nil
}

func (h *sessionServiceHandler) WatchSessionEvents(
	ctx context.Context,
	req *connect.Request[controlplanev1.WatchSessionEventsRequest],
	stream *connect.ServerStream[controlplanev1.WatchSessionEventsResponse],
) error {
	msg := req.Msg

	// Build dynamic scope matcher for live events.
	matchesScope, err := h.buildScopeMatcher(msg)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}

	// Subscribe to live events first so we don't miss events while replaying history.
	ch := h.svc.SubscribeEvents()
	defer h.svc.UnsubscribeEvents(ch)

	// 1. Replay raw events from SQLite (history catch-up).
	events, err := h.svc.LoadEventHistory(ctx, msg.SessionId, msg.ThreadId, msg.TaskId)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}

	for _, e := range events {
		if e.Sequence <= msg.AfterSequence {
			continue
		}

		cpEvent, err := deserializeAndConvertEvent(e)
		if err != nil {
			h.log.Warn("watch session events: failed to deserialize event",
				"session_id", e.SessionID, "sequence", e.Sequence, "error", err)
			continue
		}

		if err := stream.Send(&controlplanev1.WatchSessionEventsResponse{
			Event:     cpEvent,
			IsHistory: true,
		}); err != nil {
			return err
		}
	}

	// 2. Live events via pub-sub.
	for {
		select {
		case <-ctx.Done():
			return nil
		case evt := <-ch:
			match, err := matchesScope(ctx, evt.SessionID)
			if err != nil {
				h.log.Warn("watch session events: scope match failed", "session_id", evt.SessionID, "error", err)
				continue
			}
			if !match {
				continue
			}
			if err := stream.Send(&controlplanev1.WatchSessionEventsResponse{
				Event:     workerEventToCPEvent(evt.Event),
				IsHistory: false,
			}); err != nil {
				return err
			}
		}
	}
}

// deserializeAndConvertEvent deserializes a stored JSON event payload and converts it to a CP-side SessionEvent.
func deserializeAndConvertEvent(e SessionEvent) (*controlplanev1.SessionEvent, error) {
	record, err := UnmarshalRecord(e.Payload)
	if err != nil {
		return nil, fmt.Errorf("unmarshal event record: %w", err)
	}
	return RecordToCPEvent(record), nil
}

func (h *sessionServiceHandler) buildScopeMatcher(
	msg *controlplanev1.WatchSessionEventsRequest,
) (func(context.Context, string) (bool, error), error) {
	setCount := 0
	if msg.SessionId != "" {
		setCount++
	}
	if msg.ThreadId != "" {
		setCount++
	}
	if msg.TaskId != "" {
		setCount++
	}
	if setCount != 1 {
		return nil, fmt.Errorf("exactly one of session_id, thread_id, or task_id must be set")
	}

	if msg.SessionId != "" {
		target := msg.SessionId
		return func(_ context.Context, sessionID string) (bool, error) {
			return sessionID == target, nil
		}, nil
	}

	threadID := msg.ThreadId
	taskID := msg.TaskId
	cache := make(map[string]bool)

	return func(ctx context.Context, sessionID string) (bool, error) {
		if matched, ok := cache[sessionID]; ok {
			return matched, nil
		}

		sess, err := h.svc.GetSession(ctx, sessionID)
		if err != nil {
			// Session may have been removed or not persisted yet; treat as non-match.
			cache[sessionID] = false
			return false, nil
		}

		matched := (threadID != "" && sess.ThreadID == threadID) ||
			(taskID != "" && sess.TaskID == taskID)
		cache[sessionID] = matched
		return matched, nil
	}, nil
}

func sessionToProto(s Session) *controlplanev1.SessionConfig {
	return &controlplanev1.SessionConfig{
		Id:             s.ID,
		ThreadId:       s.ThreadID,
		WorkerId:       s.WorkerID,
		Prompt:         s.Prompt,
		Status:         s.Status,
		Agent:          s.Agent,
		Model:          s.Model,
		Mode:           s.Mode,
		SessionMode:    s.SessionMode,
		AgentSessionId: s.SessionID,
		CreatedAt:      s.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
		UpdatedAt:      s.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
	}
}

// workerEventToCPEvent converts a worker-side SessionEvent to a CP-side SessionEvent.
func workerEventToCPEvent(w *workerv1.SessionEvent) *controlplanev1.SessionEvent {
	e := &controlplanev1.SessionEvent{
		SessionId: w.GetSessionId(),
		Sequence:  w.GetSequence(),
		Timestamp: w.GetTimestamp(),
	}

	switch p := w.Payload.(type) {
	case *workerv1.SessionEvent_AgentMessageChunk:
		e.Payload = &controlplanev1.SessionEvent_AgentMessageChunk{
			AgentMessageChunk: &controlplanev1.AgentMessageChunk{Text: p.AgentMessageChunk.GetText()},
		}
	case *workerv1.SessionEvent_AgentThoughtChunk:
		e.Payload = &controlplanev1.SessionEvent_AgentThoughtChunk{
			AgentThoughtChunk: &controlplanev1.AgentThoughtChunk{Text: p.AgentThoughtChunk.GetText()},
		}
	case *workerv1.SessionEvent_ToolCall:
		tc := p.ToolCall
		cpTc := &controlplanev1.ToolCall{
			ToolCallId: tc.GetToolCallId(),
			Title:      tc.GetTitle(),
			Kind:       controlplanev1.ToolCallKind(tc.GetKind()),
			RawInput:   tc.GetRawInput(),
			Status:     controlplanev1.ToolCallStatus(tc.GetStatus()),
		}
		for _, loc := range tc.GetLocations() {
			cpTc.Locations = append(cpTc.Locations, &controlplanev1.ToolCallLocation{
				Path: loc.GetPath(),
				Line: loc.GetLine(),
			})
		}
		for _, cb := range tc.GetContent() {
			cpTc.Content = append(cpTc.Content, workerContentBlockToCP(cb))
		}
		e.Payload = &controlplanev1.SessionEvent_ToolCall{ToolCall: cpTc}
	case *workerv1.SessionEvent_ToolCallUpdate:
		tc := p.ToolCallUpdate
		cpTc := &controlplanev1.ToolCallUpdate{
			ToolCallId: tc.GetToolCallId(),
			Title:      tc.GetTitle(),
			Status:     controlplanev1.ToolCallStatus(tc.GetStatus()),
			RawOutput:  tc.GetRawOutput(),
		}
		for _, loc := range tc.GetLocations() {
			cpTc.Locations = append(cpTc.Locations, &controlplanev1.ToolCallLocation{
				Path: loc.GetPath(),
				Line: loc.GetLine(),
			})
		}
		for _, cb := range tc.GetContent() {
			cpTc.Content = append(cpTc.Content, workerContentBlockToCP(cb))
		}
		e.Payload = &controlplanev1.SessionEvent_ToolCallUpdate{ToolCallUpdate: cpTc}
	case *workerv1.SessionEvent_StatusChange:
		e.Payload = &controlplanev1.SessionEvent_StatusChange{
			StatusChange: &controlplanev1.StatusChange{Status: p.StatusChange.GetStatus().String()},
		}
	case *workerv1.SessionEvent_CurrentModeUpdate:
		e.Payload = &controlplanev1.SessionEvent_CurrentModeUpdate{
			CurrentModeUpdate: &controlplanev1.CurrentModeUpdate{ModeId: p.CurrentModeUpdate.GetModeId()},
		}
	case *workerv1.SessionEvent_UserMessage:
		e.Payload = &controlplanev1.SessionEvent_UserMessage{
			UserMessage: &controlplanev1.UserMessage{Text: p.UserMessage.GetText()},
		}
	}

	return e
}

// workerContentBlockToCP converts a worker-side ToolCallContentBlock to a CP-side one.
func workerContentBlockToCP(cb *workerv1.ToolCallContentBlock) *controlplanev1.ToolCallContentBlock {
	switch b := cb.Block.(type) {
	case *workerv1.ToolCallContentBlock_Diff:
		return &controlplanev1.ToolCallContentBlock{
			Block: &controlplanev1.ToolCallContentBlock_Diff{
				Diff: &controlplanev1.ToolCallDiff{
					Path:    b.Diff.GetPath(),
					NewText: b.Diff.GetNewText(),
					OldText: b.Diff.GetOldText(),
				},
			},
		}
	case *workerv1.ToolCallContentBlock_Text:
		return &controlplanev1.ToolCallContentBlock{
			Block: &controlplanev1.ToolCallContentBlock_Text{
				Text: &controlplanev1.ToolCallText{
					Text: b.Text.GetText(),
				},
			},
		}
	default:
		return &controlplanev1.ToolCallContentBlock{}
	}
}
