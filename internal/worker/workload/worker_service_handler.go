package workload

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
)

// workerServiceHandler implements workerv1connect.WorkerServiceHandler.
type workerServiceHandler struct {
	log *slog.Logger
	svc *WorkloadService
}

func (h *workerServiceHandler) ScheduleWorkload(
	ctx context.Context,
	req *connect.Request[workerv1.ScheduleWorkloadRequest],
) (*connect.Response[workerv1.ScheduleWorkloadResponse], error) {
	msg := req.Msg
	h.log.Info("ScheduleWorkload called",
		"workload_id", msg.WorkloadId,
		"agent", msg.Agent,
		"mode", msg.Mode,
	)

	agentType, err := driver.AgentTypeFromProto(msg.Agent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	mode := driver.SessionModeHeadless
	if msg.Mode != "" && driver.SessionMode(msg.Mode) != driver.SessionModeHeadless {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unsupported mode %q: only headless mode is supported", msg.Mode))
	}

	opts := driver.LaunchOpts{
		Mode:         mode,
		Prompt:       msg.Prompt,
		SystemPrompt: msg.SystemPrompt,
		Model:        msg.Model,
		Cwd:          msg.Cwd,
		SessionID:    msg.SessionId,
		Yolo:         msg.Yolo,
		AllowedTools: msg.AllowedTools,
	}

	result, err := h.svc.Schedule(ctx, msg.WorkloadId, string(agentType), opts)
	if err != nil {
		h.log.Error("ScheduleWorkload internal error", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	h.log.Info("ScheduleWorkload result",
		"accepted", result.Accepted,
		"workload_id", result.WorkloadID,
		"agent_id", result.AgentID,
		"agent_session_id", result.AgentSessionID,
		"status", result.Status,
		"mode", result.Mode,
		"message", result.Message,
	)

	return connect.NewResponse(&workerv1.ScheduleWorkloadResponse{
		Accepted:       result.Accepted,
		Message:        result.Message,
		SessionId:      result.WorkloadID,
		Agent:          driver.AgentType(result.AgentID).ProtoAgent(),
		Status:         result.Status,
		Mode:           result.Mode,
		AgentSessionId: result.AgentSessionID,
	}), nil
}
