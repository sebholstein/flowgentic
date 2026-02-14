# Session Creation Flow

## Overview

The **control plane initiates sessions**, triggered by a client calling `CreateThread` with a prompt. Sessions use an async dispatch pattern: created as pending, then picked up by a reconciler and dispatched to the worker.

## Flow

### 1. Control Plane: Thread creation triggers session

- **`internal/controlplane/thread/thread_service_handler.go:60-87`** — `CreateThread()` handler. If `Prompt` is provided, calls `sessionCreator.CreateSessionForThread()` (line 78).
- **`internal/controlplane/session/session_service.go:46-70`** — Creates a `Session` record with status `"pending"` in the DB, then calls `reconciler.Notify()`.

### 2. Control Plane: Reconciler dispatches to worker

- **`internal/controlplane/session/reconciler.go:56-128`** — Polls for pending sessions, looks up the worker, and calls the worker's `NewSession` RPC (lines 94-104). Updates status to `"running"` on success.

### 3. Worker: Receives and launches

- **`internal/worker/workload/worker_service_handler.go:162-217`** — `NewSession()` RPC handler. Validates the request, builds `LaunchOpts`, calls `svc.Schedule()`.
- **`internal/worker/workload/session_manager.go:67-114`** — `Launch()` injects `AGENTCTL_*` env vars, calls `driver.Launch()`, registers the session.

### 4. Worker: Driver runs ACP protocol

- **`internal/worker/driver/v2/subprocess.go:38-99`** — `Launch()` creates in-process pipes for the ACP connection and spawns `runSession()` in a goroutine.
- **`internal/worker/driver/v2/subprocess.go:158-270`** — `runSession()` runs the ACP protocol sequence:
  1. `conn.Initialize()` (~line 170)
  2. `conn.NewSession()` with meta containing systemPrompt, model, sessionMode (~line 195)
  3. `conn.Prompt()` with the initial prompt (~line 222)
  4. Idle loop waiting for follow-up prompts via `sess.promptCh` (~line 246)

### 5. Adapter: Translates ACP to Claude SDK

- **`internal/worker/driver/claude/acp/adapter.go:81-116`** — `NewSession()` parses meta options (systemPrompt, model, etc.)
- **`internal/worker/driver/claude/acp/adapter.go:118-170`** — `Prompt()` lazy-inits a persistent Claude SDK client, sends via `QueryWithSession()`, drains messages back through ACP.

## Key Details

- **Single RPC:** The `NewSession` RPC includes the prompt — no follow-up call needed to start work.
- **Async dispatch:** Thread creation and session dispatch are decoupled via the reconciler, so thread creation doesn't block on worker availability.
- **ACP as core protocol:** All agent communication flows through ACP, whether in-process (Claude Code, Codex) or subprocess (OpenCode, Gemini).
- **Environment injection:** The worker injects `AGENTCTL_*` env vars so agents can reach the private CTL listener for tool execution.
- **State sync:** A persistent bidi stream between worker and control plane forwards real-time session state updates.
