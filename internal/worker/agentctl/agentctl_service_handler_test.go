package agentctl

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
	v2 "github.com/sebastianm/flowgentic/internal/worker/driver/v2"
	"github.com/sebastianm/flowgentic/internal/worker/workload"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentCtlServiceHandler_SetTopic(t *testing.T) {
	d := newFakeDriver("claude-code")
	m := workload.NewSessionManager(testLogger(), "", "", d)

	agentRunID := "ar-topic"
	_, err := m.Launch(context.Background(), agentRunID, "claude-code", v2.LaunchOpts{}, nil)
	require.NoError(t, err)

	h := &agentCtlServiceHandler{log: testLogger(), handler: m}

	t.Run("sets topic successfully", func(t *testing.T) {
		req := connect.NewRequest(&workerv1.SetTopicRequest{
			AgentRunId: agentRunID,
			Topic:      "working on tests",
		})
		resp, err := h.SetTopic(context.Background(), req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("unknown session returns error", func(t *testing.T) {
		req := connect.NewRequest(&workerv1.SetTopicRequest{
			AgentRunId: "nonexistent",
			Topic:      "topic",
		})
		_, err := h.SetTopic(context.Background(), req)
		assert.Error(t, err)
	})

	t.Run("rejects topic over 140 chars", func(t *testing.T) {
		long := make([]rune, 141)
		for i := range long {
			long[i] = 'a'
		}
		req := connect.NewRequest(&workerv1.SetTopicRequest{
			AgentRunId: agentRunID,
			Topic:      string(long),
		})
		_, err := h.SetTopic(context.Background(), req)
		assert.Error(t, err)
	})
}

func TestAgentCtlServiceHandler_ReportStatus(t *testing.T) {
	d := newFakeDriver("claude-code")
	m := workload.NewSessionManager(testLogger(), "", "", d)

	agentRunID := "ar-status"
	_, err := m.Launch(context.Background(), agentRunID, "claude-code", v2.LaunchOpts{}, nil)
	require.NoError(t, err)

	h := &agentCtlServiceHandler{log: testLogger(), handler: m}

	t.Run("successful status report", func(t *testing.T) {
		req := connect.NewRequest(&workerv1.ReportStatusRequest{
			SessionId: agentRunID,
			Agent:     workerv1.Agent_AGENT_CLAUDE_CODE,
			Status:    "running",
		})
		resp, err := h.ReportStatus(context.Background(), req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
}

func TestAgentCtlServiceHandler_SubmitPlan(t *testing.T) {
	d := newFakeDriver("claude-code")
	m := workload.NewSessionManager(testLogger(), "", "", d)

	agentRunID := "ar-plan"
	_, err := m.Launch(context.Background(), agentRunID, "claude-code", v2.LaunchOpts{}, nil)
	require.NoError(t, err)

	h := &agentCtlServiceHandler{log: testLogger(), handler: m}

	t.Run("successful plan submission", func(t *testing.T) {
		req := connect.NewRequest(&workerv1.SubmitPlanRequest{
			SessionId: agentRunID,
			Agent:     workerv1.Agent_AGENT_CLAUDE_CODE,
			Plan:      []byte("# My Plan\n\n1. Do stuff\n2. Do more stuff"),
		})
		resp, err := h.SubmitPlan(context.Background(), req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
}
