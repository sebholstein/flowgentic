package v2

import (
	"context"
	"log/slog"
	"testing"

	acp "github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type modelAgent struct {
	state *acp.SessionModelState
}

func (a *modelAgent) Authenticate(context.Context, acp.AuthenticateRequest) (acp.AuthenticateResponse, error) {
	return acp.AuthenticateResponse{}, nil
}

func (a *modelAgent) Initialize(_ context.Context, _ acp.InitializeRequest) (acp.InitializeResponse, error) {
	return acp.InitializeResponse{
		ProtocolVersion: acp.ProtocolVersion(acp.ProtocolVersionNumber),
		AgentInfo: &acp.Implementation{
			Name:    "test-agent",
			Version: "1.0.0",
		},
	}, nil
}

func (a *modelAgent) Cancel(context.Context, acp.CancelNotification) error { return nil }

func (a *modelAgent) NewSession(_ context.Context, _ acp.NewSessionRequest) (acp.NewSessionResponse, error) {
	return acp.NewSessionResponse{
		SessionId: "session-1",
		Models:    a.state,
	}, nil
}

func (a *modelAgent) Prompt(context.Context, acp.PromptRequest) (acp.PromptResponse, error) {
	return acp.PromptResponse{}, nil
}

func (a *modelAgent) SetSessionMode(context.Context, acp.SetSessionModeRequest) (acp.SetSessionModeResponse, error) {
	return acp.SetSessionModeResponse{}, nil
}

func TestDiscoverModels(t *testing.T) {
	d := NewDriver(testLogger(), AgentConfig{
		AgentID: "test-agent",
		AdapterFactory: func(_ *slog.Logger) acp.Agent {
			return &modelAgent{
				state: &acp.SessionModelState{
					AvailableModels: []acp.ModelInfo{
						{ModelId: "model-a"},
						{ModelId: "model-b"},
					},
					CurrentModelId: "model-a",
				},
			}
		},
	})

	inventory, err := d.DiscoverModels(context.Background(), "/tmp")
	require.NoError(t, err)
	assert.Equal(t, []string{"model-a", "model-b"}, inventory.Models)
	assert.Equal(t, "model-a", inventory.DefaultModel)
}

func TestDiscoverModels_ErrorsWhenMetadataMissing(t *testing.T) {
	d := NewDriver(testLogger(), AgentConfig{
		AgentID: "test-agent",
		AdapterFactory: func(_ *slog.Logger) acp.Agent {
			return &modelAgent{}
		},
	})

	_, err := d.DiscoverModels(context.Background(), "/tmp")
	require.Error(t, err)
	assert.ErrorContains(t, err, "returned no model metadata")
}
