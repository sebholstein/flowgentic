# Frontend Alignment Plan

The backend entity model changed significantly. The frontend proto types will auto-regenerate with `make proto`, but the application code needs manual alignment.

## What Changed in the Backend

| Before | After |
|---|---|
| `AgentRunService` (proto) | `SessionService` |
| `AgentRunConfig` (proto) | `SessionConfig` |
| `agent_run_service_pb.ts` (generated) | `session_service_pb.ts` |
| `WorkerService.NewAgentRun` | `WorkerService.NewSession` |
| `AgentRunStatus` enum | `SessionStatus` enum |
| No task entity | `TaskService` with full CRUD |
| No plan on threads | `ThreadConfig.plan` field |
| Single-prompt sessions | Multi-turn sessions (`PromptSession`, `CancelSession` RPCs) |

## Step 0: Regenerate Proto Types

Run `make proto` from the project root. This will regenerate all `frontend/src/proto/gen/` files. After this:

- `agent_run_service_pb.ts` is gone, replaced by `session_service_pb.ts`
- `worker_service_pb.ts` has new types: `SessionStatus`, `SessionMode`, `NewSessionRequest/Response`, `SessionInfo`, `SessionState`, `PromptSessionRequest/Response`, `CancelSessionRequest/Response`, `ContentBlock`
- `thread_service_pb.ts` has a new `plan` field on `ThreadConfig` and `UpdateThreadRequest`
- New file: `task_service_pb.ts` with `TaskService`, `TaskConfig`, and full CRUD messages

Verify: `ls frontend/src/proto/gen/controlplane/v1/session_service_pb.ts` should exist.

## Step 1: Add Session Queries

Create `frontend/src/lib/queries/sessions.ts`:

```typescript
import { queryOptions } from "@tanstack/react-query";
import type { Client } from "@connectrpc/connect";
import type { SessionService } from "@/proto/gen/controlplane/v1/session_service_pb";

export function sessionsQueryOptions(
  client: Client<typeof SessionService>,
  threadId: string,
) {
  return queryOptions({
    queryKey: ["sessions", threadId],
    queryFn: () => client.listSessions({ threadId }),
    enabled: !!threadId,
    refetchInterval: 3000,
  });
}
```

## Step 2: Add Task Queries

Create `frontend/src/lib/queries/tasks.ts`:

```typescript
import { queryOptions } from "@tanstack/react-query";
import type { Client } from "@connectrpc/connect";
import type { TaskService } from "@/proto/gen/controlplane/v1/task_service_pb";

export function tasksQueryOptions(
  client: Client<typeof TaskService>,
  threadId: string,
) {
  return queryOptions({
    queryKey: ["tasks", threadId],
    queryFn: () => client.listTasks({ threadId }),
    enabled: !!threadId,
    refetchInterval: 3000,
  });
}
```

## Step 3: Update the Thread Route Mapping

In `frontend/src/routes/app/threads/$threadId/route.tsx`, the `useMemo` block (around line 96) maps `ThreadConfig` → frontend `Thread` type. Update it to include the new `plan` field:

```typescript
const thread = useMemo<Thread | undefined>(() => {
  const t = threadData?.thread;
  if (!t) return undefined;
  return {
    // ... existing fields ...
    plan: t.plan || undefined,  // NEW
  };
}, [threadData]);
```

Also update the `Thread` type in `frontend/src/types/thread.ts` to add:

```typescript
export interface Thread {
  // ... existing fields ...
  plan?: string;  // markdown plan set by agent in orchestrated mode
}
```

## Step 4: Replace Mock Tasks with Real Task Data

The thread route currently uses `initialTasks = useMemo<Task[]>(() => [], [])` (line ~127 in route.tsx). Replace this with a real query:

```typescript
import { TaskService } from "@/proto/gen/controlplane/v1/task_service_pb";
import { tasksQueryOptions } from "@/lib/queries/tasks";

// Inside ThreadLayout:
const taskClient = useClient(TaskService);
const { data: tasksData } = useQuery(tasksQueryOptions(taskClient, threadId));
```

Map the backend `TaskConfig` to the frontend `Task` type. The backend task is simpler than the frontend type — many frontend fields (`executions`, `checkIn`, `inputRequirements`, etc.) are UI-only concepts from mock data. Create a mapping function:

```typescript
function mapBackendTask(t: TaskConfig): Task {
  return {
    id: t.id,
    name: t.description,         // backend has description, not name
    description: t.description,
    status: t.status as TaskStatus,  // backend: pending/running/done/failed
    subtasks: t.subtasks.map((s, i) => ({
      id: `${t.id}-sub-${i}`,
      name: s,
      completed: false,  // TODO: track completion in subtask string format
    })),
    dependencies: [],
  };
}
```

## Step 5: Add Session List to Thread View

Sessions represent the agent's work attempts. Display them somewhere in the thread UI (sidebar or detail panel). Use the `SessionService` client:

```typescript
import { SessionService } from "@/proto/gen/controlplane/v1/session_service_pb";
import { sessionsQueryOptions } from "@/lib/queries/sessions";

const sessionClient = useClient(SessionService);
const { data: sessionsData } = useQuery(sessionsQueryOptions(sessionClient, threadId));
```

Each `SessionConfig` has: `id`, `threadId`, `workerId`, `prompt`, `status`, `agent`, `model`, `mode`, `sessionMode`, `agentSessionId`, `taskId`, `createdAt`, `updatedAt`.

- Sessions with `taskId = ""` are thread-level (overseer) sessions
- Sessions with `taskId = <id>` are task worker sessions

## Step 6: Wire SetSessionMode to Real API

The frontend already has UI for changing session mode (`sessionMode` state + selector in `AgentChatPanel`). Currently it's local state only. Wire it to the backend:

```typescript
import { SessionService } from "@/proto/gen/controlplane/v1/session_service_pb";

const sessionClient = useClient(SessionService);

// When user changes mode:
const changeModeMutation = useMutation({
  mutationFn: ({ sessionId, modeId }: { sessionId: string; modeId: string }) =>
    sessionClient.setSessionMode({ sessionId, modeId }),
});
```

The tricky part: the frontend needs to know the active session ID for the current thread. Use the sessions query to find the most recent non-stopped session.

## Step 7: Add Multi-Turn Prompt Support

The backend now supports `PromptSession` and `CancelSession` RPCs on the worker. These go through the control plane. If the frontend needs to send follow-up prompts to a running session:

1. Add `PromptSession` and `CancelSession` RPCs to the control plane proto (they currently only exist on the worker proto — the CP needs to proxy them like it does for `SetSessionMode`)
2. Or call the worker directly if the frontend has worker access

This is a future enhancement — skip for now unless the chat UI needs multi-turn.

## Step 8: Clean Up Mock Data

The frontend uses mock data from `frontend/src/data/` files. As real backend data replaces mocks:

- Remove references to `mockInboxData` in the thread route
- Remove `useTaskSimulation` hook usage (it simulates planning phases locally)
- The `TaskDetailSidebar` and `ThreadGraphView` components may need updating to work with real task data instead of the rich mock `Task` type

## Step 9: Update the Frontend `Task` Type

The current frontend `Task` type in `types/task.ts` is much richer than the backend entity. Decide on one of:

**Option A: Keep frontend type rich, map from backend.** The backend `TaskConfig` maps to a subset of the frontend `Task`. Fields like `executions`, `checkIn`, `plannerPrompt`, etc. remain frontend-only or are populated from other sources (sessions = executions).

**Option B: Simplify frontend type to match backend.** Remove unused fields and let the backend be the source of truth. This is cleaner but requires updating all components that use the rich type.

Recommended: **Option A** for now — map backend tasks and fill in UI-only fields with defaults.

## Files to Modify

| File | Change |
|---|---|
| `src/types/thread.ts` | Add `plan?: string` |
| `src/lib/queries/sessions.ts` | **New** — session query options |
| `src/lib/queries/tasks.ts` | **New** — task query options |
| `src/routes/app/threads/$threadId/route.tsx` | Add plan to thread mapping, replace mock tasks with real query, add session query |
| `src/routes/app/threads/new.tsx` | No changes needed (CreateThread API unchanged) |
| `src/components/chat/AgentChatPanel.tsx` | Wire `setSessionMode` to real API |
| `src/components/threads/TaskDetailSidebar.tsx` | Update to handle real task data |
| `src/components/threads/ThreadsSidebar.tsx` | No changes needed (uses thread list) |

## Verification

1. `make proto` — regenerates frontend proto types
2. `cd frontend && pnpm tsc --noEmit` — type check
3. `cd frontend && pnpm test` — run vitest
4. Manual: create a thread, verify sessions appear, verify tasks appear for orchestrated mode
