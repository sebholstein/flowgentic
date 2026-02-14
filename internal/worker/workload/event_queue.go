package workload

import (
	"sync"

	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
)

// sessionEventQueue holds events for a single session.
type sessionEventQueue struct {
	mu     sync.RWMutex
	events []*workerv1.SessionEvent
}

// EventQueue is a per-session event queue that buffers SessionEvents
// until they are acknowledged by the control plane.
type EventQueue struct {
	mu       sync.RWMutex
	sessions map[string]*sessionEventQueue
}

// NewEventQueue creates a new EventQueue.
func NewEventQueue() *EventQueue {
	return &EventQueue{
		sessions: make(map[string]*sessionEventQueue),
	}
}

// getOrCreate returns the sessionEventQueue for the given session,
// creating one if it does not exist. Must be called with q.mu held for writing.
func (q *EventQueue) getOrCreate(sessionID string) *sessionEventQueue {
	sq, ok := q.sessions[sessionID]
	if !ok {
		sq = &sessionEventQueue{}
		q.sessions[sessionID] = sq
	}
	return sq
}

// Append adds an event to the given session's queue, creating the session
// queue if it does not already exist.
func (q *EventQueue) Append(sessionID string, event *workerv1.SessionEvent) {
	q.mu.Lock()
	sq := q.getOrCreate(sessionID)
	q.mu.Unlock()

	sq.mu.Lock()
	sq.events = append(sq.events, event)
	sq.mu.Unlock()
}

// Pending returns all events for the given session whose Sequence is
// strictly greater than afterSeq.
func (q *EventQueue) Pending(sessionID string, afterSeq int64) []*workerv1.SessionEvent {
	q.mu.RLock()
	sq, ok := q.sessions[sessionID]
	q.mu.RUnlock()
	if !ok {
		return nil
	}

	sq.mu.RLock()
	defer sq.mu.RUnlock()

	var result []*workerv1.SessionEvent
	for _, e := range sq.events {
		if e.GetSequence() > afterSeq {
			result = append(result, e)
		}
	}
	return result
}

// Ack drops all events for the given session whose Sequence is less than
// or equal to the provided sequence number.
func (q *EventQueue) Ack(sessionID string, sequence int64) {
	q.mu.RLock()
	sq, ok := q.sessions[sessionID]
	q.mu.RUnlock()
	if !ok {
		return
	}

	sq.mu.Lock()
	defer sq.mu.Unlock()

	kept := sq.events[:0]
	for _, e := range sq.events {
		if e.GetSequence() > sequence {
			kept = append(kept, e)
		}
	}
	sq.events = kept
}

// Remove drops the entire event queue for the given session.
func (q *EventQueue) Remove(sessionID string) {
	q.mu.Lock()
	delete(q.sessions, sessionID)
	q.mu.Unlock()
}

// AllPending returns a snapshot of all pending events across every session.
// The returned map is keyed by session ID.
func (q *EventQueue) AllPending() map[string][]*workerv1.SessionEvent {
	q.mu.RLock()
	defer q.mu.RUnlock()

	result := make(map[string][]*workerv1.SessionEvent, len(q.sessions))
	for id, sq := range q.sessions {
		sq.mu.RLock()
		if len(sq.events) > 0 {
			copied := make([]*workerv1.SessionEvent, len(sq.events))
			copy(copied, sq.events)
			result[id] = copied
		}
		sq.mu.RUnlock()
	}
	return result
}
