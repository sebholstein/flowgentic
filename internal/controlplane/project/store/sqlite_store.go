package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sebastianm/flowgentic/internal/controlplane/project"
)

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
		p, err := projectFromRow(r)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

func (s *SQLiteStore) GetProject(ctx context.Context, id string) (project.Project, error) {
	row, err := s.q.GetProject(ctx, id)
	if err != nil {
		return project.Project{}, fmt.Errorf("getting project %q: %w", id, err)
	}
	return projectFromRow(row)
}

func (s *SQLiteStore) CreateProject(ctx context.Context, p project.Project) (project.Project, error) {
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	p.CreatedAt = now
	p.UpdatedAt = now

	wpJSON, err := marshalWorkerPaths(p.WorkerPaths)
	if err != nil {
		return project.Project{}, err
	}

	err = s.q.CreateProject(ctx, CreateProjectParams{
		ID:                  p.ID,
		Name:                p.Name,
		DefaultPlannerAgent: p.DefaultPlannerAgent,
		DefaultPlannerModel: p.DefaultPlannerModel,
		EmbeddedWorkerPath:  p.EmbeddedWorkerPath,
		WorkerPaths:         wpJSON,
		CreatedAt:           p.CreatedAt,
		UpdatedAt:           p.UpdatedAt,
		SortIndex:           int64(p.SortIndex),
	})
	if err != nil {
		return project.Project{}, fmt.Errorf("inserting project: %w", err)
	}
	return p, nil
}

func (s *SQLiteStore) UpdateProject(ctx context.Context, p project.Project) (project.Project, error) {
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	p.UpdatedAt = now

	wpJSON, err := marshalWorkerPaths(p.WorkerPaths)
	if err != nil {
		return project.Project{}, err
	}

	res, err := s.q.UpdateProject(ctx, UpdateProjectParams{
		Name:                p.Name,
		DefaultPlannerAgent: p.DefaultPlannerAgent,
		DefaultPlannerModel: p.DefaultPlannerModel,
		EmbeddedWorkerPath:  p.EmbeddedWorkerPath,
		WorkerPaths:         wpJSON,
		UpdatedAt:           p.UpdatedAt,
		SortIndex:           int64(p.SortIndex),
		ID:                  p.ID,
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

func projectFromRow(r Project) (project.Project, error) {
	wp, err := unmarshalWorkerPaths(r.WorkerPaths)
	if err != nil {
		return project.Project{}, err
	}
	return project.Project{
		ID:                  r.ID,
		Name:                r.Name,
		DefaultPlannerAgent: r.DefaultPlannerAgent,
		DefaultPlannerModel: r.DefaultPlannerModel,
		EmbeddedWorkerPath:  r.EmbeddedWorkerPath,
		WorkerPaths:         wp,
		CreatedAt:           r.CreatedAt,
		UpdatedAt:           r.UpdatedAt,
		SortIndex:           int32(r.SortIndex),
	}, nil
}

func marshalWorkerPaths(wp map[string]string) (string, error) {
	if wp == nil {
		return "{}", nil
	}
	b, err := json.Marshal(wp)
	if err != nil {
		return "", fmt.Errorf("marshalling worker_paths: %w", err)
	}
	return string(b), nil
}

func unmarshalWorkerPaths(s string) (map[string]string, error) {
	if s == "" || s == "{}" {
		return nil, nil
	}
	var wp map[string]string
	if err := json.Unmarshal([]byte(s), &wp); err != nil {
		return nil, fmt.Errorf("unmarshalling worker_paths: %w", err)
	}
	return wp, nil
}
