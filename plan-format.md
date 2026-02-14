# Plan Format Specification

## Overview

Agents in plan mode write markdown files with YAML frontmatter into dedicated per-thread directories.

- By default, the current thread plan directory is provided directly in the system prompt/runtime context.
- If the agent needs to plan additional threads, it must request additional plan directories from the control plane.
- Each plan directory represents exactly one thread.

Allocated plan directories live under `~/.agentflow/plans/<internal-session-id>/`.
`<internal-session-id>` is the Flowgentic internal session ID (not the agent session ID).
Agents must treat returned paths as opaque and must not construct paths manually.
Plan directories are outside the agent's CWD to avoid git diffs. When ready, the agent runs `agentctl plan commit` to validate and submit all allocated thread plan directories.

## Directory Structure

```text
~/.agentflow/plans/<internal-session-id>/
  plan.md              # Primary thread plan narrative + frontmatter
  tasks/
    01-setup-db.md     # Each task is its own file
    02-implement.md
    03-add-frontend.md
```

The current thread directory (`CURRENT_PLAN_DIR`, or via `agentctl plan dir`) maps to one thread plan.

## Runtime Context Injection

The control plane should inject plan directory context into the system prompt so the agent can start planning immediately.

Required context:

- `CURRENT_PLAN_DIR`: absolute path for the current thread plan directory.
- `ADDITIONAL_PLAN_DIRS`: zero or more already-allocated additional plan dirs (with associated thread IDs).

Resume behavior:

- If a session resumes and previously allocated additional plan directories already exist, include them in `ADDITIONAL_PLAN_DIRS`.
- The agent should continue using those directories instead of re-requesting them.

## `plan.md`

YAML frontmatter plus the primary plan narrative.

`plan.md` is not only metadata. Its markdown body should capture the overall plan context, such as:

- problem framing and approach,
- implementation details and architecture decisions,
- constraints and non-goals,
- high-level acceptance criteria.

```markdown
---
title: "Database migration and API implementation"
---

Freeform markdown describing the overall plan, rationale, approach, etc.
Can contain any markdown: lists, code blocks, links.
```

### Frontmatter fields

| Field | Type | Required | Values | Description |
|---|---|---|---|---|
| `title` | string | yes | non-empty | Short plan summary |

## `tasks/NN-slug.md`

Each task is a separate file. Filename convention: `NN-slug.md` where `NN` is a zero-padded sort index and `slug` is kebab-case.

```markdown
---
id: setup-db
depends_on: []
agent: claude-code
subtasks:
  - Create migration file
  - Add indexes
---

Freeform markdown task description.

Acceptance criteria:
- Migration file exists in `internal/database/migrations/`
- Schema matches the proto definition
```

### Frontmatter fields

| Field | Type | Required | Values | Description |
|---|---|---|---|---|
| `id` | string | yes | kebab-case (`^[a-z0-9]+(-[a-z0-9]+)*$`) | Thread-plan-scoped identifier |
| `depends_on` | []string | no | references task `id` values in the same thread plan | Tasks that must complete first |
| `agent` | string | no | agent driver name | Which agent is assigned |
| `subtasks` | []string | no | strings | Checklist items within the task |

### Subtask example

```yaml
subtasks:
  - Create migration file
  - Add indexes
  - Backfill existing rows
```

### Filename rules

- Must match pattern `^\d{2}-[a-z0-9]+(-[a-z0-9]+)*\.md$`
- The `NN` prefix determines sort order
- No duplicate sort indices
- The slug portion is for human readability only (not used as an identifier)

## Validation Rules

When `agentctl plan commit` is run, the following is validated for each allocated thread plan directory:

1. `plan.md` exists with valid frontmatter.
2. `title` is non-empty.
3. Each task file has valid frontmatter.
4. Task `id` values are unique.
5. Task `id` values are kebab-case.
6. `depends_on` references existing task IDs.
7. No circular dependencies.
8. No duplicate sort indices.
9. Each task has a non-empty markdown body.

All validation errors are reported at once so the agent can fix them in a single pass.

## Multi-Thread Planning

To plan additional threads:

1. Use `CURRENT_PLAN_DIR` from system prompt/runtime context.
2. Request an additional plan directory for each extra thread.
3. Write `plan.md` + `tasks/` in each per-thread directory.
4. Run one `agentctl plan commit` to submit all allocated directories.

Each directory is independent; task IDs and dependencies are scoped to that thread only.

## Proposed `agentctl` Commands

Current state in repo: `agentctl` only implements `set-topic`.

Planned command surface for thread plans:

1. `agentctl plan dir`
Returns the current thread's allocated plan directory path.
This is optional for inspection/debugging when `CURRENT_PLAN_DIR` is already provided in runtime context.

2. `agentctl plan request-thread-dir`
Asks control plane for an additional thread plan allocation.
Returns both:
- `thread_id` (newly allocated thread id)
- `plan_dir` (allocated directory path)

Example response shape:

```json
{"thread_id":"thread_456","plan_dir":"~/.agentflow/plans/01950f64-..."}
```

3. `agentctl plan remove-thread --thread-id <thread-id>`
Removes a previously requested additional thread plan directory from the current planning set.

4. `agentctl plan clear-current`
Clears plan files in the current thread plan directory without removing the directory allocation.

5. `agentctl plan commit`
Validates and commits all allocated thread plan directories in one operation (current thread directory + any additional requested directories).

## Command Workflow

Single-thread (default):

```bash
# CURRENT_PLAN_DIR is provided in system prompt/runtime context
# write/update $CURRENT_PLAN_DIR/plan.md + $CURRENT_PLAN_DIR/tasks/*
agentctl plan commit
```

Multiple threads:

```bash
# request extra thread plan directory
REQ="$(agentctl plan request-thread-dir)"
THREAD_ID="<thread_id from REQ>"
EXTRA_DIR="<plan_dir from REQ>"

# write $EXTRA_DIR/plan.md + $EXTRA_DIR/tasks/*
# write/update $CURRENT_PLAN_DIR too (if needed)

# one commit submits all thread plan dirs
agentctl plan commit
```

Remove an additional thread before commit:

```bash
agentctl plan remove-thread --thread-id "$THREAD_ID"
agentctl plan commit
```

Clear current thread plan content:

```bash
agentctl plan clear-current
agentctl plan commit
```

## Thread Removal Semantics

1. `remove-thread` applies only to additional requested thread directories.
2. Removing the current/default thread is not allowed via `remove-thread`.
3. To remove current thread plan content, use `clear-current`.
4. `remove-thread` deletes the plan directory allocation and excludes that thread from the next `plan commit`.
5. If the provided `--thread-id` is unknown, return a non-zero error.
6. `plan commit` operates on the post-removal set only.

## CLI Usage

```bash
# Agent writes plan files in $CURRENT_PLAN_DIR
# and in any additional requested thread plan dirs, then:
agentctl plan commit
```

## Example

```text
<CURRENT_PLAN_DIR>/
├── plan.md
└── tasks/
    ├── 01-create-schema.md
    ├── 02-implement-service.md
    └── 03-wire-handler.md
```

**plan.md:**
```markdown
---
title: "Add user authentication"
---

We need to add JWT-based authentication to the API.
The approach is to add middleware that validates tokens on protected endpoints.

Implementation details:
- Add token parsing/validation middleware in server middleware chain.
- Inject authenticated principal into request context.

Constraints:
- Keep existing public endpoints unchanged.
- Do not break existing Connect interceptor behavior.

High-level acceptance criteria:
- Protected endpoints reject missing or invalid tokens.
- Valid tokens allow access and expose principal identity to handlers.
```

**tasks/01-create-schema.md:**
```markdown
---
id: create-schema
depends_on: []
---

Create the `users` table migration with columns for id, email, password_hash, created_at.
```

**tasks/02-implement-service.md:**
```markdown
---
id: implement-service
depends_on: [create-schema]
agent: claude-code
subtasks:
  - Define service interface
  - Implement SQLite store
  - Add password hashing
---

Implement `auth_service.go` following the feature pattern.
Service should support `Register`, `Login`, `ValidateToken`.
```

## Overseer System Prompt

Use this as the overseer system prompt:

```text
You are the Flowgentic overseer agent in plan mode.

Your default job is to produce an actionable plan for the current thread and its tasks.
Split work into multiple thread plans only when:
- the user explicitly requests planning across multiple threads, or
- the scope is large enough that it should be decomposed into separate threads.

Rules:
1. After processing the first user message, run `agentctl set-topic "<short description>"` before responding.
2. Keep topic concise (max 100 chars) and update it later if focus changes substantially.
3. If `set-topic` fails, tell the user and retry once.
4. Read `CURRENT_PLAN_DIR` and any `ADDITIONAL_PLAN_DIRS` from runtime context in the system prompt.
5. Start planning in those provided directories directly; do not request current-thread dir again.
6. If planning additional threads beyond provided dirs, request them with `agentctl plan request-thread-dir` and use the returned `thread_id` and `plan_dir`.
7. In every allocated plan dir, write:
   - `plan.md` with YAML frontmatter (`title`) and a clear narrative (approach, implementation details, constraints, high-level acceptance criteria).
   - `tasks/NN-slug.md` task files with frontmatter (`id`, optional `depends_on`, optional `agent`, optional `subtasks`) and non-empty task body.
8. Keep task IDs kebab-case and dependencies valid within the same thread plan dir.
9. If an additional thread should no longer be planned, run `agentctl plan remove-thread --thread-id <thread-id>`.
10. If current thread plan content should be removed, run `agentctl plan clear-current`.
11. When planning is ready, run one `agentctl plan commit` to validate and submit all allocated thread plan dirs.
12. If validation fails, fix all reported issues and retry commit.

Output quality:
- Be concise, specific, and execution-oriented.
- Prefer small, testable tasks with clear acceptance criteria.
- Do not include unrelated work.
```

## Extensibility

Adding new frontmatter fields is non-breaking. Unknown fields are rejected by strict YAML parsing to catch typos.

Examples of future fields:

```yaml
priority: high
estimated_effort: small
tags: [backend, database]
timeout: 300s
```

## Libraries

- Frontmatter parsing: `github.com/adrg/frontmatter`
- YAML engine: `github.com/goccy/go-yaml` (passed via `frontmatter.NewFormat`)
