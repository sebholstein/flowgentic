package thread

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	controlplanev1 "github.com/sebastianm/flowgentic/internal/proto/gen/controlplane/v1"
)

// threadServiceHandler implements controlplanev1connect.ThreadServiceHandler.
type threadServiceHandler struct {
	log *slog.Logger
	svc *ThreadService
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
		Mode:      req.Msg.Mode,
	}

	created, err := h.svc.CreateThread(ctx, t)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&controlplanev1.CreateThreadResponse{
		Thread: threadToProto(created),
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

func (h *threadServiceHandler) ArchiveThread(
	ctx context.Context,
	req *connect.Request[controlplanev1.ArchiveThreadRequest],
) (*connect.Response[controlplanev1.ArchiveThreadResponse], error) {
	if req.Msg.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}

	t, err := h.svc.ArchiveThread(ctx, req.Msg.Id, req.Msg.Archived)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&controlplanev1.ArchiveThreadResponse{
		Thread: threadToProto(t),
	}), nil
}

func (h *threadServiceHandler) WatchThreadUpdates(
	ctx context.Context,
	_ *connect.Request[controlplanev1.WatchThreadUpdatesRequest],
	stream *connect.ServerStream[controlplanev1.WatchThreadUpdatesResponse],
) error {
	ch := h.svc.Subscribe()
	defer h.svc.Unsubscribe(ch)

	for {
		select {
		case <-ctx.Done():
			return nil
		case evt := <-ch:
			if err := stream.Send(&controlplanev1.WatchThreadUpdatesResponse{
				EventType: eventTypeToProto(evt.Type),
				Thread:    threadToProto(evt.Thread),
			}); err != nil {
				return err
			}
		}
	}
}

func eventTypeToProto(t EventType) controlplanev1.ThreadEventType {
	switch t {
	case EventCreated:
		return controlplanev1.ThreadEventType_THREAD_EVENT_TYPE_CREATED
	case EventUpdated:
		return controlplanev1.ThreadEventType_THREAD_EVENT_TYPE_UPDATED
	case EventRemoved:
		return controlplanev1.ThreadEventType_THREAD_EVENT_TYPE_REMOVED
	default:
		return controlplanev1.ThreadEventType_THREAD_EVENT_TYPE_UNSPECIFIED
	}
}

func threadToProto(t Thread) *controlplanev1.ThreadConfig {
	return &controlplanev1.ThreadConfig{
		Id:        t.ID,
		ProjectId: t.ProjectID,
		Mode:      t.Mode,
		Topic:     t.Topic,
		Plan:      t.Plan,
		Archived:  t.Archived,
		CreatedAt: t.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
		UpdatedAt: t.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
	}
}
