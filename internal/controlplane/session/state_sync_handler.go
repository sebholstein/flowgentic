package session

import (
	"context"
	"log/slog"
	"strings"
	"sync"
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

// chunkAccumulator buffers consecutive text chunks of the same type for a session,
// so they can be flushed as a single merged row to SQLite.
type chunkAccumulator struct {
	sessionID string
	eventType string // "agent_message_chunk" or "agent_thought_chunk"
	text      strings.Builder
	lastSeq   int64
	timestamp string // from first chunk
}

type stateSyncHandler struct {
	log          *slog.Logger
	store        Store
	topicUpdater TopicUpdater
	persister    EventPersister
	broadcaster  EventBroadcaster

	mu            sync.Mutex
	pendingChunks map[string]*chunkAccumulator // sessionID → accumulator
}

func NewStateSyncHandler(log *slog.Logger, store Store, topicUpdater TopicUpdater, broadcaster EventBroadcaster) StateSyncHandler {
	return &stateSyncHandler{
		log:           log,
		store:         store,
		topicUpdater:  topicUpdater,
		persister:     store,
		broadcaster:   broadcaster,
		pendingChunks: make(map[string]*chunkAccumulator),
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
	// 1. Persist with chunk merging — consecutive chunks are buffered and flushed as one row.
	h.persistEventMerging(event)

	// 2. Forward every event as-is to pub-sub for live frontend streaming.
	h.broadcaster.BroadcastEvent(SessionEventUpdate{
		SessionID: event.GetSessionId(),
		Event:     event,
	})
}

// FlushAll flushes all pending chunk accumulators to the database.
// Called on connection close to ensure no buffered data is lost.
func (h *stateSyncHandler) FlushAll() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for sessionID := range h.pendingChunks {
		h.flushAccumulatorLocked(sessionID)
	}
}

func (h *stateSyncHandler) persistEventMerging(event *workerv1.SessionEvent) {
	record := WorkerEventToRecord(event)
	sessionID := event.GetSessionId()

	h.mu.Lock()
	defer h.mu.Unlock()

	if isChunkType(record.Type) {
		acc, exists := h.pendingChunks[sessionID]
		if exists && acc.eventType == record.Type {
			// Same chunk type — append text and update sequence.
			acc.text.WriteString(record.Text)
			acc.lastSeq = record.Sequence
		} else {
			// Different type or new session — flush existing, start new accumulator.
			if exists {
				h.flushAccumulatorLocked(sessionID)
			}
			acc = &chunkAccumulator{
				sessionID: sessionID,
				eventType: record.Type,
				lastSeq:   record.Sequence,
				timestamp: record.Timestamp,
			}
			acc.text.WriteString(record.Text)
			h.pendingChunks[sessionID] = acc
		}
		return
	}

	// Non-chunk event — flush any pending chunks first, then persist normally.
	if _, exists := h.pendingChunks[sessionID]; exists {
		h.flushAccumulatorLocked(sessionID)
	}

	h.persistRecordLocked(sessionID, record)
}

func (h *stateSyncHandler) flushAccumulatorLocked(sessionID string) {
	acc, exists := h.pendingChunks[sessionID]
	if !exists {
		return
	}
	delete(h.pendingChunks, sessionID)

	merged := SessionEventRecord{
		V:         eventRecordVersion,
		SessionID: acc.sessionID,
		Sequence:  acc.lastSeq,
		Timestamp: acc.timestamp,
		Type:      acc.eventType,
		Text:      acc.text.String(),
	}

	h.persistRecordLocked(sessionID, merged)
}

func (h *stateSyncHandler) persistRecordLocked(sessionID string, record SessionEventRecord) {
	payload, err := MarshalRecord(record)
	if err != nil {
		h.log.Error("state sync: failed to marshal event record", "session_id", sessionID, "error", err)
		return
	}

	evt := SessionEvent{
		SessionID: sessionID,
		Sequence:  record.Sequence,
		EventType: record.Type,
		Payload:   payload,
		CreatedAt: time.Now(),
	}

	if err := h.persister.InsertSessionEvent(context.Background(), evt); err != nil {
		h.log.Error("state sync: failed to persist event",
			"session_id", sessionID,
			"sequence", record.Sequence,
			"error", err,
		)
	}
}

func isChunkType(eventType string) bool {
	return eventType == "agent_message_chunk" || eventType == "agent_thought_chunk"
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
