# driver/v2 — ACP-based Agent Driver

Unified driver layer for launching and managing AI coding agents via the [Agent Client Protocol (ACP)](https://agentclientprotocol.com). Supports both **subprocess** agents (native ACP over stdio) and **in-process** agents (adapted via `acp.Agent` interface).

## Architecture

```
AgentRunManager
  └─ Driver (per agent type)
       └─ Launch(opts, onEvent) → Session
            ├─ ACP Initialize
            ├─ ACP NewSession
            └─ ACP Prompt → streams SessionNotifications via onEvent
```

The driver handles the full ACP lifecycle: initialize the connection, create a session, send the prompt, and stream events back through the `EventCallback`.

## Key Types

| Type | Description |
|---|---|
| `Driver` | Interface — launches sessions for a specific agent type |
| `AgentConfig` | Declares how to launch an agent (subprocess command or in-process adapter) |
| `LaunchOpts` | Per-session options: prompt, model, cwd, system prompt, env vars, allowed tools, and MCP server attachments. |
| `Session` | A running agent session — inspect status, stop, wait, respond to permissions |
| `EventCallback` | `func(acp.SessionNotification)` — receives streaming updates |

## Adding a New Agent

### Subprocess Agent (native ACP)

For agents that speak ACP natively over stdio:

```go
var MyAgentConfig = v2.AgentConfig{
    AgentID:      "my-agent",
    Capabilities: []driver.Capability{driver.CapStreaming, driver.CapCustomModel},
    Command:      "my-agent-binary",
    Args:         []string{"--acp"},
    MetaBuilder:  v2.DefaultMetaBuilder,
}

// Create the driver.
d := v2.NewDriver(logger, MyAgentConfig)
```

The driver spawns the command, wires stdin/stdout to an ACP connection, and runs the protocol.

### In-Process Adapter

For agents that don't speak ACP natively (e.g. Claude Code, Codex), write an adapter implementing `acp.Agent`:

```go
var MyAgentConfig = v2.AgentConfig{
    AgentID:      "my-agent",
    Capabilities: []driver.Capability{driver.CapStreaming},
    AdapterFactory: func(log *slog.Logger) acp.Agent {
        return myadapter.NewAdapter(log)
    },
    MetaBuilder: defaultMetaBuilder,
}
```

The adapter receives ACP calls (`Initialize`, `NewSession`, `Prompt`) and translates them to the agent's native SDK. It sends streaming updates back via the `AgentSideConnection`:

```go
func (a *MyAdapter) SetConnection(conn *acp.AgentSideConnection) {
    a.conn = conn
}

// Inside Prompt(), stream updates back:
a.conn.SessionUpdate(ctx, acp.SessionNotification{
    SessionId: sessionID,
    Update:    acp.UpdateAgentMessageText("hello"),
})
```

Adapters must implement `v2.ConnectionSetter` to receive the agent-side connection reference.

## Launching a Session

```go
d := v2.NewDriver(logger, config)

sess, err := d.Launch(ctx, v2.LaunchOpts{
    Prompt:       "Fix the failing test",
    SystemPrompt: "You are a helpful assistant.",
    Model:        "claude-sonnet-4-5-20250929",
    Cwd:          "/path/to/project",
    AllowedTools: []string{"Bash(go test ./...)"},
    MCPServers:   []acp.McpServer{/* optional extra MCP servers */},
}, func(n acp.SessionNotification) {
    // Handle streaming events: messages, tool calls, plans, etc.
    fmt.Println("event:", n.Update)
})

// Wait for completion.
sess.Wait(ctx)
```

## Session Lifecycle

1. **Starting** — `Launch` called, ACP connection being established
2. **Running** — ACP session created, prompt in progress, events streaming
3. **Stopped** — Prompt completed or `sess.Stop()` called
4. **Errored** — ACP protocol error or agent failure

## Meta Builder

The `MetaBuilder` function on `AgentConfig` controls how `LaunchOpts` fields are passed to the agent via the ACP `_meta` field on `NewSession`. The default builder maps `SystemPrompt`, `Model`, `SessionMode`, `AllowedTools`, and `EnvVars`. Override it for agents that need custom meta fields.

## Session-Scoped MCP Servers

- `LaunchOpts.MCPServers` is passed through to ACP `NewSession`/`LoadSession`.
- The worker injects a default Flowgentic MCP stdio server (`agentctl mcp serve`) only when Flowgentic MCP mode is requested (`SystemPrompt` contains `## Flowgentic MCP`, or `FLOWGENTIC_ENABLE_DEFAULT_MCP=1`) and `AGENTCTL_WORKER_URL` plus `AGENTCTL_SESSION_ID` are present in `LaunchOpts.EnvVars`.
- Model discovery intentionally uses an empty MCP server list.

## Pre-built Configs

| Config | Agent | Transport |
|---|---|---|
| `ClaudeCodeConfig` | Claude Code | In-process adapter |
| `CodexConfig` | Codex | In-process adapter |
| `OpenCodeConfig` | OpenCode | Subprocess (`opencode acp`) |
| `GeminiConfig` | Gemini CLI | Subprocess (`gemini --experimental-acp`) |

## Capabilities

Drivers declare what they support via `AgentConfig.Capabilities`. The `AgentRunManager` checks these before launching:

- `streaming` — Real-time event streaming
- `session_resume` — Resume a previous session by ID
- `custom_model` — Accepts a model override
- `system_prompt` — Accepts a system prompt
- `yolo` — Auto-approve all tool calls
- `permission_request` — Supports interactive permission prompts
- `cost_tracking` — Reports token/cost usage
