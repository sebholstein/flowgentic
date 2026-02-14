package v2

import (
	"log/slog"

	acp "github.com/coder/acp-go-sdk"
	"github.com/sebastianm/flowgentic/internal/worker/driver"
)

// AgentConfig describes how to launch an ACP agent.
type AgentConfig struct {
	AgentID      string
	Capabilities []driver.Capability

	// For native ACP agents (subprocess).
	Command string
	Args    []string

	// For in-process adapted agents.
	AdapterFactory func(log *slog.Logger) acp.Agent

	// MetaBuilder constructs the _meta field for NewSession/Prompt from LaunchOpts.
	MetaBuilder func(opts LaunchOpts) map[string]any
}

// defaultMetaBuilder produces a _meta map from common LaunchOpts fields.
func defaultMetaBuilder(opts LaunchOpts) map[string]any {
	meta := map[string]any{}
	if opts.SystemPrompt != "" {
		meta["systemPrompt"] = opts.SystemPrompt
	}
	if opts.Model != "" {
		meta["model"] = opts.Model
	}
	if opts.SessionMode != "" {
		meta["sessionMode"] = opts.SessionMode
	}
	if len(opts.AllowedTools) > 0 {
		meta["allowedTools"] = opts.AllowedTools
	}
	if len(opts.EnvVars) > 0 {
		meta["envVars"] = opts.EnvVars
	}
	return meta
}

// Pre-built configs for known agents.

var OpenCodeConfig = AgentConfig{
	AgentID: string(driver.AgentTypeOpenCode),
	Capabilities: []driver.Capability{
		driver.CapStreaming,
		driver.CapCustomModel,
		driver.CapSystemPrompt,
		driver.CapPermissionRequest,
		driver.CapCostTracking,
	},
	Command:     "opencode",
	Args:        []string{"acp"},
	MetaBuilder: defaultMetaBuilder,
}

var GeminiConfig = AgentConfig{
	AgentID: string(driver.AgentTypeGemini),
	Capabilities: []driver.Capability{
		driver.CapStreaming,
		driver.CapCustomModel,
		driver.CapSystemPrompt,
	},
	Command:     "gemini",
	Args:        []string{"--experimental-acp"},
	MetaBuilder: defaultMetaBuilder,
}

// ClaudeCodeConfig is set by the claude/acp package via SetClaudeCodeConfig.
var ClaudeCodeConfig = AgentConfig{
	AgentID: string(driver.AgentTypeClaudeCode),
	Capabilities: []driver.Capability{
		driver.CapStreaming,
		driver.CapSessionResume,
		driver.CapCostTracking,
		driver.CapCustomModel,
		driver.CapSystemPrompt,
		driver.CapPermissionRequest,
	},
	MetaBuilder: defaultMetaBuilder,
}

// CodexConfig is set by the codex/acp package via SetCodexConfig.
var CodexConfig = AgentConfig{
	AgentID: string(driver.AgentTypeCodex),
	Capabilities: []driver.Capability{
		driver.CapStreaming,
		driver.CapCustomModel,
		driver.CapSystemPrompt,
		driver.CapPermissionRequest,
	},
	MetaBuilder: defaultMetaBuilder,
}
