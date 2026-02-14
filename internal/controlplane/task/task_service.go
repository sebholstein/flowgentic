package task

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID          string
	ThreadID    string
	Description string
	Subtasks    []string
	Memory      string
	Status      string
	SortIndex   int32
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Store interface {
	CreateTask(ctx context.Context, t Task) error
	GetTask(ctx context.Context, id string) (Task, error)
	ListTasksByThread(ctx context.Context, threadID string) ([]Task, error)
	UpdateTask(ctx context.Context, t Task) error
	DeleteTask(ctx context.Context, id string) error
}

type TaskService struct {
	store Store
}

func NewTaskService(store Store) *TaskService {
	return &TaskService{store: store}
}

func (s *TaskService) CreateTask(ctx context.Context, threadID, description string, subtasks []string, sortIndex int32) (Task, error) {
	if threadID == "" {
		return Task{}, fmt.Errorf("thread_id is required")
	}

	now := time.Now().UTC()
	t := Task{
		ID:          uuid.Must(uuid.NewV7()).String(),
		ThreadID:    threadID,
		Description: description,
		Subtasks:    subtasks,
		Memory:      "",
		Status:      "pending",
		SortIndex:   sortIndex,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if t.Subtasks == nil {
		t.Subtasks = []string{}
	}

	if err := s.store.CreateTask(ctx, t); err != nil {
		return Task{}, fmt.Errorf("creating task: %w", err)
	}
	return t, nil
}

func (s *TaskService) GetTask(ctx context.Context, id string) (Task, error) {
	return s.store.GetTask(ctx, id)
}

func (s *TaskService) ListTasks(ctx context.Context, threadID string) ([]Task, error) {
	if threadID == "" {
		return nil, fmt.Errorf("thread_id is required")
	}
	return s.store.ListTasksByThread(ctx, threadID)
}

func (s *TaskService) UpdateTask(ctx context.Context, id string, description string, subtasks []string, memory string, status string, sortIndex int32) (Task, error) {
	if id == "" {
		return Task{}, fmt.Errorf("task id is required")
	}

	existing, err := s.store.GetTask(ctx, id)
	if err != nil {
		return Task{}, fmt.Errorf("getting task: %w", err)
	}

	existing.Description = description
	existing.Subtasks = subtasks
	existing.Memory = memory
	existing.Status = status
	existing.SortIndex = sortIndex
	existing.UpdatedAt = time.Now().UTC()

	if err := s.store.UpdateTask(ctx, existing); err != nil {
		return Task{}, fmt.Errorf("updating task: %w", err)
	}
	return existing, nil
}

func (s *TaskService) DeleteTask(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("task id is required")
	}
	return s.store.DeleteTask(ctx, id)
}

// MarshalSubtasks serializes subtasks to JSON.
func MarshalSubtasks(subtasks []string) string {
	b, _ := json.Marshal(subtasks)
	return string(b)
}

// UnmarshalSubtasks deserializes subtasks from JSON.
func UnmarshalSubtasks(s string) []string {
	var result []string
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return []string{}
	}
	return result
}
