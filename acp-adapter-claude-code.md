# ACP Adapter Plan: Claude Code (Go)

## Goal

Build a Go binary that wraps the `claude-agent-sdk-go` SDK and exposes Claude Code as an ACP-compatible agent over stdio. Unlike the Zed TypeScript adapter, this adapter **does not replace built-in tools** — Claude keeps using its own Read/Write/Edit/Bash tools directly. The client does not need to implement file/terminal handlers.

## Why Not Use the Zed Adapter?

The existing `@zed-industries/claude-code-acp` (TypeScript):
- Requires Node.js runtime (`npx`)
- Replaces all built-in tools with client-routed MCP wrappers (client must implement `ReadTextFile`, `WriteTextFile`, `CreateTerminal`, etc.)
- Designed for editors that want control over buffers/terminals

For our headless worker, we want Claude to handle its own file I/O. A Go adapter using the same SDK we already use is simpler and avoids the Node.js dependency.

## Architecture

```
flowgentic worker
    └── spawns: claude-code-acp-go (our adapter binary)
            ├── stdin/stdout: ACP JSON-RPC 2.0 (via coder/acp-go-sdk)
            └── internally: claude-agent-sdk-go (Go SDK)
```

The adapter is an ACP **agent** (implements `acp.Agent` interface) that translates ACP requests into Claude SDK calls and Claude SDK events into ACP notifications.

## Dependencies

- `github.com/coder/acp-go-sdk` — ACP agent-side connection
- `github.com/severity1/claude-agent-sdk-go` — Claude Code Go SDK (already used in our driver)

## ACP Agent Interface Implementation

### `Initialize(ctx, InitializeRequest) → InitializeResponse`

Return agent capabilities:
```go
InitializeResponse{
    ProtocolVersion: acp.ProtocolVersionNumber,
    AgentInfo: AgentInfo{Name: "claude-code", Version: "1.0.0"},
    AgentCapabilities: AgentCapabilities{
        LoadSession: true,  // Claude supports session resume
        PromptCapabilities: PromptCapabilities{
            Image:           true,
            EmbeddedContext: true,
        },
    },
}
```

No auth methods needed — Claude uses API key from environment.

### `Authenticate(ctx, AuthenticateRequest) → AuthenticateResponse`

Not needed. Return error or no-op. API key is passed via `ANTHROPIC_API_KEY` env var at subprocess launch.

### `NewSession(ctx, NewSessionRequest) → NewSessionResponse`

- Store `cwd` from request
- Generate a session ID (UUID)
- Parse `_meta` for non-standard options:
  - `_meta.systemPrompt` → stored for SDK `WithSystemPrompt()`
  - `_meta.model` → stored for SDK `WithModel()`
  - `_meta.yolo` → stored for SDK `WithPermissionMode(BypassPermissions)`
  - `_meta.allowedTools` → stored for SDK `WithAllowedTools()`
  - `_meta.envVars` → stored for SDK `WithEnv()`
- Return available modes:
  - `default` — normal permission flow
  - `bypassPermissions` — yolo mode
  - `plan` — planning mode (if supported)
- Return config options including available models (if we can enumerate them)
- **Do not start the SDK yet** — wait for first `Prompt()`

### `Prompt(ctx, PromptRequest) → PromptResponse`

This is the core method. On first call, start the SDK; on subsequent calls, send follow-up prompts.

**First prompt (session start):**
1. Build SDK options from stored session config:
   ```go
   opts := []claudecode.Option{
       claudecode.WithCwd(session.cwd),
       claudecode.WithPartialStreaming(),
   }
   if session.model != "" { opts = append(opts, claudecode.WithModel(session.model)) }
   if session.systemPrompt != "" { opts = append(opts, claudecode.WithSystemPrompt(session.systemPrompt)) }
   if session.yolo { opts = append(opts, claudecode.WithPermissionMode(PermissionModeBypassPermissions)) }
   if len(session.allowedTools) > 0 { opts = append(opts, claudecode.WithAllowedTools(session.allowedTools...)) }
   if len(session.envVars) > 0 { opts = append(opts, claudecode.WithEnv(session.envVars)) }
   // Permission callback (see below)
   opts = append(opts, claudecode.WithCanUseTool(session.handlePermission))
   ```
2. Call `claudecode.WithClient(ctx, callback, opts...)` in a goroutine
3. Inside callback: `client.QueryWithSession(ctx, prompt, sessionID)`
4. Consume `client.ReceiveMessages(ctx)` channel, translating each message to ACP notifications

**Subsequent prompts:**
- The SDK's `QueryWithSession` needs to be called again — but `WithClient` is a blocking callback. We need to structure the adapter so the SDK client stays alive across prompts.
- Approach: Keep the `WithClient` goroutine running. Use a channel to feed new prompts to it. The goroutine calls `QueryWithSession` for each prompt and streams results.

**Message → ACP notification mapping:**

| SDK Message | ACP Notification |
|---|---|
| `*StreamEvent` (content_block_delta, type=text) | `AgentMessageChunk` with `TextBlock` |
| `*StreamEvent` (content_block_delta, type=thinking) | `AgentThoughtChunk` with `TextBlock` |
| `*StreamEvent` (content_block_start, type=tool_use) | `ToolCall` (status: pending, kind from tool name) |
| `*AssistantMessage` → `*TextBlock` | `AgentMessageChunk` with `TextBlock` |
| `*AssistantMessage` → `*ThinkingBlock` | `AgentThoughtChunk` with `TextBlock` |
| `*AssistantMessage` → `*ToolUseBlock` | `ToolCall` (status: in_progress) |
| `*AssistantMessage` → `*ToolResultBlock` | `ToolCallUpdate` (status: completed/failed) |
| `*ResultMessage` | Return `PromptResponse` with `StopReason` |
| `*SystemMessage` | `AgentMessageChunk` (extract nested text) |

### `Cancel(ctx, CancelNotification)`

Cancel the context passed to `WithClient` / `QueryWithSession`. The SDK should abort the current stream.

### `SetSessionMode(ctx, SetSessionModeRequest) → SetSessionModeResponse`

Map mode IDs to SDK options:
- `"default"` → normal permission mode
- `"bypassPermissions"` → `WithPermissionMode(BypassPermissions)`
- `"plan"` → TBD (if Claude SDK supports plan mode)

Note: Mode changes may require restarting the SDK session since options are set at initialization.

### Permission Flow

The SDK calls `WithCanUseTool(callback)` when a tool needs approval:

1. Adapter sends `conn.RequestPermission()` to client with tool details
2. Client responds with allow/deny
3. Adapter returns `PermissionResultAllow()` or `PermissionResultDeny(reason)` to SDK

```go
func (s *session) handlePermission(ctx context.Context, toolName string, input map[string]any, permCtx ToolPermissionContext) (PermissionResult, error) {
    resp, err := s.conn.RequestPermission(ctx, acp.RequestPermissionRequest{
        SessionId: s.sessionID,
        ToolCall: acp.ToolCallUpdate{
            ToolCallId: acp.ToolCallId(uuid.New().String()),
            Title:      acp.Ptr(toolName),
            Kind:       acp.Ptr(mapToolKind(toolName)),
            Status:     acp.Ptr(acp.ToolCallStatusPending),
            RawInput:   input,
        },
        Options: []acp.PermissionOption{
            {Kind: acp.PermissionOptionKindAllowOnce, Name: "Allow", OptionId: "allow"},
            {Kind: acp.PermissionOptionKindRejectOnce, Name: "Deny", OptionId: "deny"},
        },
    })
    if err != nil { return PermissionResultDeny("error"), err }
    if resp.Outcome.Selected != nil && resp.Outcome.Selected.OptionId == "allow" {
        return PermissionResultAllow(), nil
    }
    return PermissionResultDeny("user denied"), nil
}
```

## Key Design Decisions

### Built-in Tools Stay With Claude

Unlike the Zed adapter, we do **not** replace Read/Write/Edit/Bash. Claude uses its own tools directly. The client does not need to advertise `fs` or `terminal` capabilities.

This means:
- Simpler client implementation (our worker)
- No file content routing overhead
- Claude works exactly as it does today
- We lose the ability for the client to intercept/observe file operations (acceptable for headless)

### SDK Lifecycle Across Prompts

The `claudecode.WithClient()` API is a blocking callback — the SDK client only lives within that callback. For multi-turn ACP sessions, we need to keep the callback alive:

```go
go claudecode.WithClient(ctx, func(client claudecode.Client) error {
    for prompt := range s.promptChan {
        client.QueryWithSession(ctx, prompt.text, s.sessionID)
        for msg := range client.ReceiveMessages(ctx) {
            s.translateAndEmit(msg)
        }
        prompt.done <- struct{}{}
    }
    return nil
}, opts...)
```

The `Prompt()` ACP method pushes to `promptChan` and waits on `done`.

### Session Resume

For `LoadSession()`:
- Use `WithResume(sessionID)` SDK option
- Replay the session history as `session/update` notifications (if the SDK provides history)
- If the SDK doesn't provide history replay, just reconnect and note the limitation

## Open Questions

1. **Does the Claude Go SDK support multiple `QueryWithSession` calls within one `WithClient` callback?** If not, we may need to restart the SDK per prompt turn (losing streaming state).
2. **Can we enumerate available models?** The SDK may not expose a model list. We might need to hardcode known models or fetch from the Anthropic API separately.
3. **Plan mode support** — Does the Go SDK have a way to toggle plan/architect mode mid-session?
4. **Session history replay** — When resuming a session, can we get the conversation history to send as `user_message_chunk` / `agent_message_chunk` notifications?

## File Structure

```
cmd/claude-code-acp/
├── main.go           # Entry point, stdio setup, AgentSideConnection
├── agent.go          # acp.Agent implementation (Initialize, NewSession, Prompt, etc.)
├── session.go        # Per-session state, SDK lifecycle, prompt channel
├── translate.go      # SDK Message → ACP SessionUpdate translation
└── permission.go     # Permission callback bridging SDK ↔ ACP
```

## Rough Effort Estimate

- ~400-500 lines of Go code
- Main complexity: SDK lifecycle management across multi-turn prompts
- Translation logic is straightforward (similar to existing `normalize.go`)
