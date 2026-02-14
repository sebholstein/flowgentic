package project

import (
	"context"
	"fmt"
	"regexp"
	"time"
)

// k8sNameRe matches a valid k8s-style resource name.
var k8sNameRe = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)

const maxNameLen = 63

// Project is the domain representation of a project.
type Project struct {
	ID                  string
	Name                string
	DefaultPlannerAgent string
	DefaultPlannerModel string
	EmbeddedWorkerPath  string
	WorkerPaths         map[string]string
	AgentPlanningTaskPreferences string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	SortIndex           int32
}

// SortEntry maps a project ID to a sort index.
type SortEntry struct {
	ID        string
	SortIndex int32
}

// Store persists project configurations.
type Store interface {
	ListProjects(ctx context.Context) ([]Project, error)
	GetProject(ctx context.Context, id string) (Project, error)
	CreateProject(ctx context.Context, p Project) (Project, error)
	UpdateProject(ctx context.Context, p Project) (Project, error)
	DeleteProject(ctx context.Context, id string) error
	ReorderProjects(ctx context.Context, entries []SortEntry) error
}

// ProjectService implements the business logic for project CRUD.
type ProjectService struct {
	store Store
}

// NewProjectService creates a ProjectService.
func NewProjectService(store Store) *ProjectService {
	return &ProjectService{store: store}
}

// ListProjects returns all projects from the store.
func (s *ProjectService) ListProjects(ctx context.Context) ([]Project, error) {
	return s.store.ListProjects(ctx)
}

// GetProject returns a single project by ID.
func (s *ProjectService) GetProject(ctx context.Context, id string) (Project, error) {
	return s.store.GetProject(ctx, id)
}

// CreateProject validates and persists a new project.
func (s *ProjectService) CreateProject(ctx context.Context, p Project) (Project, error) {
	if err := validateResourceName(p.ID); err != nil {
		return Project{}, fmt.Errorf("invalid project id: %w", err)
	}
	if p.Name == "" {
		return Project{}, fmt.Errorf("project name is required")
	}
	for workerName := range p.WorkerPaths {
		if err := validateResourceName(workerName); err != nil {
			return Project{}, fmt.Errorf("invalid worker name %q: %w", workerName, err)
		}
	}

	created, err := s.store.CreateProject(ctx, p)
	if err != nil {
		return Project{}, fmt.Errorf("creating project: %w", err)
	}
	return created, nil
}

// UpdateProject validates and updates an existing project.
func (s *ProjectService) UpdateProject(ctx context.Context, p Project) (Project, error) {
	if p.ID == "" {
		return Project{}, fmt.Errorf("project id is required")
	}
	if p.Name == "" {
		return Project{}, fmt.Errorf("project name is required")
	}
	for workerName := range p.WorkerPaths {
		if err := validateResourceName(workerName); err != nil {
			return Project{}, fmt.Errorf("invalid worker name %q: %w", workerName, err)
		}
	}

	updated, err := s.store.UpdateProject(ctx, p)
	if err != nil {
		return Project{}, fmt.Errorf("updating project: %w", err)
	}
	return updated, nil
}

// ReorderProjects updates sort indices for the given projects.
func (s *ProjectService) ReorderProjects(ctx context.Context, entries []SortEntry) error {
	if err := s.store.ReorderProjects(ctx, entries); err != nil {
		return fmt.Errorf("reordering projects: %w", err)
	}
	return nil
}

// DeleteProject removes a project from the store.
func (s *ProjectService) DeleteProject(ctx context.Context, id string) error {
	if err := s.store.DeleteProject(ctx, id); err != nil {
		return fmt.Errorf("deleting project: %w", err)
	}
	return nil
}

// ValidateResourceName checks that name follows k8s resource name rules.
func ValidateResourceName(name string) error {
	return validateResourceName(name)
}

func validateResourceName(name string) error {
	if name == "" {
		return fmt.Errorf("must not be empty")
	}
	if len(name) > maxNameLen {
		return fmt.Errorf("must be at most %d characters", maxNameLen)
	}
	if !k8sNameRe.MatchString(name) {
		return fmt.Errorf("must match %s (lowercase alphanumeric and hyphens, starting with a letter)", k8sNameRe.String())
	}
	return nil
}
