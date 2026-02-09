package embeddedworker

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	controlplanev1 "github.com/sebastianm/flowgentic/internal/proto/gen/controlplane/v1"
)

// embeddedWorkerServiceHandler implements
// controlplanev1connect.EmbeddedWorkerServiceHandler.
type embeddedWorkerServiceHandler struct {
	log *slog.Logger
	svc *EmbeddedWorkerService
}

func (h *embeddedWorkerServiceHandler) StartEmbeddedWorker(
	ctx context.Context,
	_ *connect.Request[controlplanev1.StartEmbeddedWorkerRequest],
) (*connect.Response[controlplanev1.StartEmbeddedWorkerResponse], error) {
	err := h.svc.Start(ctx)
	status, lastErr, pid, addr := h.svc.GetStatus()
	resp := &controlplanev1.StartEmbeddedWorkerResponse{
		Status:     status,
		Error:      lastErr,
		WorkerId:   workerID,
		ListenAddr: addr,
		Pid:        int32(pid),
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

func (h *embeddedWorkerServiceHandler) StopEmbeddedWorker(
	ctx context.Context,
	_ *connect.Request[controlplanev1.StopEmbeddedWorkerRequest],
) (*connect.Response[controlplanev1.StopEmbeddedWorkerResponse], error) {
	err := h.svc.Stop(ctx)
	status, lastErr, _, _ := h.svc.GetStatus()
	resp := &controlplanev1.StopEmbeddedWorkerResponse{
		Status: status,
		Error:  lastErr,
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

func (h *embeddedWorkerServiceHandler) RestartEmbeddedWorker(
	ctx context.Context,
	_ *connect.Request[controlplanev1.RestartEmbeddedWorkerRequest],
) (*connect.Response[controlplanev1.RestartEmbeddedWorkerResponse], error) {
	err := h.svc.Restart(ctx)
	status, lastErr, pid, addr := h.svc.GetStatus()
	resp := &controlplanev1.RestartEmbeddedWorkerResponse{
		Status:     status,
		Error:      lastErr,
		WorkerId:   workerID,
		ListenAddr: addr,
		Pid:        int32(pid),
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

func (h *embeddedWorkerServiceHandler) WatchEmbeddedWorkerStatus(
	ctx context.Context,
	_ *connect.Request[controlplanev1.WatchEmbeddedWorkerStatusRequest],
	stream *connect.ServerStream[controlplanev1.WatchEmbeddedWorkerStatusResponse],
) error {
	// Send current status immediately.
	status, lastErr, pid, addr := h.svc.GetStatus()
	if err := stream.Send(&controlplanev1.WatchEmbeddedWorkerStatusResponse{
		Status:     status,
		Error:      lastErr,
		WorkerId:   workerID,
		ListenAddr: addr,
		Pid:        int32(pid),
	}); err != nil {
		return err
	}

	ch := h.svc.Subscribe()
	defer h.svc.Unsubscribe(ch)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ch:
			status, lastErr, pid, addr = h.svc.GetStatus()
			if err := stream.Send(&controlplanev1.WatchEmbeddedWorkerStatusResponse{
				Status:     status,
				Error:      lastErr,
				WorkerId:   workerID,
				ListenAddr: addr,
				Pid:        int32(pid),
			}); err != nil {
				return err
			}
		}
	}
}
