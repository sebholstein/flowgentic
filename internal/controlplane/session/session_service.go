package session

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID          string
	ThreadID    string
	WorkerID    string
	Prompt      string
	Status      string
	Agent       string
	Model       string
	Mode        string
	SessionMode string
	SessionID   string
	TaskID      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SessionEvent is the domain type for a raw persisted event.
type SessionEvent struct {
	SessionID string
	Sequence  int64
	EventType string // e.g. "agent_message_chunk", "tool_call_start"
	Payload   []byte // proto-serialized workerv1.SessionEvent
	CreatedAt time.Time
}

type Store interface {
	CreateSession(ctx context.Context, s Session) error
	GetSession(ctx context.Context, id string) (Session, error)
	ListSessionsByThread(ctx context.Context, threadID string) ([]Session, error)
	ListPendingSessions(ctx context.Context, limit int64) ([]Session, error)
	UpdateSessionStatus(ctx context.Context, id, status, sessionID string) error
	GetCwdForSession(ctx context.Context, sessionID string) (string, error)
	InsertSessionEvent(ctx context.Context, evt SessionEvent) error
	ListSessionEventsBySession(ctx context.Context, sessionID string) ([]SessionEvent, error)
	ListSessionEventsByThread(ctx context.Context, threadID string) ([]SessionEvent, error)
	ListSessionEventsByTask(ctx context.Context, taskID sql.NullString) ([]SessionEvent, error)
}

type SessionService struct {
	store      Store
	reconciler *Reconciler
	registry   WorkerRegistry

	mu               sync.Mutex
	eventSubscribers map[chan SessionEventUpdate]struct{}
}

func NewSessionService(store Store, reconciler *Reconciler, registry WorkerRegistry) *SessionService {
	return &SessionService{
		store:            store,
		reconciler:       reconciler,
		registry:         registry,
		eventSubscribers: make(map[chan SessionEventUpdate]struct{}),
	}
}

func (s *SessionService) CreateSessionForThread(ctx context.Context, threadID, workerID, prompt, agent, model, mode, sessionMode string) (string, error) {
	id := uuid.Must(uuid.NewV7()).String()
	now := time.Now().UTC()

	sess := Session{
		ID:          id,
		ThreadID:    threadID,
		WorkerID:    workerID,
		Prompt:      prompt,
		Status:      "pending",
		Agent:       agent,
		Model:       model,
		Mode:        mode,
		SessionMode: sessionMode,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.CreateSession(ctx, sess); err != nil {
		return "", fmt.Errorf("creating session: %w", err)
	}

	s.reconciler.Notify()
	return id, nil
}

func (s *SessionService) GetSession(ctx context.Context, id string) (Session, error) {
	return s.store.GetSession(ctx, id)
}

func (s *SessionService) LookupWorker(workerID string) (url string, secret string, ok bool) {
	return s.registry.Lookup(workerID)
}

func (s *SessionService) ListSessions(ctx context.Context, threadID string) ([]Session, error) {
	return s.store.ListSessionsByThread(ctx, threadID)
}

// FindActiveSessionForThread returns the most recent running session for a thread.
func (s *SessionService) FindActiveSessionForThread(ctx context.Context, threadID string) (Session, error) {
	sessions, err := s.store.ListSessionsByThread(ctx, threadID)
	if err != nil {
		return Session{}, fmt.Errorf("listing sessions: %w", err)
	}

	// Iterate from end to find the most recent running session.
	for i := len(sessions) - 1; i >= 0; i-- {
		if sessions[i].Status == "running" {
			return sessions[i], nil
		}
	}

	return Session{}, fmt.Errorf("no active session found for thread %s", threadID)
}

// --- Pub-sub for live session events ---

// SubscribeEvents returns a channel that receives session events.
func (s *SessionService) SubscribeEvents() chan SessionEventUpdate {
	ch := make(chan SessionEventUpdate, 64)
	s.mu.Lock()
	s.eventSubscribers[ch] = struct{}{}
	s.mu.Unlock()
	return ch
}

// UnsubscribeEvents removes an event subscriber channel.
func (s *SessionService) UnsubscribeEvents(ch chan SessionEventUpdate) {
	s.mu.Lock()
	delete(s.eventSubscribers, ch)
	s.mu.Unlock()
}

// BroadcastEvent implements EventBroadcaster for the stateSyncHandler.
func (s *SessionService) BroadcastEvent(evt SessionEventUpdate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.eventSubscribers {
		select {
		case ch <- evt:
		default: // drop if slow
		}
	}
}

// --- Event History ---

func (s *SessionService) LoadEventHistory(ctx context.Context, sessionID, threadID, taskID string) ([]SessionEvent, error) {
	switch {
	case sessionID != "":
		return s.store.ListSessionEventsBySession(ctx, sessionID)
	case threadID != "":
		return s.store.ListSessionEventsByThread(ctx, threadID)
	case taskID != "":
		return s.store.ListSessionEventsByTask(ctx, sql.NullString{String: taskID, Valid: true})
	default:
		return nil, fmt.Errorf("one of session_id, thread_id, or task_id must be set")
	}
}

// ListSessionIDsForThread returns the IDs of all sessions belonging to a thread.
func (s *SessionService) ListSessionIDsForThread(ctx context.Context, threadID string) ([]string, error) {
	sessions, err := s.store.ListSessionsByThread(ctx, threadID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(sessions))
	for i, sess := range sessions {
		ids[i] = sess.ID
	}
	return ids, nil
}
