package project

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
)

// projectServiceHandler implements workerv1connect.ProjectServiceHandler.
type projectServiceHandler struct {
	log *slog.Logger
	svc *ProjectService
}

func (h *projectServiceHandler) WatchFileTree(
	ctx context.Context,
	req *connect.Request[workerv1.WatchFileTreeRequest],
	stream *connect.ServerStream[workerv1.WatchFileTreeResponse],
) error {
	cwd := req.Msg.Cwd

	// 1. Send initial snapshot with full recursive tree.
	entries, err := h.svc.ListTree(cwd)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}

	pbEntries := make([]*workerv1.FileEntry, len(entries))
	for i, e := range entries {
		pbEntries[i] = fileEntryToProto(e)
	}

	if err := stream.Send(&workerv1.WatchFileTreeResponse{
		Update: &workerv1.WatchFileTreeResponse_Snapshot{
			Snapshot: &workerv1.FileTreeSnapshot{Entries: pbEntries},
		},
	}); err != nil {
		return err
	}

	// 2. Start recursive filesystem watcher and forward events.
	ch, err := h.svc.WatchTree(ctx, cwd)
	if err != nil {
		h.log.Warn("failed to start file watcher, closing stream", "error", err)
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case fe, ok := <-ch:
			if !ok {
				return nil
			}
			var evType workerv1.FileEventType
			switch fe.Type {
			case FileEventCreated:
				evType = workerv1.FileEventType_FILE_EVENT_TYPE_CREATED
			case FileEventRemoved:
				evType = workerv1.FileEventType_FILE_EVENT_TYPE_REMOVED
			}
			if err := stream.Send(&workerv1.WatchFileTreeResponse{
				Update: &workerv1.WatchFileTreeResponse_Event{
					Event: &workerv1.FileTreeEvent{
						Type:  evType,
						Entry: fileEntryToProto(fe.Entry),
					},
				},
			}); err != nil {
				return err
			}
		}
	}
}

func fileEntryToProto(e FileEntry) *workerv1.FileEntry {
	return &workerv1.FileEntry{
		Name:  e.Name,
		Path:  e.Path,
		IsDir: e.IsDir,
		Size:  e.Size,
	}
}
