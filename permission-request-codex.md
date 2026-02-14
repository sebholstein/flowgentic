# Permission Request — Codex Driver

## Status: Fully Supported

Codex already has an interactive approval mechanism via its JSON-RPC app-server protocol. The `item/commandExecution/requestApproval` notification is a **server-initiated request** (has an `id` field, expects a response). Currently the driver **auto-accepts** all approval requests. This needs to be changed to emit a permission request event and wait for a human decision.

## Current Behavior

In `codex.go`, `dispatchNotification`:

```go
if method == "item/commandExecution/requestApproval" && serverRequestID != nil {
    d.log.Debug("auto-accepting approval request", "threadID", threadID)
    // ...
    srv.respondToServerRequest(*serverRequestID, map[string]string{"decision": "accept"})
    return
}
```

The `approvalPolicy` is set to `"on-failure"` when yolo is false, and `"never"` when yolo is true. With `"on-failure"`, Codex sends approval requests for commands that fail or for potentially dangerous operations.

## Implementation Plan

### 1. Shared Driver Types

Same as Claude driver — uses the shared `EventTypePermissionRequest`, `PermissionResponder`, and `CapPermissionRequest` from `internal/worker/driver/`.

### 2. Codex Driver Changes (`internal/worker/driver/codex/`)

**codex.go — add pending permissions map to Driver (not session):**

Since approval requests arrive via the shared app-server's `dispatchNotification` and are keyed by a driver-generated request ID (not the JSON-RPC server request ID), the pending map lives on the Driver:

```go
type permissionResponse struct {
    Allow  bool
    Reason string
}

type Driver struct {
    // ... existing fields ...
    pendingPermissions   map[string]pendingPermission
    pendingPermissionsMu sync.Mutex
}

type pendingPermission struct {
    serverRequestID int64
    ch              chan permissionResponse
}
```

**codex.go — change `dispatchNotification` to emit event instead of auto-accepting:**

```go
if method == "item/commandExecution/requestApproval" && serverRequestID != nil {
    requestID := uuid.New().String()
    ch := make(chan permissionResponse, 1)

    d.pendingPermissionsMu.Lock()
    d.pendingPermissions[requestID] = pendingPermission{
        serverRequestID: *serverRequestID,
        ch:              ch,
    }
    d.pendingPermissionsMu.Unlock()

    // Extract tool info from params for the event.
    var p struct {
        ThreadID string `json:"threadId"`
        Item     struct {
            Command string `json:"command"`
        } `json:"item"`
    }
    _ = json.Unmarshal(params, &p)

    // Find the session to emit the event.
    d.mu.Lock()
    sess, ok := d.sessions[threadID]
    d.mu.Unlock()

    if ok {
        sess.emit(driver.Event{
            Type:                driver.EventTypePermissionRequest,
            Timestamp:           currentTime(),
            Agent:               agent,
            ToolName:            "command_execution",
            Text:                p.Item.Command,
            PermissionRequestID: requestID,
        })
    }

    // Wait for response in a goroutine to avoid blocking the readLoop.
    go func() {
        resp := <-ch

        d.pendingPermissionsMu.Lock()
        delete(d.pendingPermissions, requestID)
        d.pendingPermissionsMu.Unlock()

        d.mu.Lock()
        srv := d.server
        d.mu.Unlock()

        if srv != nil {
            decision := "deny"
            if resp.Allow {
                decision = "accept"
            }
            srv.respondToServerRequest(*serverRequestID, map[string]string{"decision": decision})
        }
    }()

    return
}
```

**codex.go — implement PermissionResponder on codexSession:**

The session delegates to the driver's pending map:

```go
func (s *codexSession) RespondToPermission(_ context.Context, requestID string, allow bool, reason string) error {
    s.driver.pendingPermissionsMu.Lock()
    pp, ok := s.driver.pendingPermissions[requestID]
    s.driver.pendingPermissionsMu.Unlock()

    if !ok {
        return fmt.Errorf("no pending permission request: %s", requestID)
    }

    pp.ch <- permissionResponse{Allow: allow, Reason: reason}
    return nil
}
```

**codex.go — advertise capability:**

Add `driver.CapPermissionRequest` to `Capabilities()`.

**codex.go — change approval policy:**

Change the non-yolo approval policy from `"on-failure"` to `"unless-allow-listed"` (or whatever Codex's strictest non-yolo policy is) so that more tool uses trigger approval requests. If `"on-failure"` is already the right policy, keep it.

### 3. Initialization

Add to `NewDriver`:

```go
pendingPermissions: make(map[string]pendingPermission),
```

### 4. Cleanup

When sessions are removed (server dies, explicit stop), drain pending permissions for that session. When the app-server process dies, all pending permission channels should be closed.

## Files Changed

| File | Change |
|------|--------|
| `internal/worker/driver/codex/codex.go` | Replace auto-accept with permission event + wait, add `RespondToPermission`, add pending map, advertise capability |
| `internal/worker/driver/codex/app_server.go` | No changes needed — `respondToServerRequest` already exists |
