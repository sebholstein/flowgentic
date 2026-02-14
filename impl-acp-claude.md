# ACP Claude Adapter Enhancement Plan

Comparison baseline: [zed-industries/claude-code-acp](https://github.com/zed-industries/claude-code-acp)

Excluded: images, resources, listSessions, loadSession, forkSession, resumeSession

---

## 1. Advertise Capabilities in `Initialize`

**File:** `internal/worker/driver/claude/acp/adapter.go`

```go
func (a *Adapter) Initialize(_ context.Context, _ acpsdk.InitializeRequest) (acpsdk.InitializeResponse, error) {
    return acpsdk.InitializeResponse{
        ProtocolVersion: acpsdk.ProtocolVersion(acpsdk.ProtocolVersionNumber),
        AgentInfo: &acpsdk.Implementation{
            Name:    "claude-code",
            Version: "1.0.0",
        },
        AgentCapabilities: &acpsdk.AgentCapabilities{
            LoadSession: false,
            PromptCapabilities: &acpsdk.PromptCapabilities{
                Image:           false,
                Audio:           false,
                EmbeddedContext: false,
            },
            McpCapabilities: &acpsdk.McpCapabilities{
                Http: false,
                Sse:  false,
            },
        },
    }, nil
}
```

**Effort:** ~15 min  
**Priority:** High

---

## 2. Plan Updates — Convert `TodoWrite` to `plan` Session Updates

**File:** `internal/worker/driver/claude/acp/adapter.go`

Add detection of `TodoWrite` tool calls and emit `plan` updates:

```go
// In normalizeAssistantMessage, when processing ToolUseBlock:
if b.Name == "TodoWrite" {
    if todos, ok := b.Input["todos"].([]any); ok {
        entries := make([]acpsdk.PlanEntry, 0, len(todos))
        for _, t := range todos {
            if tm, ok := t.(map[string]any); ok {
                content, _ := tm["content"].(string)
                status := "pending"
                if s, ok := tm["status"].(string); ok {
                    status = s
                }
                entries = append(entries, acpsdk.PlanEntry{
                    Content:  content,
                    Status:   acpsdk.PlanEntryStatus(status),
                    Priority: acpsdk.PlanPriorityMedium,
                })
            }
        }
        a.sendUpdate(ctx, sessionID, acpsdk.SessionUpdate{
            SessionUpdate: acpsdk.SessionUpdatePlan,
            Entries:       entries,
        })
    }
    continue // don't emit tool_call for TodoWrite
}
```

**Effort:** ~1 hour  
**Priority:** High

---

## 3. Rich Tool Call Details

**File:** `internal/worker/driver/claude/acp/adapter.go`

Create a helper to compute `ToolKind`, `Locations`, and `Content`:

```go
type toolInfo struct {
    title     string
    kind      acpsdk.ToolKind
    locations []acpsdk.ToolCallLocation
    content   []acpsdk.ToolCallContent
}

func toolInfoFromToolUse(name string, input map[string]any) toolInfo {
    switch name {
    case "Read":
        path, _ := input["file_path"].(string)
        return toolInfo{
            title: fmt.Sprintf("Read %s", path),
            kind:  acpsdk.ToolKindRead,
            locations: []acpsdk.ToolCallLocation{{Path: path}},
        }
    case "Write":
        path, _ := input["file_path"].(string)
        content, _ := input["content"].(string)
        return toolInfo{
            title: fmt.Sprintf("Write %s", path),
            kind:  acpsdk.ToolKindEdit,
            locations: []acpsdk.ToolCallLocation{{Path: path}},
            content: []acpsdk.ToolCallContent{{
                Type:    "diff",
                Path:    path,
                OldText: nil,
                NewText: content,
            }},
        }
    case "Edit":
        path, _ := input["file_path"].(string)
        oldStr, _ := input["old_string"].(string)
        newStr, _ := input["new_string"].(string)
        return toolInfo{
            title: fmt.Sprintf("Edit %s", path),
            kind:  acpsdk.ToolKindEdit,
            locations: []acpsdk.ToolCallLocation{{Path: path}},
            content: []acpsdk.ToolCallContent{{
                Type:    "diff",
                Path:    path,
                OldText: &oldStr,
                NewText: newStr,
            }},
        }
    case "Bash":
        cmd, _ := input["command"].(string)
        desc, _ := input["description"].(string)
        title := fmt.Sprintf("`%s`", cmd)
        if desc != "" {
            title = desc
        }
        return toolInfo{
            title: title,
            kind:  acpsdk.ToolKindExecute,
        }
    case "Grep":
        pattern, _ := input["pattern"].(string)
        return toolInfo{
            title: fmt.Sprintf("grep \"%s\"", pattern),
            kind:  acpsdk.ToolKindSearch,
        }
    case "Glob":
        pattern, _ := input["pattern"].(string)
        return toolInfo{
            title: fmt.Sprintf("Glob %s", pattern),
            kind:  acpsdk.ToolKindSearch,
        }
    case "WebFetch":
        url, _ := input["url"].(string)
        return toolInfo{
            title: fmt.Sprintf("Fetch %s", url),
            kind:  acpsdk.ToolKindFetch,
        }
    case "WebSearch":
        query, _ := input["query"].(string)
        return toolInfo{
            title: fmt.Sprintf("Search \"%s\"", query),
            kind:  acpsdk.ToolKindFetch,
        }
    case "Task":
        desc, _ := input["description"].(string)
        return toolInfo{
            title: desc,
            kind:  acpsdk.ToolKindThink,
        }
    default:
        return toolInfo{title: name, kind: acpsdk.ToolKindOther}
    }
}
```

Update `StartToolCall` to include these fields:

```go
info := toolInfoFromToolUse(b.Name, b.Input)
a.sendUpdate(ctx, sessionID, acpsdk.StartToolCall(
    acpsdk.ToolCallId(id),
    info.title,
    acpsdk.WithStartKind(info.kind),
    acpsdk.WithStartStatus(acpsdk.ToolCallStatusPending),
    acpsdk.WithStartRawInput(b.Input),
    acpsdk.WithStartLocations(info.locations),
    acpsdk.WithStartContent(info.content),
))
```

**Effort:** ~2 hours  
**Priority:** Medium

---

## 4. Slash Commands Advertisement

**File:** `internal/worker/driver/claude/acp/adapter.go`

Add method to query and send available commands after session creation:

```go
var unsupportedCommands = map[string]bool{
    "cost":             true,
    "login":            true,
    "logout":           true,
    "keybindings-help": true,
    "release-notes":    true,
    "todos":            true,
}

func (a *Adapter) sendAvailableCommands(ctx context.Context, sessionID acpsdk.SessionId) {
    if a.client == nil {
        return
    }
    commands, err := a.client.SupportedCommands(ctx)
    if err != nil {
        a.log.Debug("failed to get supported commands", "error", err)
        return
    }
    var avail []acpsdk.AvailableCommand
    for _, c := range commands {
        if unsupportedCommands[c.Name] {
            continue
        }
        name := c.Name
        if strings.HasSuffix(name, " (MCP)") {
            name = "mcp:" + strings.TrimSuffix(name, " (MCP)")
        }
        cmd := acpsdk.AvailableCommand{
            Name:        name,
            Description: c.Description,
        }
        if c.ArgumentHint != "" {
            cmd.Input = &acpsdk.AvailableCommandInput{Hint: c.ArgumentHint}
        }
        avail = append(avail, cmd)
    }
    a.sendUpdate(ctx, sessionID, acpsdk.SessionUpdate{
        SessionUpdate:     acpsdk.SessionUpdateAvailableCommands,
        AvailableCommands: avail,
    })
}
```

Call after `NewSession` returns (async to avoid blocking):

```go
func (a *Adapter) NewSession(...) (acpsdk.NewSessionResponse, error) {
    // ... existing code ...
    go a.sendAvailableCommands(context.Background(), acpsdk.SessionId(a.sessionID))
    return resp, nil
}
```

**Effort:** ~1.5 hours  
**Priority:** Medium

---

## 5. Mode Change Notifications

**File:** `internal/worker/driver/claude/acp/adapter.go`

When plan mode is entered/exited, notify the client:

```go
func (a *Adapter) notifyModeChange(ctx context.Context, sessionID acpsdk.SessionId, mode string) {
    a.sendUpdate(ctx, sessionID, acpsdk.SessionUpdate{
        SessionUpdate: acpsdk.SessionUpdateCurrentMode,
        CurrentModeId: mode,
    })
}
```

Detect `EnterPlanMode` / `ExitPlanMode` in tool result handling and emit the notification.

**Effort:** ~30 min  
**Priority:** Medium

---

## 6. Permission Result with Mode Updates (ExitPlanMode)

**File:** `internal/worker/driver/claude/acp/adapter.go`

Enhance `handlePermission` for `ExitPlanMode` tool to offer mode choices:

```go
func (a *Adapter) handlePermission(ctx context.Context, sessionID acpsdk.SessionId, toolName string, input map[string]any) (claudecode.PermissionResult, error) {
    if a.conn == nil {
        return claudecode.NewPermissionResultDeny("no ACP connection"), nil
    }

    if toolName == "ExitPlanMode" {
        return a.handleExitPlanMode(ctx, sessionID, input)
    }

    // ... existing permission handling ...
}

func (a *Adapter) handleExitPlanMode(ctx context.Context, sessionID acpsdk.SessionId, input map[string]any) (claudecode.PermissionResult, error) {
    resp, err := a.conn.RequestPermission(ctx, acpsdk.RequestPermissionRequest{
        SessionId: sessionID,
        ToolCall: acpsdk.RequestPermissionToolCall{
            ToolCallId: acpsdk.ToolCallId("exit-plan-mode"),
            Title:      acpsdk.Ptr("Ready to code?"),
            RawInput:   input,
        },
        Options: []acpsdk.PermissionOption{
            {OptionId: "acceptEdits", Name: "Yes, auto-accept edits", Kind: acpsdk.PermissionOptionKindAllowAlways},
            {OptionId: "default", Name: "Yes, manually approve edits", Kind: acpsdk.PermissionOptionKindAllowOnce},
            {OptionId: "plan", Name: "No, keep planning", Kind: acpsdk.PermissionOptionKindRejectOnce},
        },
    })
    if err != nil {
        return claudecode.NewPermissionResultDeny("permission request failed"), nil
    }

    if resp.Outcome.Cancelled != nil {
        return claudecode.NewPermissionResultDeny("cancelled"), nil
    }
    if resp.Outcome.Selected == nil {
        return claudecode.NewPermissionResultDeny("no selection"), nil
    }

    switch resp.Outcome.Selected.OptionId {
    case "acceptEdits":
        a.notifyModeChange(ctx, sessionID, "acceptEdits")
        return claudecode.NewPermissionResultAllowWithSuggestions([]claudecode.PermissionSuggestion{
            {Type: "setMode", Mode: "acceptEdits", Destination: "session"},
        }), nil
    case "default":
        a.notifyModeChange(ctx, sessionID, "default")
        return claudecode.NewPermissionResultAllowWithSuggestions([]claudecode.PermissionSuggestion{
            {Type: "setMode", Mode: "default", Destination: "session"},
        }), nil
    case "plan":
        return claudecode.NewPermissionResultDeny("user chose to keep planning"), nil
    default:
        return claudecode.NewPermissionResultDeny("unknown option"), nil
    }
}
```

**Effort:** ~1 hour  
**Priority:** Medium

---

## 7. Settings Integration (Basic)

**File:** `internal/worker/driver/claude/acp/settings.go` (new file)

```go
package acp

import (
    "encoding/json"
    "os"
    "path/filepath"
)

type PermissionSettings struct {
    Allow              []string `json:"allow"`
    Deny               []string `json:"deny"`
    Ask                []string `json:"ask"`
    AdditionalDirs     []string `json:"additionalDirectories"`
}

type ClaudeSettings struct {
    Permissions *PermissionSettings `json:"permissions"`
    Env         map[string]string   `json:"env"`
    Model       string              `json:"model"`
}

func LoadSettings(cwd string) (*ClaudeSettings, error) {
    paths := []string{
        filepath.Join(cwd, ".claude", "settings.local.json"),
        filepath.Join(cwd, ".claude", "settings.json"),
    }
    merged := &ClaudeSettings{
        Env: make(map[string]string),
    }
    for _, p := range paths {
        b, err := os.ReadFile(p)
        if err != nil {
            continue
        }
        var s ClaudeSettings
        if err := json.Unmarshal(b, &s); err != nil {
            continue
        }
        if s.Permissions != nil {
            if merged.Permissions == nil {
                merged.Permissions = &PermissionSettings{}
            }
            merged.Permissions.Allow = append(merged.Permissions.Allow, s.Permissions.Allow...)
            merged.Permissions.Deny = append(merged.Permissions.Deny, s.Permissions.Deny...)
            merged.Permissions.Ask = append(merged.Permissions.Ask, s.Permissions.Ask...)
            merged.Permissions.AdditionalDirs = append(merged.Permissions.AdditionalDirs, s.Permissions.AdditionalDirs...)
        }
        for k, v := range s.Env {
            merged.Env[k] = v
        }
        if s.Model != "" {
            merged.Model = s.Model
        }
    }
    return merged, nil
}
```

Use in `NewSession`:

```go
func (a *Adapter) NewSession(_ context.Context, req acpsdk.NewSessionRequest) (acpsdk.NewSessionResponse, error) {
    a.cwd = req.Cwd
    a.sessionID = uuid.New().String()

    // Load settings
    settings, err := LoadSettings(a.cwd)
    if err == nil {
        if settings.Model != "" && a.model == "" {
            a.model = settings.Model
        }
        for k, v := range settings.Env {
            if a.envVars == nil {
                a.envVars = make(map[string]string)
            }
            a.envVars[k] = v
        }
    }

    // ... existing _meta parsing ...
}
```

**Effort:** ~2 hours  
**Priority:** Low

---

## 8. Model Selection from Settings

**File:** `internal/worker/driver/claude/acp/adapter.go`

In `NewSession`, match settings model against available models:

```go
func (a *Adapter) NewSession(_ context.Context, req acpsdk.NewSessionRequest) (acpsdk.NewSessionResponse, error) {
    // ... existing code ...

    settings, _ := LoadSettings(a.cwd)
    if settings != nil && settings.Model != "" && a.model == "" {
        // Try to match against available models
        if a.modelProvider != nil {
            state, err := a.modelProvider.SessionModelState(context.Background())
            if err == nil && state != nil {
                for _, m := range state.AvailableModels {
                    if strings.Contains(string(m.ModelId), settings.Model) ||
                       strings.EqualFold(m.Name, settings.Model) {
                        a.model = string(m.ModelId)
                        break
                    }
                }
            }
        }
        if a.model == "" {
            a.model = settings.Model // use as-is if no match
        }
    }

    // ... rest of NewSession ...
}
```

**Effort:** ~30 min  
**Priority:** Low

---

## Summary

| # | Feature | Effort | Priority |
|---|---------|--------|----------|
| 1 | Advertise Capabilities | 15 min | High |
| 2 | Plan Updates (TodoWrite) | 1 hr | High |
| 3 | Rich Tool Call Details | 2 hr | Medium |
| 4 | Slash Commands | 1.5 hr | Medium |
| 5 | Mode Change Notifications | 30 min | Medium |
| 6 | ExitPlanMode Permission UI | 1 hr | Medium |
| 7 | Settings Integration | 2 hr | Low |
| 8 | Model from Settings | 30 min | Low |

**Total:** ~8.5 hours

---

## Implementation Order

1. **Phase 1 (High Priority):** #1, #2
2. **Phase 2 (Medium Priority):** #3, #4, #5, #6
3. **Phase 3 (Low Priority):** #7, #8
