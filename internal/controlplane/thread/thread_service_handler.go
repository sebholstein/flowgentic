package thread

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	controlplanev1 "github.com/sebastianm/flowgentic/internal/proto/gen/controlplane/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AgentRunCreator creates agent runs as a side effect of thread creation.
type AgentRunCreator interface {
	CreateAgentRunForThread(ctx context.Context, threadID, workerID, prompt, agent, model, mode string, yolo bool) (string, error)
}

// threadServiceHandler implements controlplanev1connect.ThreadServiceHandler.
type threadServiceHandler struct {
	log             *slog.Logger
	svc             *ThreadService
	agentRunCreator AgentRunCreator
}

func (h *threadServiceHandler) ListThreads(
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

func (h *threadServiceHandler) GetThread(
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

func (h *threadServiceHandler) CreateThread(
	ctx context.Context,
	req *connect.Request[controlplanev1.CreateThreadRequest],
) (*connect.Response[controlplanev1.CreateThreadResponse], error) {
	t := Thread{
		ProjectID: req.Msg.ProjectId,
		Agent:     req.Msg.Agent,
		Model:     req.Msg.Model,
		Mode:      req.Msg.Mode,
	}

	created, err := h.svc.CreateThread(ctx, t)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// If a prompt was provided, create an agent run for this thread.
	if req.Msg.Prompt != "" && h.agentRunCreator != nil {
		if _, err := h.agentRunCreator.CreateAgentRunForThread(ctx, created.ID, req.Msg.WorkerId, req.Msg.Prompt, req.Msg.Agent, req.Msg.Model, req.Msg.Mode, req.Msg.Yolo); err != nil {
			h.log.Error("failed to create agent run for thread", "thread_id", created.ID, "error", err)
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	return connect.NewResponse(&controlplanev1.CreateThreadResponse{
		Thread: threadToProto(created),
	}), nil
}

func (h *threadServiceHandler) UpdateThread(
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

func (h *threadServiceHandler) DeleteThread(
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
		Mode:      t.Mode,
		CreatedAt: timestamppb.New(t.CreatedAt),
		UpdatedAt: timestamppb.New(t.UpdatedAt),
	}
}
