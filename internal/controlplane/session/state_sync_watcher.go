package session

import (
	"context"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/sebastianm/flowgentic/internal/connectutil"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
	"github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1/workerv1connect"
)

type StateSyncHandler interface {
	HandleSnapshot(workerID string, sessions []*workerv1.SessionState)
	HandleSessionUpdate(workerID string, session *workerv1.SessionState)
	HandleSessionRemoved(workerID string, removed *workerv1.SessionRemoved)
	HandleSessionEvent(workerID string, event *workerv1.SessionEvent)
	FlushAll()
}

type StateSyncWatcher struct {
	log      *slog.Logger
	workerID string
	url      string
	secret   string
	handler  StateSyncHandler
}

func NewStateSyncWatcher(log *slog.Logger, workerID, url, secret string, handler StateSyncHandler) *StateSyncWatcher {
	return &StateSyncWatcher{
		log:      log.With("worker_id", workerID),
		workerID: workerID,
		url:      url,
		secret:   secret,
		handler:  handler,
	}
}

func (w *StateSyncWatcher) Run(ctx context.Context) {
	w.log.Info("state sync watcher Run() starting", "url", w.url)
	backoff := time.Second
	const maxBackoff = 10 * time.Second

	for {
		err := w.watch(ctx)
		if ctx.Err() != nil {
			return
		}
		w.log.Warn("state sync stream ended, reconnecting", "error", err, "backoff", backoff)

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		backoff = min(backoff*2, maxBackoff)
	}
}

func (w *StateSyncWatcher) watch(ctx context.Context) error {
	client := workerv1connect.NewWorkerServiceClient(
		connectutil.H2CClient,
		w.url,
		connect.WithGRPC(),
		connect.WithInterceptors(&streamSecretInterceptor{secret: w.secret}),
	)

	w.log.Info("connecting state sync stream", "url", w.url)
	stream := client.StateSync(ctx)
	defer stream.CloseResponse()
	defer stream.CloseRequest()
	defer w.handler.FlushAll()

	if err := stream.Send(&workerv1.StateSyncRequest{}); err != nil {
		w.log.Error("state sync send initial request failed", "error", err)
		return err
	}

	w.log.Info("state sync stream connected, waiting for data")
	for {
		resp, err := stream.Receive()
		if err != nil {
			w.log.Error("state sync receive error", "error", err)
			return err
		}

		switch u := resp.Update.(type) {
		case *workerv1.StateSyncResponse_Snapshot:
			w.log.Info("received snapshot", "sessions", len(u.Snapshot.Sessions))
			w.handler.HandleSnapshot(w.workerID, u.Snapshot.Sessions)
		case *workerv1.StateSyncResponse_SessionUpdate:
			w.log.Info("received session update", "session_id", u.SessionUpdate.SessionId, "topic", u.SessionUpdate.Topic)
			w.handler.HandleSessionUpdate(w.workerID, u.SessionUpdate)
		case *workerv1.StateSyncResponse_SessionRemoved:
			w.log.Info("received session removed", "session_id", u.SessionRemoved.SessionId)
			w.handler.HandleSessionRemoved(w.workerID, u.SessionRemoved)
		case *workerv1.StateSyncResponse_SessionEvent:
			w.handler.HandleSessionEvent(w.workerID, u.SessionEvent)
			// Send ACK back to worker.
			if err := stream.Send(&workerv1.StateSyncRequest{
				AckSessionId: u.SessionEvent.GetSessionId(),
				AckSequence:  u.SessionEvent.GetSequence(),
			}); err != nil {
				w.log.Error("state sync ACK send failed", "error", err)
				return err
			}
		}
	}
}

type streamSecretInterceptor struct {
	secret string
}

func (i *streamSecretInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		req.Header().Set("Authorization", "Bearer "+i.secret)
		return next(ctx, req)
	}
}

func (i *streamSecretInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		conn.RequestHeader().Set("Authorization", "Bearer "+i.secret)
		return conn
	}
}

func (i *streamSecretInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}
