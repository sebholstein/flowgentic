package main

import (
	"context"
	"fmt"
	"os"

	"connectrpc.com/connect"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
	"github.com/spf13/cobra"
)

func submitPlanCmd() *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:   "submit-plan",
		Short: "Submit a plan for human review and approval",
		RunE: func(_ *cobra.Command, _ []string) error {
			if file == "" {
				return fmt.Errorf("--file is required")
			}
			return submitPlan(file)
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "Path to the plan file (markdown)")
	return cmd
}

func submitPlan(file string) error {
	workerURL := os.Getenv("AGENTCTL_WORKER_URL")
	if workerURL == "" {
		return fmt.Errorf("AGENTCTL_WORKER_URL not set")
	}

	workloadID := os.Getenv("FLOWGENTIC_WORKLOAD_ID")
	agentName := os.Getenv("AGENTCTL_AGENT")

	protoAgent, err := driver.ParseProtoAgent(agentName)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read plan file: %w", err)
	}

	client := newAgentCtlClient()
	_, err = client.SubmitPlan(context.Background(), connect.NewRequest(&workerv1.SubmitPlanRequest{
		SessionId: workloadID,
		Agent:     protoAgent,
		Plan:      content,
	}))
	return err
}
