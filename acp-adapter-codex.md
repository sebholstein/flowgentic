# ACP Adapter Plan: Codex (Go)

## Goal

Build a Go binary that wraps the Codex `app-server` JSON-RPC protocol and exposes it as an ACP-compatible agent over stdio. This is essentially a **protocol translator** between two JSON-RPC dialects — Codex's custom protocol and ACP's standardized one.

## Why an Adapter?

Codex does not appear to have native ACP support. However, its existing protocol is already JSON-RPC 2.0 over stdio, making it the closest to ACP of all our current drivers. The adapter translates between the two protocols, letting our unified ACP client talk to Codex without a custom driver.

## Architecture

```
flowgentic worker
    └── spawns: codex-acp (our adapter binary)
            ├── stdin/stdout: ACP JSON-RPC 2.0 (via coder/acp-go-sdk)
            └── spawns internally: codex app-server
                    ├── stdin/stdout: Codex JSON-RPC 2.0
                    └── manages: codex agent subprocess
```

The adapter sits between the ACP client and the Codex app-server, translating requests in both directions.

## Dependencies

- `github.com/coder/acp-go-sdk` — ACP agent-side connection
- Codex CLI (`codex` binary) — spawned as subprocess in app-server mode

## Protocol Translation

### Lifecycle Mapping

| ACP Phase | Codex Equivalent |
|---|---|
| `initialize` | `initialize` + `initialized` |
| `session/new` | `thread/start` |
| `session/prompt` | `turn/start` (blocks until `turn/completed`) |
| `session/cancel` | `turn/interrupt` |

### ACP Agent Interface Implementation

#### `Initialize(ctx, InitializeRequest) → InitializeResponse`

1. Start the Codex app-server subprocess: `codex --app-server`
2. Send Codex `initialize` request with client info and capabilities
3. Send Codex `initialized` notification
4. Return ACP capabilities:

```go
InitializeResponse{
    ProtocolVersion: acp.ProtocolVersionNumber,
    AgentInfo: AgentInfo{Name: "codex", Version: "1.0.0"},
    AgentCapabilities: AgentCapabilities{
        // Codex does NOT support session resume or cost tracking
        PromptCapabilities: PromptCapabilities{},
    },
}
```

#### `Authenticate(ctx, AuthenticateRequest) → AuthenticateResponse`

Not needed. Codex uses `OPENAI_API_KEY` from environment.

#### `NewSession(ctx, NewSessionRequest) → NewSessionResponse`

1. Send Codex `thread/start` with:
   - `cwd` from ACP request
   - `model` from `_meta.model` (if provided)
   - `approvalPolicy`: `"never"` if yolo mode, `"on-failure"` otherwise
   - `developerInstructions` from `_meta.systemPrompt` (if provided)
2. Receive `threadID` from response
3. Map ACP `sessionId` ↔ Codex `threadID`
4. Return available modes:
   - `default` — `approvalPolicy: "on-failure"`
   - `bypassPermissions` — `approvalPolicy: "never"`
5. Start listening for Codex notifications on this thread (in background goroutine)

#### `Prompt(ctx, PromptRequest) → PromptResponse`

1. Extract text from ACP content blocks
2. Send Codex `turn/start` with:
   - `threadId`: mapped from ACP `sessionId`
   - `input`: `[{type: "text", text: "..."}]`
   - `sandboxPolicy`: `{type: "dangerFullAccess"}` if yolo, `{type: "workspaceWrite", writableRoots: [cwd]}` otherwise
3. Receive `turnID` from response
4. Block until `turn/completed` notification, translating all intermediate notifications to ACP `session/update` calls
5. Return `PromptResponse{StopReason: "end_turn"}`

#### `Cancel(ctx, CancelNotification)`

Send Codex `turn/interrupt` with current `threadId` and `turnId`.

#### `SetSessionMode(ctx, SetSessionModeRequest) → SetSessionModeResponse`

Store mode for next `turn/start`. Codex doesn't support mid-thread policy changes — the `approvalPolicy` and `sandboxPolicy` are per-turn, so we apply the mode on the next prompt.

### Notification Translation (Codex → ACP)

This is the core of the adapter — translating Codex server notifications into ACP `session/update` calls.

| Codex Notification | ACP SessionUpdate | Details |
|---|---|---|
| `item/agentMessage/delta` | `AgentMessageChunk` | `TextBlock(delta)`, streaming |
| `item/reasoning/textDelta` | `AgentThoughtChunk` | `TextBlock(delta)`, streaming |
| `item/started` (type: commandExecution) | `ToolCall` | `status: pending`, `kind: execute`, `title: command` |
| `item/completed` (type: agentMessage) | `AgentMessageChunk` | `TextBlock(text)`, final message |
| `item/completed` (type: reasoning) | `AgentThoughtChunk` | `TextBlock(text)`, final thought |
| `item/completed` (type: commandExecution) | `ToolCallUpdate` | `status: completed/failed` based on `exitCode`, content from `aggregatedOutput` |
| `item/completed` (type: fileChange) | `ToolCallUpdate` | `status: completed`, `kind: edit`, locations from `changes[].path` |
| `turn/completed` | — | Unblocks `Prompt()`, returns `PromptResponse` |
| `error` | — | Return error from `Prompt()` |

### Permission Flow (Codex → ACP)

Codex sends `item/commandExecution/requestApproval` as a **JSON-RPC request** (with `id`), expecting a response.

1. Codex sends: `{id: N, method: "item/commandExecution/requestApproval", params: {command: "npm test"}}`
2. Adapter translates to ACP: `conn.RequestPermission(ctx, ...)`
   ```go
   resp, _ := s.conn.RequestPermission(ctx, acp.RequestPermissionRequest{
       SessionId: s.acpSessionID,
       ToolCall: acp.ToolCallUpdate{
           ToolCallId: acp.ToolCallId(uuid.New().String()),
           Title:      acp.Ptr(fmt.Sprintf("Execute: %s", command)),
           Kind:       acp.Ptr(acp.ToolKindExecute),
           Status:     acp.Ptr(acp.ToolCallStatusPending),
           RawInput:   map[string]any{"command": command},
       },
       Options: []acp.PermissionOption{
           {Kind: acp.PermissionOptionKindAllowOnce, Name: "Allow", OptionId: "accept"},
           {Kind: acp.PermissionOptionKindRejectOnce, Name: "Deny", OptionId: "deny"},
       },
   })
   ```
3. Client responds with allow/deny
4. Adapter responds to Codex: `{id: N, result: {decision: "accept"}}` or `{decision: "deny"}`

This is the trickiest part — the adapter must match the Codex request `id` to send back the correct response.

## App-Server Management

The adapter manages the Codex app-server subprocess lifecycle:

```go
type codexBridge struct {
    cmd     *exec.Cmd
    stdin   io.WriteCloser  // write JSON-RPC requests to app-server
    stdout  io.ReadCloser   // read JSON-RPC responses/notifications from app-server
    pending map[int]chan json.RawMessage  // request ID → response channel
    nextID  int
}
```

### Message Routing

The adapter reads lines from app-server stdout and routes them:

1. **Response** (has `id` + `result`/`error`) → route to pending request channel by `id`
2. **Request** (has `id` + `method`) → handle server-initiated request (permission approval)
3. **Notification** (has `method`, no `id`) → translate to ACP session update

This is essentially the same routing logic as our current `app_server.go`, but the output side speaks ACP instead of our custom `EventCallback`.

## Key Design Decisions

### Codex Handles Its Own File I/O

Like the Claude adapter, we do not advertise `fs` or `terminal` client capabilities. Codex runs commands and modifies files directly within its sandbox. The ACP client only receives notifications about what happened.

### One App-Server Per Adapter Instance

Each adapter instance spawns one Codex app-server. Multiple ACP sessions map to multiple Codex threads within the same app-server (same as our current driver's design).

### Turn-Based, Not Persistent Sessions

Codex does not support session resume. Each `session/new` creates a fresh thread. `LoadSession` is not advertised in capabilities.

## Open Questions

1. **Does Codex have native ACP support in development?** If Codex adds `codex --acp`, this adapter becomes unnecessary. Worth checking with the Codex team / GitHub issues.
2. **Sandbox policy flexibility** — The `sandboxPolicy` controls what Codex can do. Should we expose this as an ACP config option or mode?
3. **Multiple threads per app-server** — If the ACP client creates multiple sessions, they share one app-server. Is this the right model, or should each session get its own app-server?
4. **App-server crash recovery** — If the app-server dies mid-session, should the adapter try to restart it?

## File Structure

```
cmd/codex-acp/
├── main.go           # Entry point, stdio setup, AgentSideConnection
├── agent.go          # acp.Agent implementation
├── session.go        # Per-session state, thread/turn tracking
├── bridge.go         # Codex app-server subprocess + JSON-RPC client
├── translate.go      # Codex notifications → ACP SessionUpdate
└── permission.go     # Permission request bridging (Codex request ↔ ACP)
```

## Rough Effort Estimate

- ~500-600 lines of Go code
- Main complexity: bidirectional JSON-RPC routing (app-server ↔ ACP)
- The `bridge.go` is essentially a simplified version of our current `app_server.go`
- Translation logic is straightforward — Codex events map cleanly to ACP updates
