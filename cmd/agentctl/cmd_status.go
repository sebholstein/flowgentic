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

func statusCmd() *cobra.Command {
	var status string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Report your current status (e.g. idle, running, errored)",
		RunE: func(_ *cobra.Command, _ []string) error {
			if status == "" {
				return fmt.Errorf("--status is required")
			}
			return reportStatus(status)
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "Status to report (e.g. idle, running, errored)")
	return cmd
}

func reportStatus(status string) error {
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

	client := newAgentCtlClient()
	_, err = client.ReportStatus(context.Background(), connect.NewRequest(&workerv1.ReportStatusRequest{
		SessionId: workloadID,
		Agent:     protoAgent,
		Status:    status,
	}))
	return err
}
