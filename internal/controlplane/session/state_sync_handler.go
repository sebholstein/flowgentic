package session

import (
	"context"
	"log/slog"
	"time"

	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
)

type TopicUpdater interface {
	UpdateTopic(ctx context.Context, id, topic string) error
}

// EventBroadcaster broadcasts raw session events to live subscribers.
type EventBroadcaster interface {
	BroadcastEvent(evt SessionEventUpdate)
}

// EventPersister persists raw session events to the store.
type EventPersister interface {
	InsertSessionEvent(ctx context.Context, evt SessionEvent) error
}

// SessionEventUpdate carries a raw session event from the worker for live subscribers.
type SessionEventUpdate struct {
	SessionID string
	Event     *workerv1.SessionEvent
}

type stateSyncHandler struct {
	log          *slog.Logger
	store        Store
	topicUpdater TopicUpdater
	persister    EventPersister
	broadcaster  EventBroadcaster
}

func NewStateSyncHandler(log *slog.Logger, store Store, topicUpdater TopicUpdater, broadcaster EventBroadcaster) StateSyncHandler {
	return &stateSyncHandler{
		log:          log,
		store:        store,
		topicUpdater: topicUpdater,
		persister:    store,
		broadcaster:  broadcaster,
	}
}

func (h *stateSyncHandler) HandleSnapshot(workerID string, sessions []*workerv1.SessionState) {
	for _, s := range sessions {
		h.processSessionUpdate(workerID, s)
	}
}

func (h *stateSyncHandler) HandleSessionUpdate(workerID string, s *workerv1.SessionState) {
	h.processSessionUpdate(workerID, s)
}

func (h *stateSyncHandler) HandleSessionRemoved(_ string, _ *workerv1.SessionRemoved) {
	// no-op: topic stays as the last known value
}

func (h *stateSyncHandler) HandleSessionEvent(_ string, event *workerv1.SessionEvent) {
	// 1. Persist raw event to DB.
	h.persistEvent(event)

	// 2. Forward to pub-sub for live frontend subscribers.
	h.broadcaster.BroadcastEvent(SessionEventUpdate{
		SessionID: event.GetSessionId(),
		Event:     event,
	})
}

func (h *stateSyncHandler) persistEvent(event *workerv1.SessionEvent) {
	record := WorkerEventToRecord(event)
	payload, err := MarshalRecord(record)
	if err != nil {
		h.log.Error("state sync: failed to marshal event record", "session_id", event.GetSessionId(), "error", err)
		return
	}

	evt := SessionEvent{
		SessionID: event.GetSessionId(),
		Sequence:  event.GetSequence(),
		EventType: record.Type,
		Payload:   payload,
		CreatedAt: time.Now(),
	}

	if err := h.persister.InsertSessionEvent(context.Background(), evt); err != nil {
		h.log.Error("state sync: failed to persist event",
			"session_id", event.GetSessionId(),
			"sequence", event.GetSequence(),
			"error", err,
		)
	}
}

func (h *stateSyncHandler) processSessionUpdate(_ string, state *workerv1.SessionState) {
	if state.Topic == "" {
		return
	}

	ctx := context.Background()

	sess, err := h.store.GetSession(ctx, state.SessionId)
	if err != nil {
		h.log.Warn("state sync: session not found", "session_id", state.SessionId, "error", err)
		return
	}

	if err := h.topicUpdater.UpdateTopic(ctx, sess.ThreadID, state.Topic); err != nil {
		h.log.Error("state sync: failed to update thread topic",
			"thread_id", sess.ThreadID,
			"session_id", state.SessionId,
			"topic", state.Topic,
			"error", err,
		)
		return
	}

	h.log.Info("state sync: thread topic updated",
		"thread_id", sess.ThreadID,
		"session_id", state.SessionId,
		"topic", state.Topic,
	)
}
