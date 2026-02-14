package task

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	controlplanev1 "github.com/sebastianm/flowgentic/internal/proto/gen/controlplane/v1"
)

type taskServiceHandler struct {
	log *slog.Logger
	svc *TaskService
}

func (h *taskServiceHandler) CreateTask(
	ctx context.Context,
	req *connect.Request[controlplanev1.CreateTaskRequest],
) (*connect.Response[controlplanev1.CreateTaskResponse], error) {
	t, err := h.svc.CreateTask(ctx, req.Msg.ThreadId, req.Msg.Description, req.Msg.Subtasks, req.Msg.SortIndex)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&controlplanev1.CreateTaskResponse{
		Task: taskToProto(t),
	}), nil
}

func (h *taskServiceHandler) GetTask(
	ctx context.Context,
	req *connect.Request[controlplanev1.GetTaskRequest],
) (*connect.Response[controlplanev1.GetTaskResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	t, err := h.svc.GetTask(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&controlplanev1.GetTaskResponse{
		Task: taskToProto(t),
	}), nil
}

func (h *taskServiceHandler) ListTasks(
	ctx context.Context,
	req *connect.Request[controlplanev1.ListTasksRequest],
) (*connect.Response[controlplanev1.ListTasksResponse], error) {
	if req.Msg.ThreadId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	tasks, err := h.svc.ListTasks(ctx, req.Msg.ThreadId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	pbTasks := make([]*controlplanev1.TaskConfig, len(tasks))
	for i, t := range tasks {
		pbTasks[i] = taskToProto(t)
	}
	return connect.NewResponse(&controlplanev1.ListTasksResponse{
		Tasks: pbTasks,
	}), nil
}

func (h *taskServiceHandler) UpdateTask(
	ctx context.Context,
	req *connect.Request[controlplanev1.UpdateTaskRequest],
) (*connect.Response[controlplanev1.UpdateTaskResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	t, err := h.svc.UpdateTask(ctx, req.Msg.Id, req.Msg.Description, req.Msg.Subtasks, req.Msg.Memory, req.Msg.Status, req.Msg.SortIndex)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&controlplanev1.UpdateTaskResponse{
		Task: taskToProto(t),
	}), nil
}

func (h *taskServiceHandler) DeleteTask(
	ctx context.Context,
	req *connect.Request[controlplanev1.DeleteTaskRequest],
) (*connect.Response[controlplanev1.DeleteTaskResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	if err := h.svc.DeleteTask(ctx, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&controlplanev1.DeleteTaskResponse{}), nil
}

func taskToProto(t Task) *controlplanev1.TaskConfig {
	return &controlplanev1.TaskConfig{
		Id:          t.ID,
		ThreadId:    t.ThreadID,
		Description: t.Description,
		Subtasks:    t.Subtasks,
		Memory:      t.Memory,
		Status:      t.Status,
		SortIndex:   t.SortIndex,
		CreatedAt:   t.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
		UpdatedAt:   t.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
	}
}
