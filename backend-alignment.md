# Full ACP Alignment: Entities + Driver

## Context

The v2 driver layer wraps ACP but only supports single-prompt sessions. The codebase uses "AgentRun" instead of ACP's "Session". And the entity model is flat (`Thread → AgentRun`) with no concept of tasks, subtasks, or plans — all orchestration is ephemeral in the agent's context window.

This plan aligns everything: entity hierarchy, naming, and driver API surface.

## Entity Hierarchy

```
Thread (sidebar)
│
├── agent                         ← the thread's agent
├── plan: string (optional)       ← markdown plan (set by agent in plan mode)
│
├── Session(s)                    ← direct work (simple mode)
│                                    OR planning phase (orchestrated mode)
│
├── Task (optional, created by agent in plan mode)
│   ├── subtasks: []string        ← checklist items, not entities
│   ├── memory: string            ← markdown, persisted across sessions
│   ├── status                    ← pending/running/done/failed
│   └── Session(s)                ← agent attempts (ralph wiggum:
│                                    restart fresh when context fills,
│                                    gets task desc + plan + memory +
│                                    subtask state)
├── Task
│   └── ...
└── ...
```

The thread's agent can either:
1. **Simple mode** — work directly, sessions on thread, no tasks
2. **Orchestrated mode** — plan first (creates plan + tasks), then tasks get their own sessions

No separate overseer entity. The thread's agent IS the orchestrator when in plan mode.

### Current DB Schema

```
projects ──< threads ──< agent_runs
                │
                └── mode: "single_agent" | "orchestrated"
```

No tasks, subtasks, plan, or memory storage. Everything is ephemeral.

### Target DB Schema

```
projects ──< threads ──< sessions     (renamed from agent_runs)
                │
                ├── plan (text)
                │
                └──< tasks
                      ├── subtasks (json text)
                      ├── memory (text)
                      ├── status
                      └──< sessions
```

A `session` belongs to either a thread directly OR a task:
- `sessions.thread_id` (always set)
- `sessions.task_id` (nullable — null = thread-level session, set = task session)

---

## Phase 1: Rename AgentRun → Session (subagent)

Mechanical rename across the codebase.

| Current | New |
|---|---|
| `controlplane/agentrun/` | `controlplane/session/` |
| `AgentRun` struct | `Session` |
| `AgentRunService` | `SessionService` |
| `AgentRunManager` (worker) | `SessionManager` |
| `agent_run_id` (proto/SQL) | `session_id` |
| `AgentRunSnapshot` | `SessionSnapshot` |
| `AgentRunStatus` (proto enum) | `SessionStatus` |
| `AgentRunConfig` (proto msg) | `SessionConfig` |
| `AgentRunState` (proto msg) | `SessionState` |
| `AgentRunCreator` interface | `SessionCreator` |
| `agent_runs` SQL table | `sessions` (migration) |
| `NewAgentRun` RPC | `NewSession` |
| `ListAgentRuns` RPC | `ListSessions` |
| `GetAgentRun` RPC | `GetSession` |

No collision: `controlplane/session.Session` (persistent record) vs `driver/v2.Session` (live ACP connection) — different packages.

**Scope:** Proto definitions, generated code (`make proto`), Go source, SQL migration, server wiring, tests. Frontend auto-generated from proto.

---

## Phase 2: Add Task Entity

### New SQL migration

```sql
CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    thread_id TEXT NOT NULL REFERENCES threads(id),
    description TEXT NOT NULL DEFAULT '',
    subtasks TEXT NOT NULL DEFAULT '[]',   -- JSON array of strings
    memory TEXT NOT NULL DEFAULT '',        -- markdown, persisted across sessions
    status TEXT NOT NULL DEFAULT 'pending', -- pending, running, done, failed
    sort_index INTEGER NOT NULL DEFAULT 0,
    created_at TEXT,
    updated_at TEXT
);
CREATE INDEX idx_tasks_thread_id ON tasks(thread_id);
```

### Update sessions table

Add nullable `task_id` to sessions:

```sql
ALTER TABLE sessions ADD COLUMN task_id TEXT REFERENCES tasks(id) DEFAULT NULL;
```

- `task_id = NULL` → overseer session (thread-level)
- `task_id = <id>` → task worker session

### Add plan to threads

```sql
ALTER TABLE threads ADD COLUMN plan TEXT NOT NULL DEFAULT '';
```

### New controlplane package: `controlplane/task/`

Following the feature pattern:
- `task_service.go` — business logic (CRUD for tasks, subtask management)
- `task.go` — `StartDeps` + `Start()`
- `task_service_handler.go` — Connect RPC handler
- `store/` — sqlc queries + models

### Proto: `controlplane/v1/task_service.proto`

```proto
service TaskService {
    rpc CreateTask(CreateTaskRequest) returns (CreateTaskResponse);
    rpc GetTask(GetTaskRequest) returns (GetTaskResponse);
    rpc ListTasks(ListTasksRequest) returns (ListTasksResponse);
    rpc UpdateTask(UpdateTaskRequest) returns (UpdateTaskResponse);
    rpc DeleteTask(DeleteTaskRequest) returns (DeleteTaskResponse);
}

message TaskConfig {
    string id = 1;
    string thread_id = 2;
    string description = 3;
    repeated string subtasks = 4;
    string memory = 5;
    string status = 6;
    int32 sort_index = 7;
    string created_at = 8;
    string updated_at = 9;
}
```

### Update thread proto

Add `plan` field to `ThreadConfig` and `UpdateThreadRequest`.

---

## Phase 3: Session Interface — Mirror ACP API

The `v2.Session` interface becomes a 1:1 mapping of `ClientSideConnection` methods.

### 1. Redesign Session interface (`v2/session.go`)

```go
type Session interface {
    // Lifecycle (our additions)
    Info() SessionInfo
    Stop(ctx context.Context) error
    Wait(ctx context.Context) error

    // ACP agent methods — direct SDK types
    Prompt(ctx context.Context, req acp.PromptRequest) (acp.PromptResponse, error)
    Cancel(ctx context.Context) error
    SetSessionMode(ctx context.Context, req acp.SetSessionModeRequest) (acp.SetSessionModeResponse, error)
    SetSessionModel(ctx context.Context, req acp.SetSessionModelRequest) (acp.SetSessionModelResponse, error)

    // Client-side bridge
    RespondToPermission(ctx context.Context, requestID string, allow bool, reason string) error
}
```

### 2. Multi-turn via `promptCh`

```go
type acpSession struct {
    info     SessionInfo
    conn     *acp.ClientSideConnection
    client   *flowgenticClient
    cancel   context.CancelFunc
    done     chan struct{}
    promptCh chan promptRequest
    cancelCh chan struct{}
    mu       sync.Mutex
}
```

`Prompt()` sends to `promptCh`, blocks on response. `Cancel()` signals `cancelCh`.

### 3. Refactor `runSession`

```
Initialize
  → if authMethods: Authenticate
  → if opts.SessionID: LoadSession, else: NewSession
  → initial Prompt (from opts.Prompt)
  → status = Idle
  → loop:
      select promptCh → Running, conn.Prompt(), Idle
      select cancelCh → conn.Cancel()
      select ctx.Done() → exit
```

### 4. Extend `SessionInfo`

Add `Modes *acp.SessionModeState` and `Models *acp.SessionModelState` from NewSession/LoadSession responses.

---

## Phase 4: Client-Side — Pluggable Handlers

Agents handle their own FS/terminal. Handlers are pluggable hooks (default = unsupported).

### Handler interfaces (`v2/handlers.go`)

```go
type ClientHandlers struct {
    FS       FileSystemHandler    // nil = unsupported
    Terminal TerminalHandler      // nil = unsupported
}

type FileSystemHandler interface {
    ReadTextFile(ctx context.Context, req acp.ReadTextFileRequest) (acp.ReadTextFileResponse, error)
    WriteTextFile(ctx context.Context, req acp.WriteTextFileRequest) (acp.WriteTextFileResponse, error)
}

type TerminalHandler interface {
    CreateTerminal(ctx context.Context, req acp.CreateTerminalRequest) (acp.CreateTerminalResponse, error)
    KillTerminalCommand(ctx context.Context, req acp.KillTerminalCommandRequest) (acp.KillTerminalCommandResponse, error)
    TerminalOutput(ctx context.Context, req acp.TerminalOutputRequest) (acp.TerminalOutputResponse, error)
    ReleaseTerminal(ctx context.Context, req acp.ReleaseTerminalRequest) (acp.ReleaseTerminalResponse, error)
    WaitForTerminalExit(ctx context.Context, req acp.WaitForTerminalExitRequest) (acp.WaitForTerminalExitResponse, error)
}
```

Wire through `LaunchOpts.Handlers *ClientHandlers`.

---

## Phase 5: Consumer Layer

### 1. `SessionManager` (renamed from `AgentRunManager`)

Add methods:
```go
func (m *SessionManager) Prompt(ctx, sessionID, blocks) (acp.PromptResponse, error)
func (m *SessionManager) Cancel(ctx, sessionID) error
func (m *SessionManager) SetSessionMode(ctx, sessionID, req) (acp.SetSessionModeResponse, error)
func (m *SessionManager) SetSessionModel(ctx, sessionID, req) (acp.SetSessionModelResponse, error)
```

### 2. Worker proto RPCs

- `PromptSession(PromptSessionRequest) → PromptSessionResponse`
- `CancelSession(CancelSessionRequest) → CancelSessionResponse`

### 3. Capabilities

```go
CapFileSystem  Capability = "file_system"
CapTerminal    Capability = "terminal"
```

---

## Files to Modify

### Phase 1 (subagent — rename)
- `controlplane/agentrun/` → `controlplane/session/` (all files)
- `controlplane/v1/agent_run_service.proto` → `session_service.proto`
- `worker/v1/worker_service.proto` (enums, messages, RPCs)
- `workload/agent_run_manager.go` → `workload/session_manager.go`
- `controlplane/server/`, `worker/server/` (imports, wiring)
- `controlplane/thread/` (`AgentRunCreator` → `SessionCreator`)
- New SQL migration: rename `agent_runs` → `sessions`
- All test files
- `make proto` to regenerate

### Phase 2 (task entity)
- New migration: `tasks` table, `sessions.task_id`, `threads.plan`
- **New** `controlplane/task/` package (service, handler, store)
- **New** `controlplane/v1/task_service.proto`
- Update `controlplane/v1/thread_service.proto` (add plan field)
- `controlplane/server/features.go` (register task feature)
- `controlplane/server/server.go` (wire + reflector)

### Phase 3–5 (driver + consumer)
- `v2/session.go` — redesign interface
- `v2/subprocess.go` — multi-turn loop, LoadSession, Authenticate, Cancel
- `v2/client.go` — add ClientHandlers
- `v2/handlers.go` — **new** handler interfaces
- `v2/driver.go` — add Handlers to LaunchOpts
- `capabilities.go` — add CapFileSystem, CapTerminal
- `workload/session_manager.go` — add Prompt/Cancel/SetSessionModel
- `worker/v1/worker_service.proto` — add PromptSession, CancelSession
- `workload/worker_service_handler.go` — wire new RPCs

## Verification

1. `make proto`
2. `make build`
3. `go test ./internal/...`
4. New tests:
   - Task CRUD (create, list, update subtasks/memory/status)
   - Session with task_id vs without (overseer vs worker)
   - Multi-turn prompt dispatch
   - LoadSession path
   - Cancel during active prompt
   - Pluggable handlers (nil = error, set = delegates)
5. Manual test with real agent for multi-turn
