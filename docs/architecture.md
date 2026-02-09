# Architecture: Communication Patterns

```mermaid
graph TB
    Electron["Electron App"]
    CP["Control Plane"]
    WorkerPub["Worker<br/>(Public Listener)"]
    WorkerCtl["Worker<br/>(Private CTL Listener)"]
    Agents["Agent Processes<br/>(claude, codex, gemini, ...)"]
    hookctl["hookctl"]
    agentctl["agentctl"]

    Electron -- "Connect RPC, no auth (yet)" --> CP
    CP -- "Connect RPC relay<br/>Authorization: Bearer" --> WorkerPub
    CP -. "process spawn / SIGTERM<br/>(embedded worker)" .-> WorkerPub
    WorkerPub -- "tmux / exec<br/>passes AGENTCTL_* env vars" --> Agents
    Agents -- "spawns on lifecycle hooks" --> hookctl
    Agents -. "user / script invokes" .-> agentctl
    hookctl -- "Connect RPC<br/>AGENTCTL_SECRET" --> WorkerCtl
    agentctl -- "Connect RPC<br/>AGENTCTL_SECRET" --> WorkerCtl

    style Electron fill:#f3e5f5,stroke:#6a1b9a
    style CP fill:#e3f2fd,stroke:#1565c0
    style WorkerPub fill:#e8f5e9,stroke:#2e7d32
    style WorkerCtl fill:#fff3e0,stroke:#e65100
    style Agents fill:#fce4ec,stroke:#b71c1c
    style hookctl fill:#f5f5f5,stroke:#616161
    style agentctl fill:#f5f5f5,stroke:#616161
```

| Connection | Transport | Auth | Network |
|---|---|---|---|
| Electron → Control Plane | Connect RPC | None (CORS) | localhost |
| Control Plane → Worker | Connect RPC (relay) | `Authorization: Bearer` header | Tailscale or localhost |
| Control Plane → Worker | Process mgmt (embedded) | Shared secret via env | localhost |
| Worker → Agents | tmux / exec | N/A (env var handoff) | localhost |
| hookctl → Worker CTL | Connect RPC | `AGENTCTL_SECRET` (ephemeral) | 127.0.0.1 only |
| agentctl → Worker CTL | Connect RPC | `AGENTCTL_SECRET` (ephemeral) | 127.0.0.1 only |

## Worker Internal Packages

```mermaid
graph LR
    Server["server/server.go"]
    Driver["driver/"]
    Workload["workload/"]
    AgentCtl["agentctl/"]
    SystemInfo["systeminfo/"]

    Server --> Driver
    Server --> Workload
    Server --> AgentCtl
    Server --> SystemInfo
    Workload --> Driver
    AgentCtl -->|"EventHandler interface"| Workload
```

- **`driver/`** — Unified interface for coding agents. Each sub-package (`claude`, `codex`, `gemini`, `opencode`) implements the same `Driver` interface, encapsulating agent-specific launch logic, event normalization, and session management. From the outside, all drivers speak the same language.
- **`workload/`** — The heart of the worker. `WorkloadManager` is responsible for spinning up agent sessions, reconciling desired state from the control plane, and keeping sessions alive. It uses `Driver` to launch sessions but owns the lifecycle.
- **`agentctl/`** — Agent-facing RPC service. Provides the Connect RPC endpoints that agent processes (via `cmd/agentctl` and `cmd/hookctl`) call into to report status, submit plans, and send hook events. Purely an inbound communication layer — delegates all logic to `WorkloadManager` via the `EventHandler` interface.
- **`systeminfo/`** — Agent discovery. Detects which coding agents are available on the system.

The internal flow is: **agentctl RPC → WorkloadManager → Driver**.
