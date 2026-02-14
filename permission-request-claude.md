# Permission Request — Claude Driver

## Status: Fully Supported

The Claude Go SDK (`claude-agent-sdk-go` v0.6.12) provides first-class support for interactive permission flows via the `WithCanUseTool` callback.

## SDK API Surface

```go
// Callback signature
type CanUseToolCallback func(
    ctx context.Context,
    toolName string,
    input map[string]any,
    permCtx ToolPermissionContext,
) (PermissionResult, error)

// Results
NewPermissionResultAllow() PermissionResultAllow
NewPermissionResultDeny(message string) PermissionResultDeny

// Option to register the callback
WithCanUseTool(callback CanUseToolCallback) Option
```

**Callback rules:**
- Only invoked for tools that prompt the user AND when using `PermissionModeDefault`.
- Read-only tools (Read, Glob, Grep) are auto-approved and do NOT trigger the callback.
- Write tools (Write, Edit, Bash) trigger the callback only in `PermissionModeDefault`.
- `PermissionModeBypassPermissions` (yolo mode) bypasses all checks — no callback invoked.
- Callbacks must be thread-safe.
- If no callback is set and not in bypass mode, tool requests are denied.

## Implementation Plan

### 1. Add Shared Driver Types

In `internal/worker/driver/`:

```go
// event.go — new event type
EventTypePermissionRequest EventType = "permission_request"

// event.go — new fields on Event
type Event struct {
    // ... existing fields ...
    PermissionRequestID string `json:"permission_request_id,omitempty"`
}

// capabilities.go — new capability
CapPermissionRequest Capability = "permission_request"

// session.go — new interface
type PermissionResponder interface {
    RespondToPermission(ctx context.Context, requestID string, allow bool, reason string) error
}
```

### 2. Claude Driver Changes (`internal/worker/driver/claude/`)

**claude.go — add pending permissions map to session:**

```go
type permissionResponse struct {
    Allow  bool
    Reason string
}

type claudeSession struct {
    // ... existing fields ...
    pendingPermissions   map[string]chan permissionResponse
    pendingPermissionsMu sync.Mutex
}
```

**claude.go — register WithCanUseTool in `buildSDKOptions`:**

When `opts.Yolo` is false, register a `WithCanUseTool` callback that:

1. Generates a unique `requestID` (UUID).
2. Creates a `chan permissionResponse` and stores it in `pendingPermissions[requestID]`.
3. Serializes `input` to `json.RawMessage` for the event's `ToolInput` field.
4. Emits `EventTypePermissionRequest` with tool name, tool input, and request ID.
5. Blocks on the channel (with `ctx.Done()` select for cancellation).
6. Returns `NewPermissionResultAllow()` or `NewPermissionResultDeny(reason)`.

```go
func (s *claudeSession) canUseTool(
    ctx context.Context,
    toolName string,
    input map[string]any,
    permCtx claudecode.ToolPermissionContext,
) (claudecode.PermissionResult, error) {
    requestID := uuid.New().String()
    ch := make(chan permissionResponse, 1)

    s.pendingPermissionsMu.Lock()
    s.pendingPermissions[requestID] = ch
    s.pendingPermissionsMu.Unlock()

    defer func() {
        s.pendingPermissionsMu.Lock()
        delete(s.pendingPermissions, requestID)
        s.pendingPermissionsMu.Unlock()
    }()

    inputJSON, _ := json.Marshal(input)
    s.emit(driver.Event{
        Type:                driver.EventTypePermissionRequest,
        Timestamp:           time.Now(),
        Agent:               agent,
        ToolName:            toolName,
        ToolInput:           inputJSON,
        PermissionRequestID: requestID,
    })

    select {
    case resp := <-ch:
        if resp.Allow {
            return claudecode.NewPermissionResultAllow(), nil
        }
        return claudecode.NewPermissionResultDeny(resp.Reason), nil
    case <-ctx.Done():
        return claudecode.NewPermissionResultDeny("session cancelled"), ctx.Err()
    }
}
```

**claude.go — implement PermissionResponder:**

```go
func (s *claudeSession) RespondToPermission(_ context.Context, requestID string, allow bool, reason string) error {
    s.pendingPermissionsMu.Lock()
    ch, ok := s.pendingPermissions[requestID]
    s.pendingPermissionsMu.Unlock()

    if !ok {
        return fmt.Errorf("no pending permission request: %s", requestID)
    }

    ch <- permissionResponse{Allow: allow, Reason: reason}
    return nil
}
```

**claude.go — advertise capability:**

Add `driver.CapPermissionRequest` to the `Capabilities()` return value.

**buildSDKOptions — wire the callback:**

The callback needs access to the session, so it must be set up in `launchSDK` rather than in the standalone `buildSDKOptions` function. After building the base options, append:

```go
if !opts.Yolo {
    sdkOpts = append(sdkOpts, claudecode.WithCanUseTool(sess.canUseTool))
}
```

### 3. Initialization

Initialize the map in `Launch`:

```go
sess := &claudeSession{
    // ... existing fields ...
    pendingPermissions: make(map[string]chan permissionResponse),
}
```

### 4. Cleanup

When the session stops or errors, drain/close any pending permission channels so blocked goroutines unblock:

```go
func (s *claudeSession) closePendingPermissions() {
    s.pendingPermissionsMu.Lock()
    defer s.pendingPermissionsMu.Unlock()
    for id, ch := range s.pendingPermissions {
        close(ch)
        delete(s.pendingPermissions, id)
    }
}
```

Call `closePendingPermissions()` in `Stop()` and in the deferred cleanup of `launchSDK`.

## Files Changed

| File | Change |
|------|--------|
| `internal/worker/driver/event.go` | Add `EventTypePermissionRequest`, `PermissionRequestID` field |
| `internal/worker/driver/session.go` | Add `PermissionResponder` interface |
| `internal/worker/driver/capabilities.go` | Add `CapPermissionRequest` |
| `internal/worker/driver/claude/claude.go` | Add pending map, `canUseTool` callback, `RespondToPermission`, capability, cleanup |
