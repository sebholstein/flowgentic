package worker

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	controlplanev1 "github.com/sebastianm/flowgentic/internal/proto/gen/controlplane/v1"
)

// workerManagementServiceHandler implements controlplanev1connect.WorkerManagementServiceHandler.
type workerManagementServiceHandler struct {
	log     *slog.Logger
	svc     *WorkerService
	pingSvc *PingService
}

func (h *workerManagementServiceHandler) ListWorkers(
	ctx context.Context,
	_ *connect.Request[controlplanev1.ListWorkersRequest],
) (*connect.Response[controlplanev1.ListWorkersResponse], error) {
	workers, err := h.svc.ListWorkers(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	pbWorkers := make([]*controlplanev1.WorkerConfig, len(workers))
	for i, w := range workers {
		pbWorkers[i] = workerToProto(w)
	}

	return connect.NewResponse(&controlplanev1.ListWorkersResponse{
		Workers: pbWorkers,
	}), nil
}

func (h *workerManagementServiceHandler) CreateWorker(
	ctx context.Context,
	req *connect.Request[controlplanev1.CreateWorkerRequest],
) (*connect.Response[controlplanev1.CreateWorkerResponse], error) {
	created, err := h.svc.CreateWorker(ctx, req.Msg.Id, req.Msg.Name, req.Msg.Url, req.Msg.Secret)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&controlplanev1.CreateWorkerResponse{
		Worker: workerToProto(created),
	}), nil
}

func (h *workerManagementServiceHandler) UpdateWorker(
	ctx context.Context,
	req *connect.Request[controlplanev1.UpdateWorkerRequest],
) (*connect.Response[controlplanev1.UpdateWorkerResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	updated, err := h.svc.UpdateWorker(ctx, req.Msg.Id, req.Msg.Name, req.Msg.Url, req.Msg.Secret)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&controlplanev1.UpdateWorkerResponse{
		Worker: workerToProto(updated),
	}), nil
}

func (h *workerManagementServiceHandler) DeleteWorker(
	ctx context.Context,
	req *connect.Request[controlplanev1.DeleteWorkerRequest],
) (*connect.Response[controlplanev1.DeleteWorkerResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	if err := h.svc.DeleteWorker(ctx, req.Msg.Id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&controlplanev1.DeleteWorkerResponse{}), nil
}

func (h *workerManagementServiceHandler) PingWorker(
	ctx context.Context,
	req *connect.Request[controlplanev1.PingWorkerRequest],
) (*connect.Response[controlplanev1.PingWorkerResponse], error) {
	workerID := req.Msg.WorkerId
	duration, err := h.pingSvc.PingWorker(ctx, workerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&controlplanev1.PingWorkerResponse{
		Duration: duration,
	}), nil
}

func workerToProto(w Worker) *controlplanev1.WorkerConfig {
	return &controlplanev1.WorkerConfig{
		Id:        w.ID,
		Name:      w.Name,
		Url:       w.URL,
		Secret:    w.Secret,
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
	}
}
