package agentctl

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
)

// agentCtlServiceHandler implements workerv1connect.AgentCtlServiceHandler.
type agentCtlServiceHandler struct {
	log     *slog.Logger
	handler EventHandler
}

func (h *agentCtlServiceHandler) ReportStatus(
	ctx context.Context,
	req *connect.Request[workerv1.ReportStatusRequest],
) (*connect.Response[workerv1.ReportStatusResponse], error) {
	agentType, err := driver.AgentTypeFromProto(req.Msg.Agent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	h.log.Debug("status report received",
		"session_id", req.Msg.SessionId,
		"agent", agentType,
		"status", req.Msg.Status,
	)

	if err := h.handler.HandleStatusReport(ctx, req.Msg.SessionId, string(agentType), req.Msg.Status); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(&workerv1.ReportStatusResponse{}), nil
}

func (h *agentCtlServiceHandler) SubmitPlan(
	ctx context.Context,
	req *connect.Request[workerv1.SubmitPlanRequest],
) (*connect.Response[workerv1.SubmitPlanResponse], error) {
	agentType, err := driver.AgentTypeFromProto(req.Msg.Agent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	h.log.Debug("plan submission received",
		"session_id", req.Msg.SessionId,
		"agent", agentType,
	)

	if err := h.handler.HandlePlanSubmission(ctx, req.Msg.SessionId, string(agentType), req.Msg.Plan); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(&workerv1.SubmitPlanResponse{}), nil
}
