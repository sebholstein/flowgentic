package main

import (
	"fmt"
	"os"

	"connectrpc.com/connect"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
	"github.com/spf13/cobra"
)

func setTopicCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-topic <topic>",
		Short: "Sets the topic for the thread - max 100 chars for the topic",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			client := newAgentCtlClient()
			agentRunID := os.Getenv("AGENTCTL_AGENT_RUN_ID")
			if agentRunID == "" {
				return fmt.Errorf("AGENTCTL_AGENT_RUN_ID env not set")
			}
			_, err := client.SetTopic(c.Context(), &connect.Request[workerv1.SetTopicRequest]{
				Msg: &workerv1.SetTopicRequest{
					AgentRunId: agentRunID,
					Topic:      args[0],
				},
			})
			if err != nil {
				return fmt.Errorf("failed to set topic (please try again): %w", err)
			}
			fmt.Println("Topic set successfully")
			return nil
		},
	}
	return cmd
}
