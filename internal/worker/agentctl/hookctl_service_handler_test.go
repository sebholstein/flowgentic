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

func TestHookCtlServiceHandler_ReportHook(t *testing.T) {
	d := newFakeDriver("claude-code")
	m := workload.NewAgentRunManager(testLogger(), "", "", d)

	agentRunID := "ar-hook"
	_, err := m.Launch(context.Background(), agentRunID, "claude-code", driver.LaunchOpts{
		Mode: driver.SessionModeHeadless,
	}, nil)
	require.NoError(t, err)

	h := &hookCtlServiceHandler{log: testLogger(), handler: m}

	t.Run("successful hook report", func(t *testing.T) {
		req := connect.NewRequest(&workerv1.ReportHookRequest{
			SessionId: agentRunID,
			Agent:     workerv1.Agent_AGENT_CLAUDE_CODE,
			HookName:  "Stop",
			Payload:   []byte(`{"reason":"user_request"}`),
		})
		resp, err := h.ReportHook(context.Background(), req)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		d.mu.Lock()
		defer d.mu.Unlock()
		require.Len(t, d.hookEvents, 1)
		assert.Equal(t, "Stop", d.hookEvents[0].HookName)
	})

	t.Run("unknown session returns error", func(t *testing.T) {
		req := connect.NewRequest(&workerv1.ReportHookRequest{
			SessionId: "nonexistent",
			Agent:     workerv1.Agent_AGENT_CLAUDE_CODE,
			HookName:  "Stop",
		})
		_, err := h.ReportHook(context.Background(), req)
		assert.Error(t, err)
	})
}
