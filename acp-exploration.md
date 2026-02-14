# ACP (Agent Client Protocol) Exploration

## What is ACP?

ACP standardizes communication between clients (editors, control planes) and AI coding agents. It's conceptually similar to LSP but for coding agents. Built on **JSON-RPC 2.0 over stdio** (newline-delimited JSON).

- Spec: https://agentclientprotocol.com
- Go SDK: https://github.com/coder/acp-go-sdk

## Current Driver Architecture

We have 4 custom driver implementations (~2000+ lines total), each with its own transport:

| Driver | Transport | Lines | Streaming | Session Resume | Permissions |
|--------|-----------|-------|-----------|---------------|-------------|
| Claude Code | Go SDK (`claude-agent-sdk-go`) | ~650 | Yes (SDK stream) | Yes | Yes (callback) |
| Codex | JSON-RPC over stdin/stdout (shared `app-server` subprocess) | ~950 | Yes (notifications) | No | Yes (server request) |
| OpenCode | HTTP API + SSE (`opencode serve`) | ~780 | Yes (SSE) | Yes | Yes (SSE event) |
| Gemini | Subprocess batch output | ~340 | No | No | No |

Each driver has its own `normalize.go` to convert agent-specific events into our unified `driver.Event` type.

## ACP Agent Support

| Agent | ACP Support | Launch Command | Notes |
|-------|------------|----------------|-------|
| **Claude Code** | Via Zed adapter | `npx @zed-industries/claude-code-acp` | No native `--acp` flag. Adapter replaces built-in tools (Read/Write/Edit/Bash) with client-routed MCP wrappers — client MUST implement file/terminal handlers. |
| **OpenCode** | Native | `opencode acp` | Built-in, replaces current HTTP/SSE approach. |
| **Gemini CLI** | Native (experimental) | `gemini --experimental-acp` | Built-in but behind experimental flag. |
| **Codex** | Unknown | — | Not confirmed as ACP adopter. Already uses JSON-RPC so semantically close. |

## How ACP Maps to Our Driver Abstractions

### Core Interfaces

| Our Abstraction | ACP Equivalent | Notes |
|----------------|----------------|-------|
| `Driver.Launch()` | `initialize` → `session/new` → `session/prompt` | ACP splits launch into explicit phases |
| `Session.Stop()` | `session/cancel` notification | Same semantics |
| `Session.Wait()` | Wait for `session/prompt` response | ACP returns a `stopReason` |
| `EventCallback` | `Client.SessionUpdate()` method | Streaming updates dispatched as callbacks while `Prompt()` blocks |
| `RespondToPermission()` | `Client.RequestPermission()` | Nearly identical flow, synchronous blocking |
| `Capabilities` | `agentCapabilities` / `clientCapabilities` | ACP has richer negotiation |

### Event Type Mapping

| Our `EventType` | ACP `SessionUpdate` | Match |
|-----------------|---------------------|-------|
| `EventTypeMessage` | `AgentMessageChunk` | Direct |
| `EventTypeToolStart` | `ToolCall` (status: pending/in_progress) | Direct |
| `EventTypeToolResult` | `ToolCallUpdate` (status: completed/failed) | Direct |
| `EventTypeThinking` | `AgentThoughtChunk` | Direct |
| `EventTypeTurnComplete` | `session/prompt` response with `stopReason` | Direct |
| `EventTypeSessionStart` | `initialize` response | Direct |
| `EventTypeError` | JSON-RPC error / tool_call failed | Direct |
| `EventTypeCostUpdate` | No ACP equivalent | **Gap** — would need `_meta` extension |
| `EventTypePermissionRequest` | `RequestPermission` (agent→client) | Direct, inverted direction |

## Feature Coverage

| Feature | ACP Standard? | How |
|---------|--------------|-----|
| Prompt | Yes | `session/prompt` with content blocks |
| Streaming | Yes | `session/update` notifications |
| Permissions | Yes | `session/request_permission` (synchronous) |
| Session resume | Yes | `session/load` (if agent advertises `loadSession`) |
| Cwd | Yes | `session/new` → `cwd` field |
| Modes | Partially | `session/set_mode` — but mode IDs are agent-specific |
| **System prompt** | **No** | Adapter uses `_meta.systemPrompt`, not standardized |
| **Model selection** | **Partially** | `session/set_config_option` (category: `model`), agent-dependent |
| **Model listing** | Yes | Agent exposes a `SessionConfigOption` with category `model`, type `select`, and an `options` array — can populate a UI dropdown |
| **Yolo** | **No** | Agent-specific mode name (`"bypassPermissions"` for Claude) |
| **Allowed tools** | **No** | No equivalent, would need `_meta` extension |
| **Env vars** | **No** | Handled at subprocess launch (`cmd.Env`), same as today |
| **Cost tracking** | **No** | Not in ACP, would need `_meta` extension |
| **Hooks** | **No** | Not in ACP, flowgentic-specific |

## Go SDK (`coder/acp-go-sdk`)

The SDK provides a ready-made Go client for connecting to any ACP agent:

```go
// Launch agent subprocess
cmd := exec.CommandContext(ctx, "opencode", "acp")
stdin, _ := cmd.StdinPipe()
stdout, _ := cmd.StdoutPipe()
cmd.Start()

// Connect
client := &myACPClient{} // implements acp.Client interface
conn := acp.NewClientSideConnection(client, stdin, stdout)

// Lifecycle
conn.Initialize(ctx, initReq)
sess, _ := conn.NewSession(ctx, newSessionReq)
resp, _ := conn.Prompt(ctx, promptReq) // blocks, streams via SessionUpdate callback
```

Streaming model: `Prompt()` blocks. While waiting, the SDK dispatches `session/update` notifications to `Client.SessionUpdate()`. No channels — pure callbacks.

### Client Interface to Implement

```go
type Client interface {
    SessionUpdate(ctx, SessionNotification) error       // streaming updates
    RequestPermission(ctx, RequestPermissionRequest) (RequestPermissionResponse, error)
    ReadTextFile(ctx, ReadTextFileRequest) (ReadTextFileResponse, error)    // optional
    WriteTextFile(ctx, WriteTextFileRequest) (WriteTextFileResponse, error) // optional
    CreateTerminal(ctx, CreateTerminalRequest) (CreateTerminalResponse, error) // optional
    // ... more terminal methods
}
```

Client capabilities (fs, terminal) are **optional** — if not advertised, the agent uses its own built-in tools. Exception: the Claude Code adapter always requires them.

## Key Concerns

### Claude Code Adapter Forces Client-Side File I/O

The `claude-code-acp` adapter (from Zed) replaces Claude's built-in Read/Write/Edit/Bash tools with MCP wrappers that route all file and terminal operations back through the ACP client. This means:

- We **must** implement `ReadTextFile`, `WriteTextFile`, `CreateTerminal`, etc.
- The agent cannot do file I/O on its own
- This adds complexity our current Claude driver avoids (the SDK handles everything internally)

This is a Zed-specific design choice (editors want control over buffers/terminals), not an ACP requirement. But since there's no native `claude --acp` flag, this adapter is currently the only path.

### Non-Standard Configuration

System prompts, model selection, yolo mode, and allowed tools are not standardized in ACP. Each agent handles them differently via `_meta` extensions or agent-specific mode names. We'd still need per-agent knowledge for setup configuration, even with a unified transport layer.

## Model Listing via Config Options

ACP agents can advertise available models as a `SessionConfigOption` with category `"model"` and type `"select"`. This returns a list of selectable values that can directly populate a UI dropdown:

```json
{
  "id": "model",
  "name": "Model",
  "category": "model",
  "type": "select",
  "currentValue": "claude-sonnet-4",
  "options": [
    {"value": "claude-sonnet-4", "label": "Sonnet 4"},
    {"value": "claude-opus-4", "label": "Opus 4"}
  ]
}
```

- Returned as part of `NewSessionResponse` (available config options for the session)
- Changed via `session/set_config_option` request
- Agent can push updates via `config_options_update` notification (e.g., if available models change)
- Our frontend model dropdown could be populated directly from this, removing the need to hardcode model lists per agent

## Potential Benefits

- Replace ~2000 lines of 4 custom drivers with a single ACP client (~500 lines) + per-agent launch configs
- Eliminate all `normalize.go` files — the Go SDK handles JSON-RPC and message dispatch
- Eliminate HTTP port management, SSE parsing, health polling (OpenCode)
- Eliminate custom JSON-RPC implementation (Codex)
- Unified permission flow across all agents
- Future-proof: new ACP-compatible agents work with zero driver code

## Recommendation

ACP is a strong fit for the **transport and session lifecycle layer** but has gaps in **agent configuration** (system prompt, model, yolo). A practical approach:

1. Implement a generic `acp.Driver` using `coder/acp-go-sdk` for the core loop
2. Keep per-agent launch configs (CLI args, env vars) for non-standard features
3. Start with OpenCode and Gemini (native ACP, no adapter quirks)
4. Defer Claude Code until either Anthropic adds native ACP support or we're willing to implement client-side file/terminal handlers
5. Check Codex ACP status before migrating

## Resources

- ACP Spec: https://agentclientprotocol.com/get-started/introduction
- Go SDK: https://github.com/coder/acp-go-sdk
- Claude Code Adapter: https://github.com/zed-industries/claude-code-acp
- OpenCode ACP Docs: https://opencode.ai/docs/acp/
