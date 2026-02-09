# Flowgentic

**Project**
- Go control-plane + worker system for running agent workloads over Connect RPC; optional embedded worker and Tailscale (tsnet) networking.


**Technologies**

- Frontend: React 19, Vite, Electron Forge, Tailwind v4, TanStack Router/Query, Zustand.

**File Overview**
- `cmd/`: Go entrypoints (`flowgentic-control-plane`, `flowgentic-worker`, `flowgentic-demo`).
- `internal/`: core packages (`controlplane`, `worker`, `agent`, `config`, `database`, `proto`, `tsnetutil`).
- `internal/proto/definitions/` + `internal/proto/gen/`: protobuf sources + generated Connect stubs.
- `frontend/`: Electron/React app, Vite configs, `src/` UI code.
- `bin/`: built Go binaries.
- `flowgentic.json` + `flowgentic.local.json`: runtime config.
- `yaak/`: API client collections.

**File Tree**
```text
.
├── AGENTS.md                 (this summary)
├── Makefile                  (build/run helpers)
├── flowgentic.json            (default config)
├── cmd/                      (Go entrypoints)
│   ├── flowgentic-control-plane/ (control plane main)
│   ├── flowgentic-worker/        (worker main)
│   └── flowgentic-demo/          (TUI demo)
├── internal/                 (core packages)
│   ├── agent/                (agent runner + TUI)
│   ├── config/               (config structs + loader)
│   ├── controlplane/         (CP features + server)
│   ├── database/             (SQLite + migrations)
│   ├── proto/                (protobuf + Connect)
│   │   ├── definitions/      (proto sources)
│   │   └── gen/              (generated code)
│   ├── tsnetutil/            (Tailscale helpers)
│   └── worker/               (worker features + server)
├── frontend/                 (Electron + React app)
├── bin/                      (built Go binaries)
└── yaak/                     (API client collections)
```

**Architecture Patterns**
- Feature package pattern: `feature_service.go` (business logic and additional `*_service.go` possible if it makes sense) + `feature.go` (StartDeps + Start) + `<proto>_handler.go` (Connect handler).
- `server/server.go` wires dependencies and starts each feature.
- Connect interceptors used for auth; RPC types isolated behind handlers.

**Components**
- Control plane: HTTP server with relay, ping, worker management, embedded worker; SQLite-backed worker store.
- Worker: exposes SystemService + WorkerService; requires `FLOWGENTIC_WORKER_SECRET`.
- Relay: allowlisted reverse proxy to worker services, routes by `X-Worker-Id`.
- Embedded worker: control plane can spawn and manage a worker process.
- Agent demo: `internal/agent` wraps the `claude` CLI; TUI renderer in `internal/agent/tui`.
