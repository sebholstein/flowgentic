package main

import (
	"net/http"
	"os"

	"connectrpc.com/connect"
	workerv1connect "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1/workerv1connect"
	"github.com/sebastianm/flowgentic/internal/worker/interceptors"
)

func newAgentCtlClient() workerv1connect.AgentCtlServiceClient {
	workerURL := os.Getenv("AGENTCTL_WORKER_URL")
	var opts []connect.ClientOption
	if secret := os.Getenv("AGENTCTL_WORKER_SECRET"); secret != "" {
		opts = append(opts, connect.WithInterceptors(interceptors.NewAuth(secret)))
	}
	return workerv1connect.NewAgentCtlServiceClient(http.DefaultClient, workerURL, opts...)
}
