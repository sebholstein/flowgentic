package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/sebastianm/flowgentic/internal/controlplane/project"
)

const timeFormat = "2006-01-02T15:04:05.000Z"

// SQLiteStore implements project.Store using sqlc-generated queries.
type SQLiteStore struct {
	db *sql.DB
	q  *Queries
}

// NewSQLiteStore creates a SQLiteStore.
func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db, q: New(db)}
}

func (s *SQLiteStore) ListProjects(ctx context.Context) ([]project.Project, error) {
	rows, err := s.q.ListProjects(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}

	projects := make([]project.Project, 0, len(rows))
	for _, r := range rows {
		p := projectFromRow(r)
		wp, err := s.loadWorkerPaths(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		p.WorkerPaths = wp
		projects = append(projects, p)
	}
	return projects, nil
}

func (s *SQLiteStore) GetProject(ctx context.Context, id string) (project.Project, error) {
	row, err := s.q.GetProject(ctx, id)
	if err != nil {
		return project.Project{}, fmt.Errorf("getting project %q: %w", id, err)
	}
	p := projectFromRow(row)
	wp, err := s.loadWorkerPaths(ctx, p.ID)
	if err != nil {
		return project.Project{}, err
	}
	p.WorkerPaths = wp
	return p, nil
}

func (s *SQLiteStore) CreateProject(ctx context.Context, p project.Project) (project.Project, error) {
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now

	nowStr := now.Format(timeFormat)
	err := s.q.CreateProject(ctx, CreateProjectParams{
		ID:                  p.ID,
		Name:                p.Name,
		DefaultPlannerAgent: p.DefaultPlannerAgent,
		DefaultPlannerModel: p.DefaultPlannerModel,
		EmbeddedWorkerPath:  p.EmbeddedWorkerPath,
		WorkerPaths:         "{}",
		CreatedAt:           nowStr,
		UpdatedAt:           nowStr,
		SortIndex:           int64(p.SortIndex),
		AgentPlanningTaskPreferences:    p.AgentPlanningTaskPreferences,
	})
	if err != nil {
		return project.Project{}, fmt.Errorf("inserting project: %w", err)
	}

	if err := s.syncWorkerPaths(ctx, p.ID, p.WorkerPaths); err != nil {
		return project.Project{}, err
	}
	return p, nil
}

func (s *SQLiteStore) UpdateProject(ctx context.Context, p project.Project) (project.Project, error) {
	now := time.Now().UTC()
	p.UpdatedAt = now

	res, err := s.q.UpdateProject(ctx, UpdateProjectParams{
		Name:                p.Name,
		DefaultPlannerAgent: p.DefaultPlannerAgent,
		DefaultPlannerModel: p.DefaultPlannerModel,
		EmbeddedWorkerPath:  p.EmbeddedWorkerPath,
		WorkerPaths:         "{}",
		UpdatedAt:           now.Format(timeFormat),
		SortIndex:           int64(p.SortIndex),
		ID:                  p.ID,
		AgentPlanningTaskPreferences:    p.AgentPlanningTaskPreferences,
	})
	if err != nil {
		return project.Project{}, fmt.Errorf("updating project: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return project.Project{}, fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return project.Project{}, fmt.Errorf("project %q not found", p.ID)
	}

	if err := s.syncWorkerPaths(ctx, p.ID, p.WorkerPaths); err != nil {
		return project.Project{}, err
	}

	return s.GetProject(ctx, p.ID)
}

func (s *SQLiteStore) DeleteProject(ctx context.Context, id string) error {
	res, err := s.q.DeleteProject(ctx, id)
	if err != nil {
		return fmt.Errorf("deleting project: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("project %q not found", id)
	}
	return nil
}

func (s *SQLiteStore) ReorderProjects(ctx context.Context, entries []project.SortEntry) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	stmt, err := tx.PrepareContext(ctx, "UPDATE projects SET sort_index = ? WHERE id = ?")
	if err != nil {
		return fmt.Errorf("preparing statement: %w", err)
	}
	defer stmt.Close()

	for _, e := range entries {
		if _, err := stmt.ExecContext(ctx, e.SortIndex, e.ID); err != nil {
			return fmt.Errorf("updating sort_index for %q: %w", e.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

func projectFromRow(r Project) project.Project {
	createdAt, _ := time.Parse(timeFormat, r.CreatedAt)
	updatedAt, _ := time.Parse(timeFormat, r.UpdatedAt)
	return project.Project{
		ID:                  r.ID,
		Name:                r.Name,
		DefaultPlannerAgent: r.DefaultPlannerAgent,
		DefaultPlannerModel: r.DefaultPlannerModel,
		EmbeddedWorkerPath:  r.EmbeddedWorkerPath,
		AgentPlanningTaskPreferences:    r.AgentPlanningTaskPreferences,
		CreatedAt:           createdAt,
		UpdatedAt:           updatedAt,
		SortIndex:           int32(r.SortIndex),
	}
}

func (s *SQLiteStore) loadWorkerPaths(ctx context.Context, projectID string) (map[string]string, error) {
	rows, err := s.q.ListWorkerProjectPaths(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("loading worker paths for project %q: %w", projectID, err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	wp := make(map[string]string, len(rows))
	for _, r := range rows {
		wp[r.WorkerID] = r.Path
	}
	return wp, nil
}

func (s *SQLiteStore) syncWorkerPaths(ctx context.Context, projectID string, paths map[string]string) error {
	if err := s.q.DeleteWorkerProjectPaths(ctx, projectID); err != nil {
		return fmt.Errorf("deleting worker paths for project %q: %w", projectID, err)
	}
	for workerID, path := range paths {
		if err := s.q.InsertWorkerProjectPath(ctx, InsertWorkerProjectPathParams{
			ProjectID: projectID,
			WorkerID:  workerID,
			Path:      path,
		}); err != nil {
			return fmt.Errorf("inserting worker path for project %q worker %q: %w", projectID, workerID, err)
		}
	}
	return nil
}
