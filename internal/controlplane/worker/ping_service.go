package worker

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
	"github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1/workerv1connect"
)

// PingService measures round-trip time to workers.
type PingService struct {
	registry WorkerRegistry
}

// NewPingService creates a PingService backed by the given worker registry.
func NewPingService(registry WorkerRegistry) *PingService {
	return &PingService{registry: registry}
}

// PingWorker pings the specified worker and returns the round-trip duration.
func (s *PingService) PingWorker(ctx context.Context, workerID string) (string, error) {
	workerURL, secret, ok := s.registry.Lookup(workerID)
	if !ok {
		return "", fmt.Errorf("unknown worker %q", workerID)
	}

	client := workerv1connect.NewSystemServiceClient(
		http.DefaultClient,
		workerURL,
		connect.WithInterceptors(secretInterceptor(secret)),
	)

	start := time.Now()
	_, err := client.Ping(ctx, connect.NewRequest(&workerv1.PingRequest{}))
	rtt := time.Since(start)

	if err != nil {
		return "", fmt.Errorf("ping worker %q: %w", workerID, err)
	}

	return fmt.Sprintf("%.2fms", float64(rtt.Microseconds())/1000.0), nil
}

// secretInterceptor injects the Authorization header into outgoing requests.
func secretInterceptor(secret string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+secret)
			return next(ctx, req)
		}
	}
}
