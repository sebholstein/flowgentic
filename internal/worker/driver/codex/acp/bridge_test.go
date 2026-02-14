package acp

import (
	"encoding/json"
	"testing"

	acpsdk "github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseModelState(t *testing.T) {
	raw := map[string]any{
		"models": map[string]any{
			"available": []any{
				map[string]any{"id": "gpt-5"},
				map[string]any{"id": "gpt-5-mini"},
			},
			"default": "gpt-5",
		},
	}
	b, err := json.Marshal(raw)
	require.NoError(t, err)

	state := parseModelState(b)
	require.NotNil(t, state)
	require.Len(t, state.AvailableModels, 2)
	assert.Equal(t, "gpt-5", string(state.CurrentModelId))
	assert.Equal(t, "gpt-5", string(state.AvailableModels[0].ModelId))
	assert.Equal(t, "gpt-5-mini", string(state.AvailableModels[1].ModelId))
}

func TestParseModelState_ReturnsNilWhenIncomplete(t *testing.T) {
	raw := map[string]any{
		"models": map[string]any{
			"available": []any{
				map[string]any{"id": "gpt-5"},
			},
		},
	}
	b, err := json.Marshal(raw)
	require.NoError(t, err)

	assert.Nil(t, parseModelState(b))
}

func TestExtractThreadID_AcceptsConversationID(t *testing.T) {
	payload := []byte(`{"conversationId":"conv-123"}`)
	assert.Equal(t, "conv-123", extractThreadID(payload))
}

func TestCodexMCPServers(t *testing.T) {
	servers := codexMCPServers([]acpsdk.McpServer{
		{
			Stdio: &acpsdk.McpServerStdio{
				Name:    "flowgentic",
				Command: "agentctl",
				Args:    []string{"mcp", "serve"},
				Env: []acpsdk.EnvVariable{
					{Name: "AGENTCTL_AGENT_RUN_ID", Value: "sess-1"},
				},
			},
		},
		{
			Http: &acpsdk.McpServerHttp{
				Name: "remote-http",
				Url:  "https://example.com/mcp",
				Headers: []acpsdk.HttpHeader{
					{Name: "Authorization", Value: "Bearer token"},
				},
			},
		},
	})

	require.Len(t, servers, 2)
	flow, ok := servers["flowgentic"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "stdio", flow["type"])
	assert.Equal(t, "agentctl", flow["command"])
	assert.Equal(t, []string{"mcp", "serve"}, flow["args"])
	assert.Equal(t, map[string]string{"AGENTCTL_AGENT_RUN_ID": "sess-1"}, flow["env"])

	httpSrv, ok := servers["remote-http"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "http", httpSrv["type"])
	assert.Equal(t, "https://example.com/mcp", httpSrv["url"])
	assert.Equal(t, map[string]string{"Authorization": "Bearer token"}, httpSrv["headers"])
}
