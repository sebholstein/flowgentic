package v2

import (
	"context"
	"testing"
	"time"

	acp "github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestPermission_AutoApprovesInArchitectMode(t *testing.T) {
	events := make([]acp.SessionNotification, 0, 2)
	client := newFlowgenticClient(func(n acp.SessionNotification) {
		events = append(events, n)
	}, nil, "architect")

	resp, err := client.RequestPermission(context.Background(), acp.RequestPermissionRequest{
		SessionId: "sess-1",
		ToolCall: acp.RequestPermissionToolCall{
			ToolCallId: "call-1",
		},
		Options: []acp.PermissionOption{
			{OptionId: "reject", Kind: acp.PermissionOptionKindRejectOnce},
			{OptionId: "allow", Kind: acp.PermissionOptionKindAllowOnce},
		},
	})
	require.NoError(t, err)

	outcome := resp.Outcome.Selected
	require.NotNil(t, outcome)
	assert.Equal(t, acp.PermissionOptionId("allow"), outcome.OptionId)

	require.Len(t, events, 2)
	require.NotNil(t, events[0].Update.ToolCall)
	require.NotNil(t, events[1].Update.ToolCallUpdate)
	status := events[1].Update.ToolCallUpdate.Status
	require.NotNil(t, status)
	assert.Equal(t, acp.ToolCallStatusCompleted, *status)
}

func TestRequestPermission_BlocksInAskModeUntilResolved(t *testing.T) {
	client := newFlowgenticClient(nil, nil, "ask")

	done := make(chan struct{})
	var (
		resp acp.RequestPermissionResponse
		err  error
	)
	go func() {
		resp, err = client.RequestPermission(context.Background(), acp.RequestPermissionRequest{
			SessionId: "sess-1",
			ToolCall: acp.RequestPermissionToolCall{
				ToolCallId: "call-ask",
			},
			Options: []acp.PermissionOption{
				{OptionId: "allow", Kind: acp.PermissionOptionKindAllowOnce},
			},
		})
		close(done)
	}()

	select {
	case <-done:
		t.Fatalf("permission request should block in ask mode")
	case <-time.After(50 * time.Millisecond):
	}

	require.NoError(t, client.resolvePermission("call-ask", true))

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatalf("permission request did not complete after resolve")
	}

	require.NoError(t, err)
	outcome := resp.Outcome.Selected
	require.NotNil(t, outcome)
	assert.Equal(t, acp.PermissionOptionId("allow"), outcome.OptionId)
}
