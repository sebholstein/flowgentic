# Permission Request — OpenCode Driver

## Status: Supported (via HTTP API)

OpenCode's server mode exposes a permission endpoint:

```
POST /session/:id/permissions/:permissionID
Body: { "response": "allow" | "deny", "remember"?: boolean }
```

Permission requests likely arrive as SSE events on `GET /global/event`. The driver already consumes this SSE stream in `consumeSSE()`.

## Current Behavior

The OpenCode driver currently has **no permission handling**. It doesn't process any permission-related SSE events, and it always runs with whatever default OpenCode uses. There's no auto-accept logic — permissions are simply ignored, which likely means the session blocks or times out when a permission prompt fires.

## Implementation Plan

### 1. Shared Driver Types

Same as other drivers — uses `EventTypePermissionRequest`, `PermissionResponder`, and `CapPermissionRequest` from `internal/worker/driver/`.

### 2. Discover the SSE Permission Event Shape

Before implementing, we need to confirm the exact SSE event type and payload for permission requests. Expected shape (needs verification against actual OpenCode behavior):

```json
{
  "type": "session.permission",
  "properties": {
    "permissionID": "perm-abc123",
    "sessionID": "sess-xyz",
    "tool": "bash",
    "input": { "command": "npm install" }
  }
}
```

**Action:** Run OpenCode in non-yolo mode, trigger a tool use, and capture the SSE event. Log unknown SSE event types in `normalizeSSEEvent` to discover the exact format.

### 3. OpenCode Driver Changes (`internal/worker/driver/opencode/`)

**opencode.go — add pending permissions map to session:**

```go
type permissionResponse struct {
    Allow  bool
    Reason string
}

type openCodeSession struct {
    // ... existing fields ...
    pendingPermissions   map[string]chan permissionResponse
    pendingPermissionsMu sync.Mutex
}
```

**opencode.go — handle permission SSE events in `normalizeSSEEvent`:**

Add a new case for the permission event type (exact type TBD):

```go
case "session.permission": // exact type name TBD
    var props struct {
        PermissionID string          `json:"permissionID"`
        Tool         string          `json:"tool"`
        Input        json.RawMessage `json:"input"`
    }
    _ = json.Unmarshal(evt.Properties, &props)
    return []driver.Event{{
        Type:                driver.EventTypePermissionRequest,
        Timestamp:           now,
        Agent:               agent,
        ToolName:            props.Tool,
        ToolInput:           props.Input,
        PermissionRequestID: props.PermissionID,
    }}
```

**Key difference from Claude/Codex:** OpenCode permissions don't block in-process. Instead:
- The SSE event notifies us of a pending permission.
- We emit a `EventTypePermissionRequest` up the stack.
- When the human responds, we call `POST /session/:id/permissions/:permissionID`.
- OpenCode's server handles unblocking internally.

This means `RespondToPermission` makes an HTTP call instead of unblocking a channel:

```go
func (s *openCodeSession) RespondToPermission(ctx context.Context, requestID string, allow bool, reason string) error {
    s.mu.Lock()
    serverURL := s.serverURL
    sessionID := s.openCodeSessionID
    s.mu.Unlock()

    if serverURL == "" || sessionID == "" {
        return fmt.Errorf("opencode session not ready")
    }

    response := "deny"
    if allow {
        response = "allow"
    }

    body := struct {
        Response string `json:"response"`
    }{Response: response}

    payload, err := json.Marshal(body)
    if err != nil {
        return err
    }

    url := fmt.Sprintf("%s/session/%s/permissions/%s", serverURL, sessionID, requestID)
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(payload)))
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        respBody, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("permission response failed: HTTP %d: %s", resp.StatusCode, string(respBody))
    }

    return nil
}
```

**opencode.go — advertise capability:**

Add `driver.CapPermissionRequest` to `Capabilities()`.

### 4. Initialization

```go
sess := &openCodeSession{
    // ... existing fields ...
    pendingPermissions: make(map[string]chan permissionResponse),
}
```

(The pending map may not be needed if we go with the HTTP-call approach — the permission ID from the SSE event is the same one we POST back. No in-process blocking required.)

### 5. Investigation Needed

| Question | How to find out |
|----------|----------------|
| Exact SSE event type for permissions | Run OpenCode in non-yolo mode, log all SSE events |
| Exact payload shape of permission event | Same as above |
| Does OpenCode send permission events on the global SSE stream or a session-specific one? | Test or check OpenCode source |
| What values does `response` accept? (`"allow"` / `"deny"`? `true` / `false`?) | Test against the API |

## Architectural Advantage

Unlike Claude and Codex where the permission callback blocks a goroutine, OpenCode's HTTP-based approach is fully async. The SSE event arrives, we forward it as a driver event, and the response is a separate HTTP POST. No channels or pending maps needed for the core flow — the OpenCode server manages the blocking internally.

## Files Changed

| File | Change |
|------|--------|
| `internal/worker/driver/opencode/opencode.go` | Add permission SSE event handling, `RespondToPermission`, advertise capability |
