package session

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	controlplanev1 "github.com/sebastianm/flowgentic/internal/proto/gen/controlplane/v1"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
	"github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1/workerv1connect"
)

type sessionServiceHandler struct {
	log   *slog.Logger
	svc   *SessionService
	store Store
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

func (h *sessionServiceHandler) PromptSession(
	ctx context.Context,
	req *connect.Request[controlplanev1.PromptSessionRequest],
) (*connect.Response[controlplanev1.PromptSessionResponse], error) {
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
		h.log.Error("PromptSession: no active session", "thread_id", threadID, "error", err)
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("no active session: %w", err))
	}

	h.log.Info("PromptSession: forwarding to worker", "thread_id", threadID, "session_id", sess.ID, "worker_id", sess.WorkerID, "session_status", sess.Status)

	// Persist the user message as a raw SessionEvent.
	h.persistUserMessage(sess.ID, text)

	// Forward to the worker.
	workerURL, secret, ok := h.svc.LookupWorker(sess.WorkerID)
	if !ok {
		h.log.Error("PromptSession: worker not reachable", "worker_id", sess.WorkerID)
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("worker %s not reachable", sess.WorkerID))
	}

	client := workerv1connect.NewWorkerServiceClient(
		http.DefaultClient,
		workerURL,
		connect.WithInterceptors(secretInterceptor(secret)),
	)
	_, err = client.PromptSession(ctx, connect.NewRequest(&workerv1.PromptSessionRequest{
		SessionId: sess.ID,
		ContentBlocks: []*workerv1.ContentBlock{
			{Type: "text", Text: text},
		},
	}))
	if err != nil {
		h.log.Error("PromptSession: forward to worker failed", "session_id", sess.ID, "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("forward to worker: %w", err))
	}
	h.log.Info("PromptSession: completed", "session_id", sess.ID)

	return connect.NewResponse(&controlplanev1.PromptSessionResponse{}), nil
}

// persistUserMessage creates and persists a synthetic UserMessage event.
func (h *sessionServiceHandler) persistUserMessage(sessionID, text string) {
	// Build a worker-side SessionEvent with UserMessage payload.
	// We use sequence=0 here; the store will assign a proper sequence via the unique index.
	// Actually, we need a proper sequence. We'll use the current max + 1.
	// For simplicity, use time-based sequence that won't conflict with worker events
	// (worker events use monotonic counters starting from 1).
	// A cleaner approach: query max sequence and add 1.
	now := time.Now().UTC()
	event := &workerv1.SessionEvent{
		SessionId: sessionID,
		Timestamp: now.Format(time.RFC3339Nano),
		Payload: &workerv1.SessionEvent_UserMessage{
			UserMessage: &workerv1.UserMessage{Text: text},
		},
	}

	payload, err := proto.Marshal(event)
	if err != nil {
		h.log.Error("persistUserMessage: marshal failed", "session_id", sessionID, "error", err)
		return
	}

	// Get next sequence by loading existing events and finding max.
	// This is simple and correct for the single-writer (CP) case.
	events, err := h.store.ListSessionEventsBySession(context.Background(), sessionID)
	if err != nil {
		h.log.Error("persistUserMessage: failed to load events", "session_id", sessionID, "error", err)
		return
	}
	var maxSeq int64
	for _, e := range events {
		if e.Sequence > maxSeq {
			maxSeq = e.Sequence
		}
	}
	seq := maxSeq + 1

	// Update the event with the correct sequence before persisting.
	event.Sequence = seq
	payload, _ = proto.Marshal(event)

	evt := SessionEvent{
		SessionID: sessionID,
		Sequence:  seq,
		EventType: "user_message",
		Payload:   payload,
		CreatedAt: now,
	}
	if err := h.store.InsertSessionEvent(context.Background(), evt); err != nil {
		h.log.Error("persistUserMessage: insert failed", "session_id", sessionID, "error", err)
	}
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

// deserializeAndConvertEvent deserializes a stored event payload and converts it to a CP-side SessionEvent.
func deserializeAndConvertEvent(e SessionEvent) (*controlplanev1.SessionEvent, error) {
	var workerEvent workerv1.SessionEvent
	if err := proto.Unmarshal(e.Payload, &workerEvent); err != nil {
		return nil, fmt.Errorf("unmarshal event: %w", err)
	}
	return workerEventToCPEvent(&workerEvent), nil
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
	case *workerv1.SessionEvent_ToolCallStart:
		tc := p.ToolCallStart
		cpTc := &controlplanev1.ToolCallStart{
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
		e.Payload = &controlplanev1.SessionEvent_ToolCallStart{ToolCallStart: cpTc}
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
		e.Payload = &controlplanev1.SessionEvent_ToolCallUpdate{ToolCallUpdate: cpTc}
	case *workerv1.SessionEvent_StatusChange:
		e.Payload = &controlplanev1.SessionEvent_StatusChange{
			StatusChange: &controlplanev1.StatusChange{Status: p.StatusChange.GetStatus().String()},
		}
	case *workerv1.SessionEvent_ModeChange:
		e.Payload = &controlplanev1.SessionEvent_ModeChange{
			ModeChange: &controlplanev1.ModeChange{ModeId: p.ModeChange.GetModeId()},
		}
	case *workerv1.SessionEvent_UserMessage:
		e.Payload = &controlplanev1.SessionEvent_UserMessage{
			UserMessage: &controlplanev1.UserMessage{Text: p.UserMessage.GetText()},
		}
	}

	return e
}
