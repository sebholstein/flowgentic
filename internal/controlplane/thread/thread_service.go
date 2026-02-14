package thread

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EventType describes the kind of thread lifecycle event.
type EventType int

const (
	EventCreated EventType = iota + 1
	EventUpdated
	EventRemoved
)

// ThreadEvent is emitted when a thread is created, updated, or removed.
type ThreadEvent struct {
	Type   EventType
	Thread Thread
}

// Thread is the domain representation of a thread.
type Thread struct {
	ID        string
	ProjectID string
	Mode      string
	Topic     string
	Plan      string
	Archived  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Store persists thread configurations.
type Store interface {
	ListThreads(ctx context.Context, projectID string) ([]Thread, error)
	GetThread(ctx context.Context, id string) (Thread, error)
	CreateThread(ctx context.Context, t Thread) (Thread, error)
	DeleteThread(ctx context.Context, id string) error
	UpdateThreadTopic(ctx context.Context, id, topic string) error
	UpdateThreadPlan(ctx context.Context, id, plan string) error
	UpdateThreadArchived(ctx context.Context, id string, archived bool) error
}

// ThreadService implements the business logic for thread CRUD.
type ThreadService struct {
	store Store

	mu          sync.Mutex
	subscribers map[chan ThreadEvent]struct{}
}

// NewThreadService creates a ThreadService.
func NewThreadService(store Store) *ThreadService {
	return &ThreadService{
		store:       store,
		subscribers: make(map[chan ThreadEvent]struct{}),
	}
}

// broadcast sends a ThreadEvent to all subscribers.
func (s *ThreadService) broadcast(evt ThreadEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.subscribers {
		select {
		case ch <- evt:
		default: // drop if subscriber is slow
		}
	}
}

// UpdateTopic persists a new topic for a thread and notifies subscribers.
func (s *ThreadService) UpdateTopic(ctx context.Context, id, topic string) error {
	if err := s.store.UpdateThreadTopic(ctx, id, topic); err != nil {
		return fmt.Errorf("updating topic: %w", err)
	}

	t, err := s.store.GetThread(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching thread after topic update: %w", err)
	}
	s.broadcast(ThreadEvent{Type: EventUpdated, Thread: t})
	return nil
}

// UpdatePlan persists a new plan for a thread and notifies subscribers.
func (s *ThreadService) UpdatePlan(ctx context.Context, id, plan string) error {
	if err := s.store.UpdateThreadPlan(ctx, id, plan); err != nil {
		return fmt.Errorf("updating plan: %w", err)
	}

	t, err := s.store.GetThread(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching thread after plan update: %w", err)
	}
	s.broadcast(ThreadEvent{Type: EventUpdated, Thread: t})
	return nil
}

// Subscribe returns a channel that receives thread events.
func (s *ThreadService) Subscribe() chan ThreadEvent {
	ch := make(chan ThreadEvent, 16)
	s.mu.Lock()
	s.subscribers[ch] = struct{}{}
	s.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber channel.
func (s *ThreadService) Unsubscribe(ch chan ThreadEvent) {
	s.mu.Lock()
	delete(s.subscribers, ch)
	s.mu.Unlock()
}

// ListThreads returns all threads for a project.
func (s *ThreadService) ListThreads(ctx context.Context, projectID string) ([]Thread, error) {
	if projectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	return s.store.ListThreads(ctx, projectID)
}

// GetThread returns a single thread by ID.
func (s *ThreadService) GetThread(ctx context.Context, id string) (Thread, error) {
	return s.store.GetThread(ctx, id)
}

// CreateThread generates an ID, validates, and persists a new thread.
func (s *ThreadService) CreateThread(ctx context.Context, t Thread) (Thread, error) {
	t.ID = uuid.NewString()
	if t.ProjectID == "" {
		return Thread{}, fmt.Errorf("project_id is required")
	}

	created, err := s.store.CreateThread(ctx, t)
	if err != nil {
		return Thread{}, fmt.Errorf("creating thread: %w", err)
	}
	s.broadcast(ThreadEvent{Type: EventCreated, Thread: created})
	return created, nil
}

// ArchiveThread sets or clears the archived flag on a thread.
func (s *ThreadService) ArchiveThread(ctx context.Context, id string, archived bool) (Thread, error) {
	if id == "" {
		return Thread{}, fmt.Errorf("thread id is required")
	}
	if err := s.store.UpdateThreadArchived(ctx, id, archived); err != nil {
		return Thread{}, fmt.Errorf("updating archived: %w", err)
	}
	t, err := s.store.GetThread(ctx, id)
	if err != nil {
		return Thread{}, fmt.Errorf("fetching thread after archive: %w", err)
	}
	s.broadcast(ThreadEvent{Type: EventUpdated, Thread: t})
	return t, nil
}

// DeleteThread removes a thread from the store.
func (s *ThreadService) DeleteThread(ctx context.Context, id string) error {
	t, err := s.store.GetThread(ctx, id)
	if err != nil {
		return fmt.Errorf("fetching thread before delete: %w", err)
	}
	if err := s.store.DeleteThread(ctx, id); err != nil {
		return fmt.Errorf("deleting thread: %w", err)
	}
	s.broadcast(ThreadEvent{Type: EventRemoved, Thread: t})
	return nil
}
