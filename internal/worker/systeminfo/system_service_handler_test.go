package systeminfo

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	workerv1 "github.com/sebastianm/flowgentic/internal/proto/gen/worker/v1"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
	v2 "github.com/sebastianm/flowgentic/internal/worker/driver/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemServiceHandler_GetAgentModels(t *testing.T) {
	t.Run("returns models for valid request", func(t *testing.T) {
		d := &fakeModelDriver{
			agent: string(driver.AgentTypeCodex),
			inv: v2.ModelInventory{
				Models:       []string{"gpt-5"},
				DefaultModel: "gpt-5",
			},
		}
		svc := NewSystemInfoService(fakeAgentInfo{}, []v2.Driver{d}, "/tmp")
		h := &systemServiceHandler{svc: svc}

		resp, err := h.GetAgentModels(context.Background(), connect.NewRequest(&workerv1.GetAgentModelsRequest{
			Agent: workerv1.Agent_AGENT_CODEX,
		}))
		require.NoError(t, err)
		assert.Equal(t, workerv1.Agent_AGENT_CODEX, resp.Msg.Agent)
		assert.Equal(t, []string{"gpt-5"}, resp.Msg.Models)
		assert.Equal(t, "gpt-5", resp.Msg.DefaultModel)
	})

	t.Run("rejects invalid agent", func(t *testing.T) {
		svc := NewSystemInfoService(fakeAgentInfo{}, nil, "/tmp")
		h := &systemServiceHandler{svc: svc}

		_, err := h.GetAgentModels(context.Background(), connect.NewRequest(&workerv1.GetAgentModelsRequest{
			Agent: workerv1.Agent_AGENT_UNSPECIFIED,
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})

	t.Run("maps discovery errors to unavailable", func(t *testing.T) {
		d := &fakeModelDriver{
			agent: string(driver.AgentTypeCodex),
			err:   errors.New("probe failed"),
		}
		svc := NewSystemInfoService(fakeAgentInfo{}, []v2.Driver{d}, "/tmp")
		h := &systemServiceHandler{svc: svc}

		_, err := h.GetAgentModels(context.Background(), connect.NewRequest(&workerv1.GetAgentModelsRequest{
			Agent: workerv1.Agent_AGENT_CODEX,
		}))
		require.Error(t, err)
		assert.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
	})
}
