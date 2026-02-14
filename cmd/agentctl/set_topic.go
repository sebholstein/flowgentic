package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
)

func runSetTopic(ctx context.Context, topic string) error {
	agentRunID := os.Getenv(agentCtlSessionIDEnv)
	if agentRunID == "" {
		return fmt.Errorf("%s env not set", agentCtlSessionIDEnv)
	}
	if strings.TrimSpace(topic) == "" {
		return fmt.Errorf("topic is required")
	}

	client := newAgentCtlClient()
	_, err := client.SetTopic(ctx, &connect.Request[workerv1.SetTopicRequest]{
		Msg: &workerv1.SetTopicRequest{
			AgentRunId: agentRunID,
			Topic:      topic,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to set topic (please try again): %w", err)
	}
	return nil
}
