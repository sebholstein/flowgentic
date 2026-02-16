package systeminfo

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
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

func (h *systemServiceHandler) GetAgentModels(
	ctx context.Context,
	req *connect.Request[workerv1.GetAgentModelsRequest],
) (*connect.Response[workerv1.GetAgentModelsResponse], error) {
	agent, err := driver.AgentTypeFromProto(req.Msg.Agent)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	models, err := h.svc.GetAgentModels(ctx, agent, req.Msg.DisableCache)
	if err != nil {
		switch {
		case errors.Is(err, ErrUnsupportedAgent):
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.Is(err, ErrModelDiscovery):
			return nil, connect.NewError(connect.CodeUnavailable, err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	protoModels := make([]*workerv1.ModelInfo, len(models.Models))
	for i, m := range models.Models {
		protoModels[i] = &workerv1.ModelInfo{
			Id:          m.ID,
			DisplayName: m.DisplayName,
			Description: m.Description,
		}
	}

	return connect.NewResponse(&workerv1.GetAgentModelsResponse{
		Agent:        req.Msg.Agent,
		Models:       protoModels,
		DefaultModel: models.DefaultModel,
	}), nil
}
