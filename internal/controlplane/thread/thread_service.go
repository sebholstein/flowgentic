package thread

import (
	"context"
	"fmt"

	"github.com/sebastianm/flowgentic/internal/controlplane/project"
)

// Thread is the domain representation of a thread.
type Thread struct {
	ID        string
	ProjectID string
	Agent     string
	Model     string
	CreatedAt string
	UpdatedAt string
}

// Store persists thread configurations.
type Store interface {
	ListThreads(ctx context.Context, projectID string) ([]Thread, error)
	GetThread(ctx context.Context, id string) (Thread, error)
	CreateThread(ctx context.Context, t Thread) (Thread, error)
	UpdateThread(ctx context.Context, t Thread) (Thread, error)
	DeleteThread(ctx context.Context, id string) error
}

// ThreadService implements the business logic for thread CRUD.
type ThreadService struct {
	store Store
}

// NewThreadService creates a ThreadService.
func NewThreadService(store Store) *ThreadService {
	return &ThreadService{store: store}
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

// CreateThread validates and persists a new thread.
func (s *ThreadService) CreateThread(ctx context.Context, t Thread) (Thread, error) {
	if err := project.ValidateResourceName(t.ID); err != nil {
		return Thread{}, fmt.Errorf("invalid thread id: %w", err)
	}
	if t.ProjectID == "" {
		return Thread{}, fmt.Errorf("project_id is required")
	}
	if t.Agent == "" {
		return Thread{}, fmt.Errorf("agent is required")
	}

	created, err := s.store.CreateThread(ctx, t)
	if err != nil {
		return Thread{}, fmt.Errorf("creating thread: %w", err)
	}
	return created, nil
}

// UpdateThread validates and updates an existing thread.
func (s *ThreadService) UpdateThread(ctx context.Context, t Thread) (Thread, error) {
	if t.ID == "" {
		return Thread{}, fmt.Errorf("thread id is required")
	}
	if t.Agent == "" {
		return Thread{}, fmt.Errorf("agent is required")
	}

	updated, err := s.store.UpdateThread(ctx, t)
	if err != nil {
		return Thread{}, fmt.Errorf("updating thread: %w", err)
	}
	return updated, nil
}

// DeleteThread removes a thread from the store.
func (s *ThreadService) DeleteThread(ctx context.Context, id string) error {
	if err := s.store.DeleteThread(ctx, id); err != nil {
		return fmt.Errorf("deleting thread: %w", err)
	}
	return nil
}
