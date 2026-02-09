package thread

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	controlplanev1 "github.com/sebastianm/flowgentic/internal/proto/gen/controlplane/v1"
)

// threadManagementServiceHandler implements controlplanev1connect.ThreadManagementServiceHandler.
type threadManagementServiceHandler struct {
	log *slog.Logger
	svc *ThreadService
}

func (h *threadManagementServiceHandler) ListThreads(
	ctx context.Context,
	req *connect.Request[controlplanev1.ListThreadsRequest],
) (*connect.Response[controlplanev1.ListThreadsResponse], error) {
	threads, err := h.svc.ListThreads(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	pbThreads := make([]*controlplanev1.ThreadConfig, len(threads))
	for i, t := range threads {
		pbThreads[i] = threadToProto(t)
	}

	return connect.NewResponse(&controlplanev1.ListThreadsResponse{
		Threads: pbThreads,
	}), nil
}

func (h *threadManagementServiceHandler) GetThread(
	ctx context.Context,
	req *connect.Request[controlplanev1.GetThreadRequest],
) (*connect.Response[controlplanev1.GetThreadResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	t, err := h.svc.GetThread(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&controlplanev1.GetThreadResponse{
		Thread: threadToProto(t),
	}), nil
}

func (h *threadManagementServiceHandler) CreateThread(
	ctx context.Context,
	req *connect.Request[controlplanev1.CreateThreadRequest],
) (*connect.Response[controlplanev1.CreateThreadResponse], error) {
	t := Thread{
		ID:        req.Msg.Id,
		ProjectID: req.Msg.ProjectId,
		Agent:     req.Msg.Agent,
		Model:     req.Msg.Model,
	}

	created, err := h.svc.CreateThread(ctx, t)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&controlplanev1.CreateThreadResponse{
		Thread: threadToProto(created),
	}), nil
}

func (h *threadManagementServiceHandler) UpdateThread(
	ctx context.Context,
	req *connect.Request[controlplanev1.UpdateThreadRequest],
) (*connect.Response[controlplanev1.UpdateThreadResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	t := Thread{
		ID:    req.Msg.Id,
		Agent: req.Msg.Agent,
		Model: req.Msg.Model,
	}

	updated, err := h.svc.UpdateThread(ctx, t)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&controlplanev1.UpdateThreadResponse{
		Thread: threadToProto(updated),
	}), nil
}

func (h *threadManagementServiceHandler) DeleteThread(
	ctx context.Context,
	req *connect.Request[controlplanev1.DeleteThreadRequest],
) (*connect.Response[controlplanev1.DeleteThreadResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	if err := h.svc.DeleteThread(ctx, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&controlplanev1.DeleteThreadResponse{}), nil
}

func threadToProto(t Thread) *controlplanev1.ThreadConfig {
	return &controlplanev1.ThreadConfig{
		Id:        t.ID,
		ProjectId: t.ProjectID,
		Agent:     t.Agent,
		Model:     t.Model,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}
