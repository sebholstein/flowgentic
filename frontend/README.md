# Flowgentic Frontend

Desktop and browser UI for orchestrating agentic workflows. Built with React 19, TanStack Router, and Electron.

## Quick Start

```bash
pnpm install

# Browser development (port 3000)
pnpm dev:browser

# Electron desktop app
pnpm dev
```

## Scripts

| Command              | Description                         |
| -------------------- | ----------------------------------- |
| `pnpm dev:browser`   | Start Vite dev server on `:3000`    |
| `pnpm dev`           | Launch Electron app with hot reload |
| `pnpm build`         | Production build                    |
| `pnpm test`          | Run tests (Vitest)                  |
| `pnpm fmt`           | Format code (oxfmt)                 |
| `pnpm proto`         | Generate protobuf types             |
| `pnpm package`       | Package Electron app                |
| `pnpm electron:make` | Create distributable installers     |

## Tech Stack

- **Framework:** React 19, TypeScript (strict)
- **Routing:** TanStack Router (file-based)
- **Data fetching:** TanStack React Query
- **State management:** Zustand
- **Styling:** TailwindCSS v4, shadcn/ui (Radix primitives)
- **Icons:** Lucide React, Tabler Icons
- **Backend communication:** ConnectRPC + Protobuf
- **Desktop:** Electron 33 (via Electron Forge)
- **Bundler:** Vite
- **Terminal:** Ghostty Web

## Project Structure

```
src/
├── components/         # React components (see below)
├── routes/             # File-based routes (TanStack Router)
├── types/              # TypeScript type definitions
├── data/               # Mock data for development
├── hooks/              # Custom React hooks
├── stores/             # Zustand state stores
├── constants/          # Status configs, colors
├── lib/                # Utilities, ConnectRPC setup, query hooks
├── proto/              # Protobuf generated code
├── electron/           # Electron main & preload scripts
├── main.tsx            # App entry point
├── router.tsx          # Router configuration
└── styles.css          # Global styles & CSS variables
```

### Routes

| Route                          | Description                                                   |
| ------------------------------ | ------------------------------------------------------------- |
| `/`                            | Welcome / splash                                              |
| `/app`                         | Main dashboard                                                |
| `/app/threads`                 | Thread list                                                   |
| `/app/threads/new`             | Create thread                                                 |
| `/app/threads/$threadId`       | Thread detail (overview, tasks, check-ins, resources, memory) |
| `/app/tasks/$threadId/$taskId` | Task detail (overview, progress, context, changes, memory)    |
| `/app/settings`                | Application settings                                          |
| `/app/overseer`                | Overseer interface                                            |

### Components

| Directory          | Purpose                                              |
| ------------------ | ---------------------------------------------------- |
| `ui/`              | shadcn/ui primitives + custom components             |
| `threads/`         | Thread management, task details, feedback lists      |
| `chat/`            | Agent chat panels, thinking blocks, tool calls       |
| `inbox/`           | Feedback items, plan approvals, decision escalations |
| `code-review/`     | Diff viewer, file tree, comment threads              |
| `layout/`          | App sidebar, main layout, window chrome              |
| `terminal/`        | Integrated terminal (Ghostty)                        |
| `command-palette/` | Command palette                                      |
| `vcs/`             | Git/branch visualization                             |
| `resources/`       | Resource management                                  |
| `settings/`        | Settings UI                                          |
| `servers/`         | Server/control plane configuration                   |

## Domain Model

The core domain revolves around **Threads** and **Tasks**:

- **Thread** — A top-level unit of work with a mode (`plan` or `build`), optional model override, an overseer, resources, and VCS context. Contains one or more tasks.
- **Task** — An executable unit within a thread. Supports an optional planning phase (`pending` &rarr; `in_progress` &rarr; `awaiting_approval` &rarr; `approved`/`rejected`/`skipped`) before execution.
- **TaskExecution** — A single attempt at completing a task by an agent.
- **InboxItem** — A request for user attention. Types: `execution_selection`, `thread_review`, `planning_approval`, `task_plan_approval`, `questionnaire`, `decision_escalation`, `direction_clarification`.
- **Resource** — Scoped to a thread or task, with type, status, provenance, and content (files or URLs).

## Conventions

- **pnpm** as the package manager
- **No barrel files** — import directly from the source module
- **Path aliases** — `@/*` maps to `src/*`
- **Mock data** in `src/data/` for UI development
- **Dark/light theme** via CSS variables (OKLch color space), persisted in localStorage
