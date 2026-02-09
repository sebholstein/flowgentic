package worker

import (
	"context"
	"fmt"

	"github.com/sebastianm/flowgentic/internal/controlplane/project"
)

// Worker is the domain representation of a worker endpoint.
type Worker struct {
	ID        string
	Name      string
	URL       string
	Secret    string
	CreatedAt string
	UpdatedAt string
}

// Store persists worker configurations.
type Store interface {
	ListWorkers(ctx context.Context) ([]Worker, error)
	CreateWorker(ctx context.Context, w Worker) (Worker, error)
	UpdateWorker(ctx context.Context, w Worker) (Worker, error)
	DeleteWorker(ctx context.Context, id string) error
	GetWorker(ctx context.Context, id string) (Worker, error)
}

// RegistryUpdater keeps the relay registry in sync with the database.
type RegistryUpdater interface {
	AddWorker(id string, rawURL string, secret string) error
	RemoveWorker(id string)
}

// WorkerRegistry provides lookup of worker connection details.
type WorkerRegistry interface {
	Lookup(workerID string) (url string, secret string, ok bool)
}

// WorkerService implements the business logic for worker CRUD.
type WorkerService struct {
	store    Store
	registry RegistryUpdater
}

// NewWorkerService creates a WorkerService.
func NewWorkerService(store Store, registry RegistryUpdater) *WorkerService {
	return &WorkerService{store: store, registry: registry}
}

// ListWorkers returns all workers from the store.
func (s *WorkerService) ListWorkers(ctx context.Context) ([]Worker, error) {
	return s.store.ListWorkers(ctx)
}

// CreateWorker persists a new worker and registers it with the relay.
// The id must be a valid k8s-style resource name (provided by the caller).
func (s *WorkerService) CreateWorker(ctx context.Context, id, name, url, secret string) (Worker, error) {
	if err := project.ValidateResourceName(id); err != nil {
		return Worker{}, fmt.Errorf("invalid worker id: %w", err)
	}

	w := Worker{
		ID:     id,
		Name:   name,
		URL:    url,
		Secret: secret,
	}

	created, err := s.store.CreateWorker(ctx, w)
	if err != nil {
		return Worker{}, fmt.Errorf("creating worker: %w", err)
	}

	if err := s.registry.AddWorker(created.ID, created.URL, created.Secret); err != nil {
		return Worker{}, fmt.Errorf("registering worker in relay: %w", err)
	}

	return created, nil
}

// UpdateWorker updates an existing worker and refreshes the relay entry.
func (s *WorkerService) UpdateWorker(ctx context.Context, id, name, url, secret string) (Worker, error) {
	w := Worker{
		ID:     id,
		Name:   name,
		URL:    url,
		Secret: secret,
	}

	updated, err := s.store.UpdateWorker(ctx, w)
	if err != nil {
		return Worker{}, fmt.Errorf("updating worker: %w", err)
	}

	if err := s.registry.AddWorker(updated.ID, updated.URL, updated.Secret); err != nil {
		return Worker{}, fmt.Errorf("updating worker in relay: %w", err)
	}

	return updated, nil
}

// DeleteWorker removes a worker from the store and the relay.
func (s *WorkerService) DeleteWorker(ctx context.Context, id string) error {
	if err := s.store.DeleteWorker(ctx, id); err != nil {
		return fmt.Errorf("deleting worker: %w", err)
	}
	s.registry.RemoveWorker(id)
	return nil
}
