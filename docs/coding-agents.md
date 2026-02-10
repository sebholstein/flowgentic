# Coding Agent CLI Reference

Capabilities and behavior of each supported coding agent CLI, focused on what matters for the flowgentic driver integration.

## Claude Code

**Binary:** `claude`

### Session Management
- **Set ID on new session:** `--session-id <uuid>` — accepts any valid UUID, creates a new session if it doesn't exist.
- **Resume:** `--resume --session-id <uuid>` or `--resume` (interactive picker) or `--continue` (most recent in cwd).
- **List sessions:** No CLI command. Sessions stored in `~/.claude/projects/`.

### Interactive Mode (TUI)
- **Initial prompt:** Positional argument `claude [options] [prompt]` — starts the TUI and immediately sends the prompt.
- **Flags:** `--model`, `--system-prompt`, `--allow-dangerously-skip-permissions`, `--add-dir`, `--session-id`.
- **Prompt for interactive continuation:** `-i` / `--prompt-interactive` does **not** exist. Use the positional argument.

### Headless Mode
- **Flags:** `-p` / `--print` enables non-interactive mode. `--output-format stream-json` for JSONL streaming.
- **Prompt delivery:** Via stdin or positional argument.
- **Streaming output:** JSONL on stdout with message deltas, tool use events, cost info.

### Hooks
- Supports lifecycle hooks (`Stop`, `SessionStart`, `PreToolUse`, `PostToolUse`) configured via `~/.claude/hooks.json` or `CLAUDE.md`.
- Hook processes receive context as JSON via environment variables.

### Session ID Strategy
We generate a UUID and pass it via `--session-id` at launch. The ID is known before the session starts — no post-launch discovery needed.

---

## OpenAI Codex

**Binary:** `codex`

### Session Management
- **Set ID on new session:** Not supported. Codex assigns its own UUIDs internally.
- **Resume:** `codex resume <uuid>` or `codex resume --last` or `codex exec resume <uuid> "prompt"`.
- **List sessions:** No CLI command. The `codex resume` command opens an interactive picker. Sessions stored as JSONL files in `~/.codex/sessions/YYYY/MM/DD/rollout-<timestamp>-<uuid>.jsonl`.
- **App-server API:** Codex has a full app-server (`codex app-server`) with JSON-RPC endpoints: `thread/list`, `thread/start`, `thread/resume`, `thread/fork`, `thread/read`. This provides programmatic session management but requires running the server separately.

### Interactive Mode (TUI)
- **Initial prompt:** Positional argument `codex [options] [PROMPT]`.
- **Flags:** `--model`, `--full-auto`, `-C` / `--cd`, `--add-dir`, `-s` / `--sandbox`.
- **Yolo mode:** `--full-auto` (sandbox + auto-approve) or `--dangerously-bypass-approvals-and-sandbox`.

### Headless Mode
- **Command:** `codex exec [options] [PROMPT]`.
- **Streaming output:** `--json` flag for JSONL on stdout. First event is `session_meta` containing `{"id": "<uuid>"}`.
- **Resume in headless:** `codex exec resume <uuid> "prompt"`.

### Hooks
- No built-in hook system equivalent to Claude's. Integration relies on parsing JSONL output or using the app-server API.

### Session ID Strategy
- **Headless:** Parse `session_meta` event from `--json` stream to get the UUID.
- **Interactive:** Scan `~/.codex/sessions/` filesystem for the newest file after launch (requires scheduling lock to avoid races).

---

## Google Gemini CLI

**Binary:** `gemini`

### Session Management
- **Set ID on new session:** Not supported. Gemini assigns its own UUIDs internally.
- **Resume:** `--resume <uuid>` or `--resume <index>` or `--resume latest`.
- **List sessions:** `gemini --list-sessions` — displays sessions with index, description, age, and **unique UUID**.

### Interactive Mode (TUI)
- **Initial prompt (stay interactive):** `-i` / `--prompt-interactive "prompt"` — executes the prompt and continues in interactive mode. This is the preferred flag for our use case.
- **Initial prompt (positional):** `gemini [query..]` — also works as a positional argument.
- **Flags:** `--model`, `--yolo` / `--approval-mode`, `--include-directories`.

### Headless Mode
- **Flags:** `-p` / `--prompt "text"` for non-interactive mode.
- **Streaming output:** `--output-format stream-json` for JSONL streaming, `--output-format json` for single JSON result.

### Hooks
- Supports hooks via `gemini hooks` subcommand. `AfterAgent` hook is used for turn-complete signaling.

### Session ID Strategy
- **After launch:** Run `gemini --list-sessions` and parse the output to find the new session UUID (requires scheduling lock to avoid races).
- **Headless:** Stream output may contain session info (needs verification).

---

## OpenCode

**Binary:** `opencode`

### Session Management
- **Set ID on new session:** `-s` / `--session <id>` — works for both new and existing sessions.
- **Resume:** `-s <id> --continue` or `-c` / `--continue` (most recent).
- **List sessions:** `opencode session list --format json` — programmatic JSON output with session IDs.

### Interactive Mode (TUI)
- **Initial prompt:** `--prompt "text"` flag.
- **Flags:** `--model`, `--agent`, `-s` / `--session`.
- **Note:** No positional prompt argument. The positional arg is `[project]` (path to start in).

### Headless Mode
- **Command:** `opencode run [message..]` for non-interactive execution.
- **Server mode:** `opencode serve` starts a headless server with HTTP/SSE API.
- **Prompt delivery (server):** `POST /session/prompt` with JSON body.

### Hooks
- No built-in hook system. Integration via SSE event stream in server mode.

### Session ID Strategy
We generate an ID and pass it via `-s` / `--session` at launch. The ID is known before the session starts — no post-launch discovery needed. Can verify via `opencode session list --format json`.

---

## Comparison Matrix

| Capability | Claude | Codex | Gemini | OpenCode |
|---|---|---|---|---|
| Set session ID at launch | `--session-id <uuid>` | No | No | `-s <id>` |
| Resume by ID | `--resume --session-id` | `codex resume <uuid>` | `--resume <uuid>` | `-s <id> --continue` |
| List sessions (CLI) | No | No (filesystem scan only) | `--list-sessions` | `session list --format json` |
| Initial prompt (interactive) | Positional arg | Positional arg | `-i "prompt"` | `--prompt "text"` |
| Initial prompt (headless) | stdin or positional | Positional arg | `-p "prompt"` | `opencode run [msg]` |
| Streaming output (headless) | `--output-format stream-json` | `--json` (JSONL) | `--output-format stream-json` | SSE via `opencode serve` |
| Yolo / auto-approve | `--allow-dangerously-skip-permissions` | `--full-auto` | `--yolo` | N/A |
| Hook system | Yes (lifecycle hooks) | No (parse JSONL) | Yes (`gemini hooks`) | No (SSE events) |
| Session ID discovery | Not needed (set upfront) | Parse `session_meta` from JSONL or filesystem scan | `--list-sessions` output | Not needed (set upfront) |

## Session ID Strategy Summary

For reliable session tracking across concurrent workloads:

1. **Claude & OpenCode:** Generate a UUID, pass at launch via `--session-id` / `-s`. ID is known immediately — stored in `SessionInfo.AgentSessionID`.
2. **Codex & Gemini:** Cannot set ID upfront. These drivers implement `SessionResolver` for post-launch discovery. The `AgentRunManager` uses per-cwd mutexes to serialize launches and avoid races during discovery.
   - **Codex headless:** Parse `session_meta` event from the first JSONL line (contains `{"id":"<uuid>"}`).
   - **Codex interactive:** `SessionResolver` scans `~/.codex/sessions/YYYY/MM/DD/` for the newest `rollout-<timestamp>-<uuid>.jsonl` file.
   - **Gemini:** `SessionResolver` runs `gemini --list-sessions` and parses the most recent session UUID from the output.

The resolved `AgentSessionID` is returned in the `ScheduleWorkloadResponse` proto message for future resume support.
