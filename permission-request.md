# Permission Request Flow

## Problem

When an agent session runs without YOLO mode, Claude will request permission before using certain tools (e.g. `Bash`, `Write`, `Edit`). Previously, with the subprocess-based driver, there was no way to handle these prompts in headless mode — the session would block indefinitely.

With the migration to the Go SDK (`claude-agent-sdk-go`), the `WithCanUseTool` callback gives us a programmatic hook that fires whenever Claude requests permission. The callback blocks until we return allow or deny. This unlocks interactive permission flows where a human reviews and approves tool usage in real time.

## Current Architecture

Events flow one direction — driver to control plane to frontend:

```
Claude SDK  →  Driver (EventCallback)  →  Worker RPC  →  Control Plane  →  Frontend
```

There is no return path. The driver's `Session` interface only exposes `Info()`, `Stop()`, and `Wait()`. The control plane has no way to send a response back into a running session.

## Proposed Solution

### 1. Driver Layer

Add a permission request/response mechanism to the session.

```go
// New event type emitted when Claude requests permission.
EventTypePermissionRequest EventType = "permission_request"

// New fields on Event for permission requests.
type Event struct {
    // ... existing fields ...
    PermissionRequestID string `json:"permission_request_id,omitempty"`
}

// PermissionResponder is implemented by sessions that support interactive permission flows.
type PermissionResponder interface {
    // RespondToPermission resolves a pending permission request.
    // The requestID correlates to the PermissionRequestID in the emitted event.
    // If allow is true, the tool execution proceeds. If false, it is denied with the given reason.
    RespondToPermission(ctx context.Context, requestID string, allow bool, reason string) error
}
```

Inside the claude driver, the `WithCanUseTool` callback:

1. Generates a unique `requestID`.
2. Creates a response channel and stores it in a `pendingPermissions` map on the session.
3. Emits an `EventTypePermissionRequest` event with the tool name, input, and `requestID`.
4. Blocks on the response channel (with context cancellation support).
5. Returns `PermissionResultAllow` or `PermissionResultDeny` based on the response.

`RespondToPermission` looks up the channel by `requestID` and sends the decision.

```go
type permissionResponse struct {
    Allow  bool
    Reason string
}

type claudeSession struct {
    // ... existing fields ...
    pendingPermissions map[string]chan permissionResponse
}
```

### 2. Worker Layer

The worker service handler needs a new RPC endpoint that the control plane can call to deliver permission responses:

```protobuf
// In worker_service.proto
rpc RespondToPermission(RespondToPermissionRequest) returns (RespondToPermissionResponse);

message RespondToPermissionRequest {
    string agent_run_id = 1;
    string permission_request_id = 2;
    bool allow = 3;
    string reason = 4;
}

message RespondToPermissionResponse {}
```

The handler looks up the session via the agent run manager and calls `RespondToPermission` on it.

### 3. Control Plane Layer

The control plane needs to:

- Receive `permission_request` events from the worker's event stream and persist them (or forward to the frontend via SSE/WebSocket).
- Expose an RPC endpoint for the frontend to submit a decision.
- Forward the decision to the worker via `RespondToPermission`.

```protobuf
// In thread_service.proto (or a new permission_service.proto)
rpc RespondToPermission(RespondToPermissionRequest) returns (RespondToPermissionResponse);

message RespondToPermissionRequest {
    string thread_id = 1;
    string permission_request_id = 2;
    bool allow = 3;
    string reason = 4;
}
```

### 4. Frontend

The frontend displays a permission request card when a `permission_request` event arrives:

- Shows the tool name, input parameters, and any context.
- Provides Allow / Deny buttons.
- Optionally supports "Always allow this tool" which sets a session-level auto-allow rule.
- Calls the control plane RPC with the decision.

## Event Flow

```
1. Claude decides to call Bash("npm install")
2. SDK invokes WithCanUseTool("Bash", {"command": "npm install"}, ctx)
3. Claude driver:
   a. Generates requestID = "perm-abc123"
   b. Emits Event{Type: permission_request, ToolName: "Bash", ToolInput: ..., PermissionRequestID: "perm-abc123"}
   c. Blocks on pendingPermissions["perm-abc123"]
4. Event flows: Worker → Control Plane → Frontend (SSE)
5. Frontend shows: "Claude wants to run: npm install" [Allow] [Deny]
6. User clicks Allow
7. Frontend → Control Plane RPC → Worker RPC → session.RespondToPermission("perm-abc123", true, "")
8. Channel unblocks, callback returns PermissionResultAllow
9. Claude executes Bash("npm install") and continues
```

## Capabilities

Add a new driver capability so the system knows which drivers support interactive permissions:

```go
CapPermissionRequest Capability = "permission_request"
```

Only drivers that implement `PermissionResponder` advertise this capability. For drivers that don't support it (Codex, Gemini), the existing behavior (yolo or fail) remains unchanged.

## Implementation Order

1. **Driver interface**: Add `EventTypePermissionRequest`, `PermissionResponder`, event fields.
2. **Claude driver**: Implement `WithCanUseTool` callback + `RespondToPermission` + pending map.
3. **Worker RPC**: Add `RespondToPermission` endpoint, wire through agent run manager.
4. **Control plane**: Forward permission events, add `RespondToPermission` RPC.
5. **Frontend**: Permission request card component + RPC call.

## Open Questions

- Should permission decisions be persisted (audit log)?
- Should there be project-level or thread-level default permission rules (e.g. "always allow Read in this project")?
- Should the control plane support delegating permission decisions to an automated policy engine instead of a human?
