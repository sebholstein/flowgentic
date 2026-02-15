# Thread & Planning UX Flow Design

## Context

The goal: a **unified thread model** that covers everything from "change button colors to blue" to "build an authentication system with competing plans from multiple AI agents" — without forcing the user to pick modes upfront.

A thread is the fundamental unit of work. It starts as a chat and can evolve into an orchestrated, multi-task execution environment — naturally, based on what the work requires.

---

## Design Decisions

- **No upfront mode selection.** Every thread starts as a chat. Planning and task decomposition emerge organically.
- **Planning is a behavior, not a mode.** Any session can enter planning mode (agent analyzes, doesn't write code, proposes a plan). This is a constraint on agent behavior, not a thread type.
- **Sessions as tabs.** Every thread supports multiple session tabs via a [+] button. Each session is an independent agent chat on the same codebase.
- **Plan approval triggers evolution.** When a plan is approved:
  - If the plan is a single unit of work → context resets, agent executes in the same thread.
  - If the plan has multiple tasks → tasks are created, the thread becomes an orchestrator, context resets with the plan as the new foundation.
- **No explicit "overseer" persona.** In a thread with tasks, the thread-level chat session is automatically coordination-aware (has plan context, task statuses, learnings). The user talks to the thread, not to "an overseer."
- **Additional planners are optional.** The planning agent can optionally spawn sessions from other agents/models for competing plans. This isn't a mode — it's an action.

---

## Thread Lifecycle

### 1. Chat (Default State)

Every thread starts here. Like Claude Code, Codex, or any coding agent — just a session with an LLM.

```
Thread: Fix button colors
├── [Session: Claude Code] [+]
├────────────────────────────────────────┤
│                                        │
│  You: Change all button colors to blue │
│                                        │
│  Agent: I'll update the button styles. │
│  ┌─ Edit: src/components/Button.tsx ──┐│
│  │ - bg-emerald-500                   ││
│  │ + bg-blue-500                      ││
│  └────────────────────────────────────┘│
│  Agent: Done. Changed 4 files.         │
│                                        │
│  [Input box]                           │
└────────────────────────────────────────┘
```

**Multiple sessions:** User can open additional session tabs with the [+] button — different agents, different models, same working directory (or git worktree). Each session is independent, no coordination.

```
Thread: Make app responsive
├── [Session: Claude Code] [Session: Codex] [+]
├────────────────────────────────────────────────┤
│ (active session's chat)                        │
└────────────────────────────────────────────────┘
```

**No tasks, no plan, no orchestration.** Just sessions.

---

### 2. Planning (Transitional)

Planning can start in two ways:
- **Agent-initiated:** Agent detects complexity and suggests: "This is a larger change — want me to create a plan first?"
- **User-initiated:** User explicitly asks the agent to plan (e.g., "Plan this out before doing anything").

When planning, the agent enters **planning mode** — it analyzes the codebase, reads docs, thinks through the approach, but **does not modify code**. The plan appears in the right panel.

**Single-plan scenario (most common):**
```
┌──────────────────────┬──────────────────────┐
│ Chat                 │ Proposed Plan        │
│                      │                      │
│ You: Add dark mode   │ Dark Mode Support    │
│ support              │ Claude Code          │
│                      │                      │
│ Agent: I've analyzed │ 1. Create theme      │
│ the codebase. Here's │    context & tokens  │
│ my plan...           │ 2. Update components │
│                      │ 3. Add toggle switch │
│                      │ 4. Persist user pref │
│                      │                      │
│ You: Can you also    │ Deps: 3,4 depend on 1│
│ handle system pref?  │                      │
│                      │ [Approve] [Reject]   │
│ Agent: Good point,   │                      │
│ I'll add that to     │                      │
│ step 4...            │                      │
│                      │                      │
│ [Input box]          │                      │
└──────────────────────┴──────────────────────┘
```

**Multiple competing plans (optional):**

The planning agent can spawn additional planning sessions from other agents/models. These appear as new tabs. The user can switch between tabs to watch planners work and give feedback.

```
┌──────────────────────────────────────────────────────┐
│ [Claude Code] [Planner: GPT-5] [Planner: Gemini] [+]│
├──────────────────────┬───────────────────────────────┤
│ (Claude tab shown)   │ Plans                         │
│                      │                               │
│ Agent: I've started  │ ┌─ Plan A: Claude (done) ────┐│
│ planning. I also     │ │ 8 tasks, Stripe SDK        ││
│ kicked off GPT-5     │ │ [View] [Select]            ││
│ and Gemini for       │ └────────────────────────────┘│
│ alternative plans.   │                               │
│                      │ ┌─ Plan B: GPT-5 (working) ──┐│
│ GPT-5 is still       │ │ ...                        ││
│ working.             │ └────────────────────────────┘│
│                      │                               │
│ [Input box]          │                               │
└──────────────────────┴───────────────────────────────┘
```

User can switch to a planner tab to give direct feedback:

```
┌──────────────────────────────────────────────────────┐
│ [Claude Code] [Planner: GPT-5] [+]                   │
├──────────────────────┬───────────────────────────────┤
│ (GPT-5 tab shown)    │ Plans                         │
│                      │                               │
│ GPT-5: I see an      │ (plan cards)                  │
│ existing payments    │                               │
│ table...             │                               │
│                      │                               │
│ You: We want to      │                               │
│ support multiple     │                               │
│ providers, not just  │                               │
│ Stripe.              │                               │
│                      │                               │
│ GPT-5: Good point.   │ (plan updates live as         │
│ I'll design for      │  planner revises)             │
│ multi-provider...    │                               │
│                      │                               │
│ [Input box]          │                               │
└──────────────────────┴───────────────────────────────┘
```

**Plan detail view** (right panel with tab navigation):

```
┌──────────────────────────────────────────────┐
│ [Plan A: Claude] [Plan B: GPT-5]            │  <- plan tabs (if multiple)
├──────────────────────────────────────────────┤
│ [Description] [Tasks] [Graph]               │  <- detail sub-tabs
├──────────────────────────────────────────────┤
│                                              │
│ Approach: Stripe SDK + webhook handlers      │
│                                              │
│ Summary: Build payment integration using     │
│ Stripe's official SDK...                     │
│                                              │
│ Considerations:                              │
│ - PCI compliance via Stripe.js              │
│ - Idempotency keys for retries              │
│                                              │
│ 8 tasks across 4 parallel groups            │
│                                              │
│                        [Open] [Select]       │
└──────────────────────────────────────────────┘
```

Tasks sub-tab shows the task list with per-task plan approval mode (auto/user). Graph sub-tab shows the ReactFlow DAG.

**Plan comparison (multi-agent):**

| Criteria | Plan A (Claude) | Plan B (GPT-5) |
|----------|----------------|----------------|
| Tasks | 8 | 6 |
| Approach | Stripe SDK direct | Adapter pattern |
| Parallel groups | 4 | 3 |
| Key risk | PCI compliance | Over-abstraction |

---

### 3. Plan Approval & Context Reset

When the user approves a plan, the session context **resets**. The planning conversation is preserved (user can scroll back) but the agent starts fresh with:

- The thread goal
- The approved plan (structured)
- Template configuration (if any)

This keeps the agent's working context clean and focused for execution.

**What happens after approval depends on the plan:**

#### Simple plan (no tasks)

The plan describes a single unit of work. The agent executes it directly in the same thread, same session. No tasks created, no orchestration needed.

```
Thread: Add dark mode
├── [Session: Claude Code] [+]
├────────────────────────────────────────────┤
│ ── Context reset ──                        │
│                                            │
│ Agent: Plan approved. I'll start by        │
│ creating the theme context...              │
│                                            │
│ [Tool calls, edits, progress...]           │
│                                            │
│ Agent: Done. Dark mode is working.         │
│                                            │
│ [Input box]                                │
└────────────────────────────────────────────┘
```

#### Multi-task plan

The plan describes multiple tasks with dependencies. Tasks are created, and the thread-level chat becomes coordination-aware.

```
┌──────────────────────┬──────────────────────┐
│ Thread Chat          │ Tasks (DAG view)     │
│                      │                      │
│ ── Context reset ──  │  [ReactFlow graph]   │
│                      │  Setup -> Models     │
│ Agent: Plan approved.│            |         │
│ 8 tasks created.     │  Webhooks -> Tests   │
│ Starting with setup  │                      │
│ tasks...             │ * Task 2: Models     │
│                      │   Agent: Claude      │
│ [Task 1 done]        │   [View session ->]  │
│ [Task 2 running]     │                      │
│ [Task 3 blocked]     │ Progress: 1/8        │
│                      │                      │
│ Agent: Task 2 agent  │                      │
│ found that the DB    │                      │
│ uses UUIDs. I've     │                      │
│ updated Task 3's     │                      │
│ context with this.   │                      │
│                      │                      │
│ [Input box]          │                      │
└──────────────────────┴──────────────────────┘
```

#### Multi-thread plan (rare, complex)

For very large workstreams, the plan can specify creating **child threads**, each handling a slice of the work. Each child thread does its own planning (with or without task decomposition) and executes independently.

```
Thread A (parent)
  | Plan approved: 3 workstreams
  |
  |-- Thread B: "OAuth integration"
  |     \-- does its own planning -> maybe 3 tasks
  |
  |-- Thread C: "API key management"
  |     \-- does its own planning -> executes directly
  |
  \-- Thread D: "RBAC system"
        \-- does its own planning -> 5 tasks with dependencies
```

The parent thread tracks child thread progress. Each child thread is autonomous — it gets context from the plan and operates independently.

---

### 4. Execution (Threads with Tasks)

After plan approval and context reset, the thread-level chat is coordination-aware. It's not a separate "overseer" persona — it's the same agent, same session, just with richer context (plan, task statuses, accumulated learnings).

**What the thread-level agent can do during execution:**
- Track progress across all tasks (receives status updates)
- Surface escalations from task agents ("Task 3 hit a blocker")
- Enrich upcoming task context with learnings from completed tasks
- Answer coordination questions ("Should task 4 wait for task 2?")
- Propose plan amendments (add/modify/remove pending tasks)
- Detect problems (stuck tasks, conflicting file changes, direction drift)
- Execute simple follow-up work directly (without spinning up a task)

**What it doesn't do:**
- Write code in task sessions (tasks have their own agents)

---

## Plan Amendments

After a plan is approved and tasks are running, the user may want to change scope. This happens through the **thread-level chat**.

### Amendment via Chat

```
User: "Also add rate limiting to the API endpoints"

Agent proposes amendment:
┌─ Plan Amendment ──────────────────────┐
│ Add 2 new tasks:                      │
│                                       │
│ + T9: Implement rate limiter          │
│   depends on: T3 (API routes)         │
│                                       │
│ + T10: Add rate limit tests           │
│   depends on: T9                      │
│                                       │
│ [Approve amendment] [Reject]          │
└───────────────────────────────────────┘

User approves -> tasks added, graph updates, execution continues
```

### Amendment via New Planning Session

For bigger amendments, the thread-level agent can start a **new planning session** — re-enter planning mode, analyze the current state (completed tasks, learnings, running work), and propose a structured set of changes.

```
User: "We need to rethink the API layer based on what Task 1 discovered"

Agent: "Let me plan this properly. Starting a planning session..."

[Agent enters planning mode, analyzes current state + learnings]

Agent proposes amendment:
  + T7: Redesign API routes (replaces pending T4)
  + T8: Update integration tests
  - T4: removed (superseded by T7)
  ~ T5: modified description (depends on T7 instead of T4)

User approves -> plan updates, execution continues
```

### Amendment Principles

- **Completed tasks are immutable** — can't modify or remove them
- **Running tasks can be aborted** — agent stops them, marks as cancelled
- **Pending/blocked tasks can be modified, reordered, or removed** — full flexibility on work that hasn't started
- The thread-level agent proposes amendments (not the user directly), so dependencies and ordering stay consistent
- Amendment history is visible in the chat for auditability
- Running tasks are **not interrupted** by amendments — new tasks slot in based on their dependencies

---

## Task-Level Planning

Each task can optionally have a **planning phase** before execution begins. This is configured either:

1. **From the plan template** — template sets a default `planApproval` for all tasks
2. **From the thread-level plan** — the planner sets `planApproval` per task when creating the plan
3. **From explicit user input** — user overrides on a specific task

### Plan Approval Modes

| Mode | Who approves | When to use |
|------|-------------|-------------|
| `"auto"` | Thread-level agent approves automatically | Routine/straightforward tasks, low risk |
| `"user"` | User must approve | Critical tasks, architectural decisions, high-risk changes |

### Task Planning Flow

```
Task status: pending
     |
     v (dependencies met)
Task status: planning
     |
     v (planner agent analyzes task context + learnings from prior tasks)
Plan proposed
     |
     |-- planApproval: "auto" -> thread-level agent reviews -> approved
     |
     \-- planApproval: "user" -> user sees plan in task detail
                                    |
                              +-----+-----+
                              |           |
                           Approve     Reject
                              |      (with feedback)
                              |           |
                              v           v
                        Task executes  Planner revises
```

### Why Task-Level Planning Matters

- Each task benefits from **accumulated learnings** from prior tasks
- The planner can adapt the approach based on what was discovered during earlier execution
- For complex tasks, a quick plan ensures the agent doesn't go off-track
- `"auto"` approval keeps things fast for routine tasks; `"user"` gives control where it matters

---

## Task-Level UX (Clicking into a Task)

Each task can have **multiple session tabs**:

**Task in planning phase:**
```
┌──────────────────────────────────────────────┐
│ Task: Build auth middleware                   │
│ [Planner: Claude] [+]                        │
├──────────────────────┬───────────────────────┤
│ Task Planner Chat    │ Task Plan             │
│                      │                       │
│ Claude: Based on     │ Status: Planning      │
│ learnings from T1    │ Approval: user        │
│ (DB uses UUID keys), │                       │
│ I'll use UUID for    │ Proposed steps:       │
│ auth tokens too.     │ 1. Create middleware  │
│                      │ 2. Add JWT validation │
│ Here's my plan...    │ 3. Error handlers     │
│                      │                       │
│                      │ [Approve] [Reject]    │
│ [Input box]          │                       │
└──────────────────────┴───────────────────────┘
```

**Task in execution phase:**
```
┌──────────────────────────────────────────────┐
│ Task: Build auth middleware                   │
│ [Session 1: Claude] [Session 2: GPT-5] [+]  │
├──────────────────────┬───────────────────────┤
│ Task Chat            │ Task Details          │
│                      │                       │
│ Agent: Implementing  │ Status: Running       │
│ the auth middleware   │ Context: inherited    │
│ per the approved     │ Dependencies: T1      │
│ plan...              │                       │
│                      │ Learnings from T1:    │
│ [Tool calls...]      │ - DB uses UUID keys   │
│                      │ - Auth table exists   │
│                      │                       │
│                      │ Files changed:        │
│                      │ - auth/middleware.ts   │
│                      │ - auth/jwt.ts         │
│                      │                       │
│ [Input box]          │                       │
└──────────────────────┴───────────────────────┘
```

Each session tab is an independent agent chat working on the same task. All sessions share the same directory/branch.

---

## Task Execution Environment

Task agents don't just edit files — they need to **run** software. This requires controlled access to system resources, declared upfront in the plan and verified at completion.

### Port Allocation

Ports are **declared in the task plan** with named IDs (following k8s resource naming conventions: lowercase, alphanumeric, dashes). The agent requests them by ID at runtime, and the system allocates actual port numbers.

**In the plan (textual, per-task):**
```
Task: Build auth middleware

Ports:
  - id: api-server
    description: "Express dev server for the auth API"
  - id: test-db
    description: "PostgreSQL instance for integration tests"
```

**At runtime:**
- Agent calls `request_port(id: "api-server")` -> worker allocates port 3001 -> returns `{ id: "api-server", port: 3001 }`
- Agent calls `request_port(id: "test-db")` -> worker allocates port 5433 -> returns `{ id: "test-db", port: 5433 }`
- Ports are isolated per task — no collisions between parallel tasks
- When a task completes/fails/cancels, its ports are released back to the pool

**Verification**: After the agent reports completion, the system checks that all declared ports were actually requested. If the plan says the task needs `api-server` and `test-db` but the agent only requested `api-server`, that's a verification failure — the agent likely skipped part of the work.

```
AllocatedPort {
  id: string                      // Plan-declared ID (e.g. "api-server")
  port: number                    // Actual allocated port (e.g. 3001)
  description: string             // From the plan
  allocatedAt: timestamp
  status: "active" | "released"
}
```

The thread-level agent can see all port allocations across tasks — useful for wiring services together (e.g. "Task 3's `api-server` is on :3001, Task 4 should connect there").

### Terminal Access

Task agents can request terminal sessions to run long-lived processes (dev servers, watchers, build processes). Terminals are managed by the worker, visible in the task UI, and killed on task completion.

Flowgentic uses the **ACP terminal standard** for this:
- Capability check via `initialize` response (`clientCapabilities.terminal = true`)
- `terminal/create` to start commands
- `terminal/output` for incremental output reads
- `terminal/wait_for_exit` to wait for completion
- `terminal/kill` for timeout/abort behavior
- `terminal/release` to free resources when done

Reference: `https://agentclientprotocol.com/protocol/terminals.md`

```
Terminal {
  id: string
  label: string                   // e.g. "dev server", "test runner"
  command: string                 // The running command
  status: "running" | "exited"
  exitCode?: number
  port?: string                   // Associated port ID, if any
}
```

**Task UI with terminals and ports:**
```
┌──────────────────────────────────────────────┐
│ Task: Build auth middleware                   │
│ [Session 1: Claude] [+]                      │
├──────────────────────┬───────────────────────┤
│ Task Chat            │ Task Details          │
│                      │                       │
│ Agent: Starting the  │ Status: Running       │
│ dev server to test   │                       │
│ the middleware...    │ Ports:                │
│                      │ * api-server  -> :3001│
│ [Tool calls...]      │ * test-db     -> :5433│
│                      │                       │
│                      │ Terminals:            │
│                      │ * dev server (api-srv)│
│                      │ * test watcher        │
│                      │                       │
│ [Input box]          │                       │
└──────────────────────┴───────────────────────┘
```

---

## Verification Steps

When a task agent reports completion, the system runs **verification steps** — deterministic checks (bash commands, health checks, file checks) and agent-based review steps. If verification fails, results are fed back to the agent, and it loops.

### Verification Flow

```
Agent reports: "I'm done"
     |
     v
Run verification steps (sequentially)
     |
     |-- Step 0: Resource checks (automatic)
     |   - Were all declared ports requested?
     |   - Are required terminals still running?
     |
     |-- Steps 1..N: Defined verification steps (from plan/template)
     |   - Commands (tests/lint/build), health checks, file checks
     |   - Agent review steps (prompt + agent + optional model)
     |
     |-- All required steps pass
     |      |
     |      v
     |   Human feedback gate (final)
     |      |-- Approve -> Task status: completed
     |      \-- Feedback/reject -> Agent revises, reruns verification
     |
     \-- Any fail -> Feed failure output back to agent
                      |
                      v
                Agent fixes and reports done again
                      |
                      v
                (loop -- up to maxRetries)
                      |
                      \-- Max retries exceeded -> Task status: failed
                                                  Escalate to user
```

### Verification Enforcement Rules

- Required steps cannot be skipped
- If a required step does not execute (timeout, tool error, infra error), verification fails
- Task completion requires all required steps to execute and pass
- Optional steps can fail without blocking completion (but are still reported)

### Verification Step Types

| Type | Example | What it checks |
|------|---------|---------------|
| `command` | `pnpm test --filter auth` | Exit code 0 = pass |
| `command` | `make lint` | Exit code 0 = pass |
| `command` | `pnpm tsc --noEmit` | TypeScript compilation |
| `command` | `go test ./internal/auth/...` | Go tests for a package |
| `http_health` | `GET http://localhost:${ports.api-server}/health` | Returns 2xx |
| `file_exists` | `src/auth/middleware.ts` | File was actually created |
| `port_requested` | (automatic) | All declared ports were requested |
| `script` | `./scripts/verify-migrations.sh` | Custom project-specific check |
| `review_agent` | Prompted security/code review | Agent reviews output; blocking controlled by `required` |

Verification steps can reference port IDs with `${ports.<id>}` syntax — resolved to actual port numbers at runtime.

### Agent-Requested Verification

Task agents can **trigger verification on demand** during execution (not just at the end):

- Agent calls `run_verification` tool -> system runs all steps -> results returned to agent
- This is informational only — doesn't affect task status
- Useful for: "let me check if my changes so far pass tests before continuing"

### Verification in the UI

```
┌──────────────────────────────────────────────┐
│ Task: Build auth middleware                   │
├──────────────────────┬───────────────────────┤
│ Task Chat            │ Verification          │
│                      │                       │
│ Agent: Done! I've    │ Attempt 2 of 3        │
│ implemented the      │                       │
│ middleware and tests. │ done Port allocation  │
│                      │ done TypeScript       │
│ System: Verification │ done Lint             │
│ failed (attempt 1).  │ * Unit tests (running)│
│ Unit tests:          │ o Health check :3001  │
│ FAIL middleware.test  │                       │
│ Expected 401...      │ Attempt 1: failed     │
│                      │  done Ports/TS/Lint   │
│ Agent: I see, the    │  fail Tests           │
│ status code was      │  skip Health          │
│ wrong. Fixing...     │                       │
│                      │                       │
│ [Input box]          │                       │
└──────────────────────┴───────────────────────┘
```

---

## Task Recovery & Abort

When things go wrong during execution:

- **Abort a task**: User or thread-level agent can stop a running task. Session is terminated, task marked "cancelled". Dependencies that relied on it become "blocked".
- **Retry a task**: User can retry a failed/cancelled task — spawns a new session for it. Previous session preserved for reference.
- **Abort all**: User can stop the entire plan execution. All running tasks cancelled, pending tasks stay pending. User can amend and restart.
- **Conflict resolution**: If a task agent's changes conflict with another session's work, the thread-level agent surfaces this and asks the user how to resolve (keep A, keep B, merge manually).

---

## Learnings Pattern

When a task completes, its learnings automatically bubble up:
- **Subtask -> Parent task**: Parent accumulates child learnings
- **Top-level task -> Thread**: Thread-level agent receives task learnings
- Learnings include: "what worked", "what was unexpected", "what the next task should know"
- The thread-level agent uses accumulated learnings to enrich upcoming task context and inform coordination decisions

---

## Feedback Loops & Oversight

Execution uses layered feedback loops to stay autonomous without removing human control.

### 1) Verification Feedback Loop (mandatory)

Every executing task/agent runs a fix-and-retry loop:
- Implement a change
- Run verification steps in order
- Inspect failures/findings
- Fix and rerun

### 2) Supervisory Feedback Loop (thread-level)

The thread-level agent watches loop outcomes across tasks:
- Repeated verification failures / rising retries
- No-progress windows
- Direction drift from approved plan
- Conflicts between concurrent sessions

Based on this, it can: continue, pause for user input, propose a plan amendment, or recommend abort/retry.

### 3) Human Feedback Loop (final approval gate)

Human feedback is the final step after verification:
- User reviews verified output
- User approves, or provides feedback for revision
- On feedback/reject, agent revises and reruns verification
- Task/thread finishes only after required human approval points are satisfied

---

## Plan Templates

Plan templates define **project-level conventions** that apply to all plans. They capture verification, branching, resource, and task-splitting defaults so every plan inherits a consistent baseline.

A project typically has **one default template** and possibly a few variants (e.g. "hotfix" for minimal process, "spike" for exploratory work).

Templates are discovered from both:
- **Repo scope**: `.flowgentic/plan-templates/`
- **User scope**: `~/.flowgentic/plan-templates/`

### Template Locations & Format

Each template lives in its own directory:

```
.flowgentic/plan-templates/<template-name>/TEMPLATE.md
~/.flowgentic/plan-templates/<template-name>/TEMPLATE.md
```

`TEMPLATE.md` is the source of truth — structured frontmatter + freeform guidance. Optional supporting files can live beside it (e.g. `checks/`, `assets/`, `agents/openai.yaml`).

### Template Discovery & Selection

At plan creation, Flowgentic lists templates from both scopes in one picker.

Selection precedence:
1. User explicitly selected template
2. Repo default template (if defined)
3. User default template (if defined)
4. Built-in fallback template

If two templates share the same name, both are shown with scope labels:
- `Default (Repo)`
- `Default (User)`

### What a Template Defines

```
PlanTemplate {
  id: string
  name: string                      // e.g. "Default", "Hotfix", "Spike"
  description: string
  source: "repo" | "user" | "builtin"
  path: string                      // Path to TEMPLATE.md used for this run

  // Verification steps applied to every task
  verification: {
    steps: VerificationStep[]       // e.g. build, lint, test — always run
    maxRetries: number              // Default retry limit (default: 3)
  }

  // Resource conventions
  resources: {
    defaultPorts?: { id: string, description: string }[]
  }

  // Branching & PR strategy
  git: {
    branchStrategy: "per-task" | "shared"
    branchPrefix?: string           // e.g. "flowgentic/" -> "flowgentic/task-3-auth"
    prTarget?: string               // Target branch for PRs
    openPR: boolean
  }

  // Planning hints (injected into planner context)
  planningHints: string

  // Agent instructions (injected into every task agent's context)
  agentInstructions?: string
}
```

### Example Templates

**Default:**
```yaml
name: "Default"
description: "Standard project conventions"

verification:
  steps:
    - type: command
      label: "Build"
      command: "make build"
    - type: command
      label: "Lint"
      command: "make lint"
    - type: command
      label: "Tests"
      command: "make test"
    - type: review_agent
      label: "Security review"
      agent: "claude-code"
      model: "sonnet"
      prompt: "Review auth/token/session changes for security issues."
      required: true
  maxRetries: 3

resources:
  defaultPorts:
    - id: dev-server
      description: "Application dev server"

git:
  branchStrategy: "per-task"
  branchPrefix: "flowgentic/"
  prTarget: "main"
  openPR: true

planningHints: |
  - Split tasks so each is independently testable
  - Database migrations should be their own task
  - Keep tasks small: 1-3 files changed is ideal, 5+ means split it
  - If a task needs more than 2 ports, it's probably too big
```

**Hotfix:**
```yaml
name: "Hotfix"
description: "Minimal process for urgent fixes"

verification:
  steps:
    - type: command
      label: "Build"
      command: "make build"
    - type: command
      label: "Tests"
      command: "make test"
  maxRetries: 2

git:
  branchStrategy: "shared"
  branchPrefix: "hotfix/"
  prTarget: "main"
  openPR: true

planningHints: |
  - Keep it to 1-2 tasks maximum
  - Focus on the fix, not refactoring
  - Add a regression test
```

### How Templates Are Used

1. Template's `planningHints` are injected into the planner's context alongside the user's goal
2. When tasks are created, template's `verification.steps` are auto-attached to each task
3. Template's `git` strategy determines branching and PR behavior
4. Template's `defaultPorts` are auto-declared on every task
5. Template's `agentInstructions` are injected into every task agent's context
6. The planner can override any template default per-task (e.g. skip lint for a docs-only task)
7. The thread records template `source` + `path` for reproducibility/auditability

---

## FLOWGENTIC.md — Project Knowledge Map (Optional)

Projects can optionally include a `FLOWGENTIC.md` file — a Flowgentic-specific extension to AGENTS.md that tells Flowgentic where to find structured project knowledge.

### When to use it

- **Not required.** Projects work fine without it.
- **Useful when** a project wants structured knowledge management — design docs, plan history, quality scores.
- **Flexible paths.** FLOWGENTIC.md just points to wherever the docs live.

### What it contains

A short file (~100 lines max) that acts as a **table of contents**:

```markdown
# FLOWGENTIC.md

## Knowledge Base
- Architecture: docs/architecture.md
- Conventions: docs/conventions.md
- Design docs: docs/design/

## Plans
- Active plans: docs/plans/active/
- Completed plans: docs/plans/completed/
- Tech debt: docs/plans/tech-debt.md

## Quality
- Quality scores: docs/quality.md

## References
- External docs (LLM-friendly): docs/references/
```

### How Flowgentic uses it

- **Planners** read it to discover design docs, prior plans, and architectural context
- **Task agents** use it to find conventions and references
- **Thread-level agents** use it to update plan status and record learnings
- If it doesn't exist, agents fall back to AGENTS.md / CLAUDE.md — nothing breaks

### Plans as repo artifacts (optional)

When FLOWGENTIC.md points to a plans directory, Flowgentic can persist execution plans as versioned markdown:
- Active plans with progress logs in `plans/active/`
- Completed plans moved to `plans/completed/`
- Agents can read past plans to understand prior decisions

---

## Multi-Session Model

### Sessions as Tabs (Uniform Pattern)

Both threads and tasks support multiple session tabs via a [+] button. Each session is an independent agent chat.

**Thread-level sessions:**
```
Thread: Make app responsive
├── [Session: Claude Code] [Session: Codex] [+ New]
├────────────────────────────────────────────────────┤
│ (active session's chat)                            │
└────────────────────────────────────────────────────┘
```

- User manually spawns sessions with [+] and picks agent/model
- All sessions share the same directory/branch (or git worktree)
- No dependency tracking, no status rollup — just parallel chats
- If two sessions edit the same file, the UI surfaces which files each session has modified

**Task-level sessions:**
```
Task: Build auth middleware
├── [Session 1: Claude] [Session 2: GPT-5] [+ New]
├──────────────────────┬───────────────────────────┤
│ Agent chat           │ Task details              │
│ (active tab's chat)  │ Files, status, etc.       │
└──────────────────────┴───────────────────────────┘
```

Same pattern — multiple agents working on the same task, each in their own session tab.

### Planner Sessions (Thread-Level, During Planning)

When the thread-level agent spawns additional planners, those appear as session tabs too. Visually they may be distinguished (e.g. labeled "Planner: GPT-5") but they use the same tab infrastructure.

After plan approval, planner session tabs can be archived/collapsed — their work is done.

---

## Data Model

### Thread

```
Thread {
  id, projectId, topic, status, archived

  plan?: Plan                          // The approved plan (if any)
  parentThreadId?: string              // For child threads (multi-thread plans)

  // Template used for this thread's plan
  templateSource?: "repo" | "user" | "builtin"
  templatePath?: string
}
```

A thread's state is derived from its data:
- Has no plan, no tasks → chat thread
- Has a plan, no tasks → single-unit planned thread (agent executes directly)
- Has a plan and tasks → multi-task planned thread (coordination-aware)
- Has child threads → multi-thread planned thread (rare)

### Session

```
Sessions within a thread:
  - Thread-level sessions: task_id = null. Independent chats on the same codebase.
  - Planner sessions: task_id = null, role = "planner". Created during planning phase.
  - Task sessions: task_id set. Execution agents for a specific task.
```

### Task

```
Task {
  id, threadId, description, dependencies
  status: pending | planning | running | completed | failed | blocked | cancelled

  // Context (inherited + enriched with learnings)
  context: {
    own: string              // This task's specific context/instructions
    parent?: string          // Parent task context (for subtasks)
    thread: string           // Thread-level context (goal, plan summary)
  }

  // Task-level planning (runs BEFORE execution)
  planApproval: "auto" | "user"
  planStatus?: TaskPlanStatus
  plan?: TaskPlan

  // Learnings (bubble up to thread on completion)
  learnings: string[]

  // Declared resources (from plan)
  declaredPorts: { id: string, description: string }[]

  // Runtime resources (populated during execution)
  resources: {
    ports: AllocatedPort[]
    terminals: Terminal[]
  }

  // Verification
  verification: {
    steps: VerificationStep[]
    maxRetries: number
    currentAttempt: number
    results: VerificationResult[]
  }
}
```

### VerificationStep

```
VerificationStep {
  id: string
  type: "command" | "http_health" | "file_exists" | "script" | "review_agent"
  label: string
  required?: boolean                // default true
  command?: string                  // For command/script types (supports ${ports.xxx})
  url?: string                      // For http_health type (supports ${ports.xxx})
  path?: string                     // For file_exists type
  prompt?: string                   // For review_agent type
  agent?: string                    // For review_agent type
  model?: string                    // Optional model override for review_agent
  timeout?: number                  // Max execution time (ms)
  continueOnFail?: boolean          // Run remaining steps even if this fails (default: false)
}
```

### VerificationResult

```
VerificationResult {
  attempt: number
  steps: {
    stepId: string
    passed: boolean
    output: string
    durationMs: number
  }[]
  allPassed: boolean
  timestamp: timestamp
}
```

---

## Summary of Flows

### Chat Flow (most common)
```
Create thread -> User chats with agent -> Agent executes -> Done
                                            |
                                  (if complex, agent suggests planning)
                                            |
                                            v
                                  User confirms -> enters planning
```

### Single-Unit Plan Flow
```
Agent enters planning mode -> analyzes, proposes plan
                                            |
                              Plan appears in right panel
                                            |
                        +-------------------+-------------------+
                        |                   |                   |
                   Chat with           [Approve]           [Reject]
                   agent to                 |           (with feedback)
                   refine plan              |                   |
                        |                   v                   v
                   Agent revises     Context resets        Agent revises
                        |           Agent executes             |
                   (loop back)      directly in thread    (loop back)
                                            |
                                          Done
```

### Multi-Task Plan Flow
```
Agent enters planning mode -> analyzes, proposes plan with tasks
                                            |
                        (optionally spawns additional planner sessions)
                                            |
                              Plans displayed in right panel
                                            |
                        +-------------------+-------------------+
                        |                   |                   |
                   Chat with           [Approve/             [Reject]
                   planners             Select]           (with feedback)
                   in their tabs             |                   |
                        |                   v                   v
                   Planners revise   Context resets        Planners revise
                        |           Tasks created              |
                   (loop back)      Thread coordinates    (loop back)
                                    task execution
                                            |
                                      +-----+-----+
                                      |           |
                                  Task done    Task fails
                                      |           |
                                      v           v
                                 Next task   Escalate to user
```

---

## Next Steps

This is a **design document**, not an implementation plan. Key implementation areas:

1. Remove old thread mode enum (`"single_agent" | "orchestrated"`) — thread state is derived from data
2. Add session role concept for planner sessions
3. Build adaptive thread UI that changes layout based on thread state (chat vs. planned vs. tasks)
4. Implement planning mode behavior (agent constraint: analyze but don't modify code)
5. Build plan display + comparison view
6. Context reset mechanism after plan approval
7. Multi-session tab UI for both threads and tasks
8. Plan amendment flow (chat-based and planning-session-based)
9. Learnings accumulation and context enrichment pipeline
