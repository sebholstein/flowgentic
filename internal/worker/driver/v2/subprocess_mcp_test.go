package v2

import (
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
			"AGENTCTL_AGENT_RUN_ID":  "run-1",
			"AGENTCTL_AGENT":         "codex",
		},
	})

	require.Len(t, servers, 1)
	require.NotNil(t, servers[0].Stdio)
	assert.Equal(t, "flowgentic", servers[0].Stdio.Name)
	assert.NotEmpty(t, servers[0].Stdio.Command)
	require.GreaterOrEqual(t, len(servers[0].Stdio.Args), 2)
	assert.Equal(t, []string{"mcp", "serve"}, servers[0].Stdio.Args[len(servers[0].Stdio.Args)-2:])
	assert.Equal(t, []acp.EnvVariable{
		{Name: "AGENTCTL_WORKER_URL", Value: "http://127.0.0.1:9999"},
		{Name: "AGENTCTL_WORKER_SECRET", Value: "secret"},
		{Name: "AGENTCTL_AGENT_RUN_ID", Value: "run-1"},
		{Name: "AGENTCTL_AGENT", Value: "codex"},
	}, servers[0].Stdio.Env)
}

func TestSessionMCPServers_UsesAgentctlBinOverride(t *testing.T) {
	tmp := t.TempDir()
	agentctlPath := filepath.Join(tmp, "agentctl")
	require.NoError(t, os.WriteFile(agentctlPath, []byte("#!/bin/sh\nif [ \"$1\" = \"mcp\" ] && [ \"$2\" = \"serve\" ] && [ \"$3\" = \"--help\" ]; then exit 0; fi\nexit 0\n"), 0o755))

	servers := sessionMCPServers(LaunchOpts{
		SystemPrompt: "## Flowgentic MCP",
		EnvVars: map[string]string{
			"AGENTCTL_WORKER_URL":   "http://127.0.0.1:9999",
			"AGENTCTL_AGENT_RUN_ID": "run-1",
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
					Args:    []string{"mcp", "serve"},
				},
			},
		},
		EnvVars: map[string]string{
			"AGENTCTL_WORKER_URL":   "http://127.0.0.1:9999",
			"AGENTCTL_AGENT_RUN_ID": "run-1",
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
			"AGENTCTL_AGENT_RUN_ID": "run-1",
		},
	})

	assert.Empty(t, servers)
}

func TestSessionMCPServers_InjectsWithEnvOverride(t *testing.T) {
	servers := sessionMCPServers(LaunchOpts{
		SystemPrompt: "normal chat session",
		EnvVars: map[string]string{
			"AGENTCTL_WORKER_URL":           "http://127.0.0.1:9999",
			"AGENTCTL_AGENT_RUN_ID":         "run-1",
			"FLOWGENTIC_ENABLE_DEFAULT_MCP": "1",
		},
	})

	require.Len(t, servers, 1)
	require.NotNil(t, servers[0].Stdio)
	assert.Equal(t, "flowgentic", servers[0].Stdio.Name)
}
