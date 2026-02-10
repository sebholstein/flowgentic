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

var statusToProto = map[driver.SessionStatus]workerv1.AgentRunStatus{
	driver.SessionStatusStarting: workerv1.AgentRunStatus_AGENT_RUN_STATUS_STARTING,
	driver.SessionStatusRunning:  workerv1.AgentRunStatus_AGENT_RUN_STATUS_RUNNING,
	driver.SessionStatusIdle:     workerv1.AgentRunStatus_AGENT_RUN_STATUS_IDLE,
	driver.SessionStatusStopping: workerv1.AgentRunStatus_AGENT_RUN_STATUS_STOPPING,
	driver.SessionStatusStopped:  workerv1.AgentRunStatus_AGENT_RUN_STATUS_STOPPED,
	driver.SessionStatusErrored:  workerv1.AgentRunStatus_AGENT_RUN_STATUS_ERRORED,
}

var modeToProto = map[driver.SessionMode]workerv1.AgentRunMode{
	driver.SessionModeHeadless: workerv1.AgentRunMode_AGENT_RUN_MODE_HEADLESS,
}

func (h *workerServiceHandler) ListAgentRuns(
	ctx context.Context,
	_ *connect.Request[workerv1.ListAgentRunsRequest],
) (*connect.Response[workerv1.ListAgentRunsResponse], error) {
	entries := h.svc.ListAgentRuns(ctx)
	runs := make([]*workerv1.AgentRunInfo, 0, len(entries))
	for _, e := range entries {
		runs = append(runs, &workerv1.AgentRunInfo{
			AgentRunId:     e.AgentRunID,
			Agent:          driver.AgentType(e.Info.AgentID).ProtoAgent(),
			Status:         statusToProto[e.Info.Status],
			Mode:           modeToProto[e.Info.Mode],
			SessionId:      e.AgentRunID,
			AgentSessionId: e.Info.AgentSessionID,
		})
	}
	return connect.NewResponse(&workerv1.ListAgentRunsResponse{
		AgentRuns: runs,
	}), nil
}

func (h *workerServiceHandler) NewAgentRun(
	ctx context.Context,
	req *connect.Request[workerv1.NewAgentRunRequest],
) (*connect.Response[workerv1.NewAgentRunResponse], error) {
	msg := req.Msg
	h.log.Info("NewAgentRun called",
		"agent_run_id", msg.AgentRunId,
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

	result, err := h.svc.Schedule(ctx, msg.AgentRunId, string(agentType), opts)
	if err != nil {
		h.log.Error("NewAgentRun internal error", "error", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	h.log.Info("NewAgentRun result",
		"accepted", result.Accepted,
		"agent_run_id", result.AgentRunID,
		"agent_id", result.AgentID,
		"agent_session_id", result.AgentSessionID,
		"status", result.Status,
		"mode", result.Mode,
		"message", result.Message,
	)

	return connect.NewResponse(&workerv1.NewAgentRunResponse{
		Accepted:       result.Accepted,
		Message:        result.Message,
		SessionId:      result.AgentRunID,
		Agent:          driver.AgentType(result.AgentID).ProtoAgent(),
		Status:         result.Status,
		Mode:           result.Mode,
		AgentSessionId: result.AgentSessionID,
	}), nil
}
