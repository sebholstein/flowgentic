package task

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memStore struct {
	tasks map[string]Task
}

func newMemStore() *memStore {
	return &memStore{tasks: make(map[string]Task)}
}

func (m *memStore) CreateTask(_ context.Context, t Task) error {
	if _, exists := m.tasks[t.ID]; exists {
		return fmt.Errorf("task %q already exists", t.ID)
	}
	m.tasks[t.ID] = t
	return nil
}

func (m *memStore) GetTask(_ context.Context, id string) (Task, error) {
	t, ok := m.tasks[id]
	if !ok {
		return Task{}, fmt.Errorf("task %q not found", id)
	}
	return t, nil
}

func (m *memStore) ListTasksByThread(_ context.Context, threadID string) ([]Task, error) {
	var out []Task
	for _, t := range m.tasks {
		if t.ThreadID == threadID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (m *memStore) UpdateTask(_ context.Context, t Task) error {
	if _, exists := m.tasks[t.ID]; !exists {
		return fmt.Errorf("task %q not found", t.ID)
	}
	m.tasks[t.ID] = t
	return nil
}

func (m *memStore) DeleteTask(_ context.Context, id string) error {
	if _, exists := m.tasks[id]; !exists {
		return fmt.Errorf("task %q not found", id)
	}
	delete(m.tasks, id)
	return nil
}

func TestTaskService(t *testing.T) {
	t.Run("create and get", func(t *testing.T) {
		svc := NewTaskService(newMemStore())
		ctx := context.Background()

		created, err := svc.CreateTask(ctx, "thread-1", "implement auth", []string{"design", "code", "test"}, 0)
		require.NoError(t, err)
		assert.NotEmpty(t, created.ID)
		assert.Equal(t, "thread-1", created.ThreadID)
		assert.Equal(t, "implement auth", created.Description)
		assert.Equal(t, []string{"design", "code", "test"}, created.Subtasks)
		assert.Equal(t, "pending", created.Status)

		got, err := svc.GetTask(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
	})

	t.Run("create rejects empty thread_id", func(t *testing.T) {
		svc := NewTaskService(newMemStore())
		_, err := svc.CreateTask(context.Background(), "", "desc", nil, 0)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "thread_id is required")
	})

	t.Run("list by thread", func(t *testing.T) {
		svc := NewTaskService(newMemStore())
		ctx := context.Background()

		_, err := svc.CreateTask(ctx, "thread-1", "task 1", nil, 0)
		require.NoError(t, err)
		_, err = svc.CreateTask(ctx, "thread-1", "task 2", nil, 1)
		require.NoError(t, err)
		_, err = svc.CreateTask(ctx, "thread-2", "other", nil, 0)
		require.NoError(t, err)

		tasks, err := svc.ListTasks(ctx, "thread-1")
		require.NoError(t, err)
		assert.Len(t, tasks, 2)
	})

	t.Run("update", func(t *testing.T) {
		svc := NewTaskService(newMemStore())
		ctx := context.Background()

		created, err := svc.CreateTask(ctx, "thread-1", "original", nil, 0)
		require.NoError(t, err)

		updated, err := svc.UpdateTask(ctx, created.ID, "updated desc", []string{"a", "b"}, "some memory", "running", 1)
		require.NoError(t, err)
		assert.Equal(t, "updated desc", updated.Description)
		assert.Equal(t, []string{"a", "b"}, updated.Subtasks)
		assert.Equal(t, "some memory", updated.Memory)
		assert.Equal(t, "running", updated.Status)
		assert.Equal(t, int32(1), updated.SortIndex)
	})

	t.Run("delete", func(t *testing.T) {
		svc := NewTaskService(newMemStore())
		ctx := context.Background()

		created, err := svc.CreateTask(ctx, "thread-1", "to delete", nil, 0)
		require.NoError(t, err)

		err = svc.DeleteTask(ctx, created.ID)
		require.NoError(t, err)

		_, err = svc.GetTask(ctx, created.ID)
		require.Error(t, err)
	})
}

func TestSubtasksSerialization(t *testing.T) {
	t.Run("marshal and unmarshal", func(t *testing.T) {
		input := []string{"step 1", "step 2", "step 3"}
		json := MarshalSubtasks(input)
		output := UnmarshalSubtasks(json)
		assert.Equal(t, input, output)
	})

	t.Run("unmarshal empty", func(t *testing.T) {
		result := UnmarshalSubtasks("[]")
		assert.Empty(t, result)
	})

	t.Run("unmarshal invalid", func(t *testing.T) {
		result := UnmarshalSubtasks("not json")
		assert.Empty(t, result)
	})
}
