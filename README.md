# Flowgentic

Flowgentic is a Go-based control-plane + worker system for running coding-agent workloads over Connect RPC, with an optional embedded worker and a desktop UI built with Electron + React.

## Stack

- Backend: Go, Connect RPC, SQLite, optional Tailscale (`tsnet`)
- Frontend: Electron Forge, React 19, Vite, Tailwind v4, TanStack Router/Query, Zustand
- API/schema tooling: Protobuf (`buf`), `sqlc`

## Repository Layout

- `cmd/`: Go entrypoints (`flowgentic-control-plane`, `flowgentic-worker`, `agentctl`, `hookctl`, `acpchat`)
- `internal/`: core packages (`controlplane`, `worker`, `agent`, `config`, `database`, `proto`, `tsnetutil`)
- `internal/proto/definitions/`: protobuf source definitions
- `internal/proto/gen/`: generated protobuf + Connect code
- `frontend/`: Electron + React app
- `bin/`: built binaries
- `docs/`: architecture and project notes
- `flowgentic.json`: default runtime config

## Prerequisites

- Go `1.25.5` (or compatible Go 1.25.x)
- Node.js + `pnpm` (for frontend)
- `buf` (for protobuf generation)
- `sqlc` (for SQL code generation)

## Quick Start (Backend)

From repo root:

```bash
# 1) Build required binaries
make build-worker build-control-plane build-agentctl

# 2) Start worker (in terminal 1)
make run-worker

# 3) Start control plane (in terminal 2)
make run-cp
```

Default local ports:

- Control plane: `127.0.0.1:8420`
- Worker: `:8081`

Health check:

```bash
curl http://127.0.0.1:8420/
```

## Frontend Development

```bash
cd frontend
pnpm install
pnpm dev
```

Useful frontend scripts:

- `pnpm dev`: run Electron app
- `pnpm dev:browser`: run renderer in browser on port `3000`
- `pnpm build`: production build
- `pnpm test`: run Vitest tests

## Configuration

Flowgentic reads config from:

- `FLOWGENTIC_CONFIG` env var, or
- `flowgentic.json` in current working directory

Example config (`flowgentic.json`):

```json
{
  "controlPlane": {
    "port": 8420,
    "tailscale": { "enabled": false },
    "embeddedWorker": { "enabled": true }
  },
  "worker": {
    "port": 8081,
    "tailscale": { "enabled": false }
  }
}
```

## Required Environment Variables

Worker requires:

- `FLOWGENTIC_WORKER_SECRET`: bearer token used to authenticate requests to worker public RPC services.

The `make run-worker` target sets a local dev value automatically (`dev-secret`).

## Common Make Targets

- `make build`: lint + build all binaries
- `make run-worker`: run worker with dev secret
- `make run-cp`: run control plane (auto-starts embedded worker)
- `make proto`: lint + regenerate protobuf code
- `make sqlc`: regenerate SQL access code
- `make clean`: remove generated proto + `bin/`

## Architecture

At a high level:

- Electron frontend calls control-plane APIs over Connect RPC.
- Control plane manages workers and relays requests to them.
- Worker runs agent workloads and exposes system/project/terminal/worker services.
- A session-scoped MCP server (`agentctl mcp serve`) is attached to agent sessions and communicates with worker on a private localhost control channel.

See `docs/architecture.md` for communication diagrams and package-level details.
