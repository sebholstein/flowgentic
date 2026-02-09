package systeminfo

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
)

// systemServiceHandler implements workerv1connect.SystemServiceHandler.
type systemServiceHandler struct {
	log *slog.Logger
	svc *SystemInfoService
}

func (h *systemServiceHandler) Ping(
	_ context.Context,
	_ *connect.Request[workerv1.PingRequest],
) (*connect.Response[workerv1.PingResponse], error) {
	h.svc.Ping()
	return connect.NewResponse(&workerv1.PingResponse{}), nil
}

func (h *systemServiceHandler) ListAgents(
	ctx context.Context,
	req *connect.Request[workerv1.ListAgentsRequest],
) (*connect.Response[workerv1.ListAgentsResponse], error) {


	agents, err := h.svc.ListAgents(ctx, req.Msg.DisableCache)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := make([]*workerv1.AgentInfo, len(agents))
	for i, a := range agents {
		out[i] = &workerv1.AgentInfo{
			Id:      a.ID,
			Name:    a.Name,
			Version: a.Version,
			Enabled: true,
		}
	}

	return connect.NewResponse(&workerv1.ListAgentsResponse{Agents: out}), nil
}
