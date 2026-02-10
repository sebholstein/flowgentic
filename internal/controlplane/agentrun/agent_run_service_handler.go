package agentrun

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	controlplanev1 "github.com/sebastianm/flowgentic/internal/proto/gen/controlplane/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// agentRunServiceHandler implements controlplanev1connect.AgentRunServiceHandler.
type agentRunServiceHandler struct {
	log *slog.Logger
	svc *AgentRunService
}

func (h *agentRunServiceHandler) GetAgentRun(
	ctx context.Context,
	req *connect.Request[controlplanev1.GetAgentRunRequest],
) (*connect.Response[controlplanev1.GetAgentRunResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	r, err := h.svc.GetAgentRun(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&controlplanev1.GetAgentRunResponse{
		AgentRun: agentRunToProto(r),
	}), nil
}

func (h *agentRunServiceHandler) ListAgentRuns(
	ctx context.Context,
	req *connect.Request[controlplanev1.ListAgentRunsRequest],
) (*connect.Response[controlplanev1.ListAgentRunsResponse], error) {
	if req.Msg.ThreadId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	runs, err := h.svc.ListAgentRuns(ctx, req.Msg.ThreadId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	pbRuns := make([]*controlplanev1.AgentRunConfig, len(runs))
	for i, r := range runs {
		pbRuns[i] = agentRunToProto(r)
	}

	return connect.NewResponse(&controlplanev1.ListAgentRunsResponse{
		AgentRuns: pbRuns,
	}), nil
}

func agentRunToProto(r AgentRun) *controlplanev1.AgentRunConfig {
	return &controlplanev1.AgentRunConfig{
		Id:        r.ID,
		ThreadId:  r.ThreadID,
		WorkerId:  r.WorkerID,
		Prompt:    r.Prompt,
		Status:    r.Status,
		Agent:     r.Agent,
		Model:     r.Model,
		Mode:      r.Mode,
		Yolo:      r.Yolo,
		SessionId: r.SessionID,
		CreatedAt: timestamppb.New(r.CreatedAt),
		UpdatedAt: timestamppb.New(r.UpdatedAt),
	}
}
