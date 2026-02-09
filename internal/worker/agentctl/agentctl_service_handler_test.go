package agentctl

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
	"github.com/sebastianm/flowgentic/internal/worker/workload"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentCtlServiceHandler_ReportStatus(t *testing.T) {
	d := newFakeDriver("claude-code")
	m := workload.NewWorkloadManager(testLogger(), "", "", d)

	workloadID := "wl-status"
	_, err := m.Launch(context.Background(), workloadID, "claude-code", driver.LaunchOpts{
		Mode: driver.SessionModeHeadless,
	}, nil)
	require.NoError(t, err)

	h := &agentCtlServiceHandler{log: testLogger(), handler: m}

	t.Run("successful status report", func(t *testing.T) {
		req := connect.NewRequest(&workerv1.ReportStatusRequest{
			SessionId: workloadID,
			Agent:     workerv1.Agent_AGENT_CLAUDE_CODE,
			Status:    "running",
		})
		resp, err := h.ReportStatus(context.Background(), req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("unknown session returns error", func(t *testing.T) {
		req := connect.NewRequest(&workerv1.ReportStatusRequest{
			SessionId: "nonexistent",
			Agent:     workerv1.Agent_AGENT_CLAUDE_CODE,
			Status:    "idle",
		})
		_, err := h.ReportStatus(context.Background(), req)
		assert.Error(t, err)
	})
}

func TestAgentCtlServiceHandler_SubmitPlan(t *testing.T) {
	d := newFakeDriver("claude-code")
	m := workload.NewWorkloadManager(testLogger(), "", "", d)

	workloadID := "wl-plan"
	_, err := m.Launch(context.Background(), workloadID, "claude-code", driver.LaunchOpts{
		Mode: driver.SessionModeHeadless,
	}, nil)
	require.NoError(t, err)

	h := &agentCtlServiceHandler{log: testLogger(), handler: m}

	t.Run("successful plan submission", func(t *testing.T) {
		req := connect.NewRequest(&workerv1.SubmitPlanRequest{
			SessionId: workloadID,
			Agent:     workerv1.Agent_AGENT_CLAUDE_CODE,
			Plan:      []byte("# My Plan\n\n1. Do stuff\n2. Do more stuff"),
		})
		resp, err := h.SubmitPlan(context.Background(), req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("unknown session returns error", func(t *testing.T) {
		req := connect.NewRequest(&workerv1.SubmitPlanRequest{
			SessionId: "nonexistent",
			Agent:     workerv1.Agent_AGENT_CLAUDE_CODE,
			Plan:      []byte("# Plan"),
		})
		_, err := h.SubmitPlan(context.Background(), req)
		assert.Error(t, err)
	})
}
