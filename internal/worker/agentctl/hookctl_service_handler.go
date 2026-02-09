package agentctl

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"

	"github.com/sebastianm/flowgentic/internal/worker/driver"
)

// hookCtlServiceHandler implements workerv1connect.HookCtlServiceHandler.
type hookCtlServiceHandler struct {
	log     *slog.Logger
	handler EventHandler
}

func (h *hookCtlServiceHandler) ReportHook(
	ctx context.Context,
	req *connect.Request[workerv1.ReportHookRequest],
) (*connect.Response[workerv1.ReportHookResponse], error) {
	agentType, err := driver.AgentTypeFromProto(req.Msg.Agent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	h.log.Debug("hook received",
		"session_id", req.Msg.SessionId,
		"agent", agentType,
		"hook_name", req.Msg.HookName,
	)

	event := driver.HookEvent{
		SessionID: req.Msg.SessionId,
		Agent:     string(agentType),
		HookName:  req.Msg.HookName,
		Payload:   req.Msg.Payload,
	}

	if err := h.handler.HandleHookEvent(ctx, event); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	return connect.NewResponse(&workerv1.ReportHookResponse{}), nil
}
