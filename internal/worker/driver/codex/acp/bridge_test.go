package acp

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
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

func TestParseAvailableCommands_FromInitializeShapes(t *testing.T) {
	raw := map[string]any{
		"availableCommands": []any{
			map[string]any{"name": "init", "description": "create AGENTS.md"},
		},
		"capabilities": map[string]any{
			"commands": []any{
				map[string]any{"name": "review", "description": "review local changes"},
			},
		},
		"skills": []any{
			map[string]any{"name": "vercel-react-best-practices", "description": "React guidance"},
		},
	}
	b, err := json.Marshal(raw)
	require.NoError(t, err)

	cmds := parseAvailableCommands(b)
	require.Len(t, cmds, 3)
	assert.Equal(t, "init", cmds[0].Name)
	assert.Equal(t, "review", cmds[1].Name)
	assert.Equal(t, "vercel-react-best-practices", cmds[2].Name)
}

func TestParseAvailableCommands_ReturnsEmptyWhenUnknownShape(t *testing.T) {
	raw := map[string]any{
		"foo": map[string]any{
			"bar": "baz",
		},
	}
	b, err := json.Marshal(raw)
	require.NoError(t, err)

	assert.Empty(t, parseAvailableCommands(b))
}

func TestParseAvailableCommands_DeduplicatesByNameDeterministically(t *testing.T) {
	raw := map[string]any{
		"availableCommands": []any{
			map[string]any{"name": "init", "description": "first"},
			map[string]any{"name": "init", "description": "second"},
			map[string]any{"name": "review", "description": "third"},
		},
	}
	b, err := json.Marshal(raw)
	require.NoError(t, err)

	cmds := parseAvailableCommands(b)
	require.Len(t, cmds, 2)
	assert.Equal(t, "init", cmds[0].Name)
	assert.Equal(t, "first", cmds[0].Description)
	assert.Equal(t, "review", cmds[1].Name)
}

func TestBridgeReadStderrLoop_LogsDebugLines(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	b := newBridge(logger, nil)
	b.readStderrLoop(strings.NewReader("first line\n\nsecond line\n"))

	output := logBuf.String()
	assert.Contains(t, output, "codex stderr")
	assert.Contains(t, output, "first line")
	assert.Contains(t, output, "second line")
}

func TestParseModelState_ExtractsDisplayNameAndDescription(t *testing.T) {
	raw := map[string]any{
		"models": map[string]any{
			"available": []any{
				map[string]any{"id": "gpt-5", "displayName": "GPT-5", "description": "flagship model"},
				map[string]any{"id": "gpt-5-mini", "name": "GPT-5 Mini"},
			},
			"default": "gpt-5",
		},
	}
	b, err := json.Marshal(raw)
	require.NoError(t, err)

	state := parseModelState(b)
	require.NotNil(t, state)
	require.Len(t, state.AvailableModels, 2)

	assert.Equal(t, "GPT-5", state.AvailableModels[0].Name)
	require.NotNil(t, state.AvailableModels[0].Description)
	assert.Equal(t, "flagship model", *state.AvailableModels[0].Description)

	assert.Equal(t, "GPT-5 Mini", state.AvailableModels[1].Name)
	assert.Nil(t, state.AvailableModels[1].Description)
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
				Args:    nil,
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
	assert.Equal(t, []string{}, flow["args"])
	assert.Equal(t, map[string]string{"AGENTCTL_AGENT_RUN_ID": "sess-1"}, flow["env"])

	httpSrv, ok := servers["remote-http"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "http", httpSrv["type"])
	assert.Equal(t, "https://example.com/mcp", httpSrv["url"])
	assert.Equal(t, map[string]string{"Authorization": "Bearer token"}, httpSrv["headers"])
}
