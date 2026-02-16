package v2

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	acp "github.com/coder/acp-go-sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionMCPServers_InjectsFlowgenticServer(t *testing.T) {
	servers := sessionMCPServers(LaunchOpts{
		SystemPrompt: "foo\n## Flowgentic MCP\nbar",
		EnvVars: map[string]string{
			"AGENTCTL_WORKER_URL":    "http://127.0.0.1:9999",
			"AGENTCTL_WORKER_SECRET": "secret",
			"AGENTCTL_SESSION_ID":  "run-1",
			"AGENTCTL_AGENT":         "codex",
		},
	})

	require.Len(t, servers, 1)
	require.NotNil(t, servers[0].Stdio)
	assert.Equal(t, "flowgentic", servers[0].Stdio.Name)
	assert.NotEmpty(t, servers[0].Stdio.Command)
	assert.Empty(t, servers[0].Stdio.Args)
	assert.Equal(t, []acp.EnvVariable{
		{Name: "AGENTCTL_WORKER_URL", Value: "http://127.0.0.1:9999"},
		{Name: "AGENTCTL_WORKER_SECRET", Value: "secret"},
		{Name: "AGENTCTL_SESSION_ID", Value: "run-1"},
		{Name: "AGENTCTL_AGENT", Value: "codex"},
	}, servers[0].Stdio.Env)
}

func TestSessionMCPServers_UsesAgentctlBinOverride(t *testing.T) {
	tmp := t.TempDir()
	agentctlPath := filepath.Join(tmp, "agentctl")
	require.NoError(t, os.WriteFile(agentctlPath, []byte("#!/bin/sh\nexit 0\n"), 0o755))

	servers := sessionMCPServers(LaunchOpts{
		SystemPrompt: "## Flowgentic MCP",
		EnvVars: map[string]string{
			"AGENTCTL_WORKER_URL":   "http://127.0.0.1:9999",
			"AGENTCTL_SESSION_ID": "run-1",
			"AGENTCTL_BIN":          agentctlPath,
		},
	})

	require.Len(t, servers, 1)
	require.NotNil(t, servers[0].Stdio)
	assert.Equal(t, agentctlPath, servers[0].Stdio.Command)
}

func TestSessionMCPServers_NoInjectionWithoutAgentCtlEnv(t *testing.T) {
	servers := sessionMCPServers(LaunchOpts{
		SystemPrompt: "## Flowgentic MCP",
		EnvVars: map[string]string{
			"AGENTCTL_WORKER_SECRET": "secret",
		},
	})

	assert.Empty(t, servers)
}

func TestSessionMCPServers_DoesNotDuplicateFlowgenticServer(t *testing.T) {
	servers := sessionMCPServers(LaunchOpts{
		SystemPrompt: "## Flowgentic MCP",
		MCPServers: []acp.McpServer{
			{
				Stdio: &acp.McpServerStdio{
					Name:    "flowgentic",
					Command: "agentctl",
					Args:    nil,
				},
			},
		},
		EnvVars: map[string]string{
			"AGENTCTL_WORKER_URL":   "http://127.0.0.1:9999",
			"AGENTCTL_SESSION_ID": "run-1",
		},
	})

	require.Len(t, servers, 1)
	require.NotNil(t, servers[0].Stdio)
	assert.Equal(t, "flowgentic", servers[0].Stdio.Name)
}

func TestSessionMCPServers_NoDefaultInjectionWithoutMarker(t *testing.T) {
	servers := sessionMCPServers(LaunchOpts{
		SystemPrompt: "normal chat session",
		EnvVars: map[string]string{
			"AGENTCTL_WORKER_URL":   "http://127.0.0.1:9999",
			"AGENTCTL_SESSION_ID": "run-1",
		},
	})

	assert.Empty(t, servers)
}

func TestSessionMCPServers_InjectsWithEnvOverride(t *testing.T) {
	servers := sessionMCPServers(LaunchOpts{
		SystemPrompt: "normal chat session",
		EnvVars: map[string]string{
			"AGENTCTL_WORKER_URL":           "http://127.0.0.1:9999",
			"AGENTCTL_SESSION_ID":         "run-1",
			"FLOWGENTIC_ENABLE_DEFAULT_MCP": "1",
		},
	})

	require.Len(t, servers, 1)
	require.NotNil(t, servers[0].Stdio)
	assert.Equal(t, "flowgentic", servers[0].Stdio.Name)
}

func TestSessionMCPServers_PreservesExplicitEmptySlice(t *testing.T) {
	servers := sessionMCPServers(LaunchOpts{
		MCPServers: []acp.McpServer{},
	})

	require.NotNil(t, servers)
	assert.Empty(t, servers)
}

func TestDefaultFlowgenticMCPServer_ArgsSerializeAsEmptyArray(t *testing.T) {
	// Regression test: Args must serialize as JSON [] (not null).
	// OpenCode's ACP Zod schema uses z.array(z.string()) which rejects null.
	server, ok := defaultFlowgenticMCPServer(map[string]string{
		"AGENTCTL_WORKER_URL":   "http://127.0.0.1:9999",
		"AGENTCTL_SESSION_ID": "run-1",
	})
	require.True(t, ok)
	require.NotNil(t, server.Stdio)

	b, err := json.Marshal(server)
	require.NoError(t, err)

	var raw map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(b, &raw))

	assert.JSONEq(t, `[]`, string(raw["args"]),
		"args must serialize as [] not null; ACP agents using Zod validation reject null arrays")
}

func TestResolveAgentctlInvocation_ArgsNeverNil(t *testing.T) {
	// Ensure resolveAgentctlInvocation always returns a non-nil slice so that
	// JSON serialization produces [] instead of null.
	_, args := resolveAgentctlInvocation(map[string]string{})
	require.NotNil(t, args, "args slice must be non-nil to serialize as [] in JSON")
}
