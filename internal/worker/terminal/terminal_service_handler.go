package terminal

import (
	"context"
	"errors"
	"io"
	"log/slog"

	"connectrpc.com/connect"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
)

// terminalServiceHandler implements workerv1connect.TerminalServiceHandler.
type terminalServiceHandler struct {
	log *slog.Logger
	svc *TerminalService
}

func (h *terminalServiceHandler) CreateSession(
	_ context.Context,
	req *connect.Request[workerv1.CreateSessionRequest],
) (*connect.Response[workerv1.CreateSessionResponse], error) {
	id, err := h.svc.Create(
		req.Msg.Cwd,
		req.Msg.Cols,
		req.Msg.Rows,
		req.Msg.Shell,
		req.Msg.Env,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&workerv1.CreateSessionResponse{TerminalId: id}), nil
}

func (h *terminalServiceHandler) DestroySession(
	_ context.Context,
	req *connect.Request[workerv1.DestroySessionRequest],
) (*connect.Response[workerv1.DestroySessionResponse], error) {
	if err := h.svc.Destroy(req.Msg.TerminalId); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&workerv1.DestroySessionResponse{}), nil
}

func (h *terminalServiceHandler) Resize(
	_ context.Context,
	req *connect.Request[workerv1.ResizeRequest],
) (*connect.Response[workerv1.ResizeResponse], error) {
	if err := h.svc.Resize(req.Msg.TerminalId, req.Msg.Cols, req.Msg.Rows); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&workerv1.ResizeResponse{}), nil
}

func (h *terminalServiceHandler) Stream(
	ctx context.Context,
	stream *connect.BidiStream[workerv1.StreamRequest, workerv1.StreamResponse],
) error {
	// Receive the first message to get the terminal_id.
	first, err := stream.Receive()
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("expected initial message with terminal_id"))
	}

	terminalID := first.TerminalId
	reader, done, err := h.svc.Reader(terminalID)
	if err != nil {
		return connect.NewError(connect.CodeNotFound, err)
	}

	// Write initial data if present.
	if len(first.Data) > 0 {
		if _, err := h.svc.Write(terminalID, first.Data); err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Read goroutine: PTY → client.
	readErr := make(chan error, 1)
	go func() {
		defer cancel()
		buf := make([]byte, 4096)
		for {
			n, rerr := reader.Read(buf)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				if serr := stream.Send(&workerv1.StreamResponse{
					Event: &workerv1.StreamResponse_Data{Data: data},
				}); serr != nil {
					readErr <- serr
					return
				}
			}
			if rerr != nil {
				if errors.Is(rerr, io.EOF) {
					// Wait for the process to finish so we can get exit code.
					<-done
					sess, gerr := h.svc.get(terminalID)
					exitCode := int32(-1)
					if gerr == nil {
						exitCode = int32(sess.exitCode)
					}
					_ = stream.Send(&workerv1.StreamResponse{
						Event: &workerv1.StreamResponse_ExitCode{ExitCode: exitCode},
					})
				}
				readErr <- rerr
				return
			}
		}
	}()

	// Main goroutine: client → PTY.
	for {
		select {
		case <-ctx.Done():
			return nil
		case rerr := <-readErr:
			if errors.Is(rerr, io.EOF) {
				return nil
			}
			return connect.NewError(connect.CodeInternal, rerr)
		default:
		}

		msg, err := stream.Receive()
		if err != nil {
			if errors.Is(err, io.EOF) {
				// Client closed the stream — destroy the session.
				_ = h.svc.Destroy(terminalID)
				return nil
			}
			return err
		}

		if len(msg.Data) > 0 {
			if _, err := h.svc.Write(terminalID, msg.Data); err != nil {
				return connect.NewError(connect.CodeInternal, err)
			}
		}
	}
}
