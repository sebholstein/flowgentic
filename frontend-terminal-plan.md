# Frontend Terminal Component — Gap Analysis & Changes

## Current State

`Terminal.tsx` wraps ghostty-web (xterm.js-compatible API) and exposes a `TerminalHandle` ref with `write`, `writeln`, `clear`, `focus`. It handles theming, FitAddon auto-resize, and user input via `onData` callback.

`AgentTerminal.tsx` is a demo wrapper that plays back hardcoded ANSI output — not relevant for the real backend integration.

## Backend Protocol Recap

| RPC | Purpose |
|-----|---------|
| `CreateSession(cwd, cols, rows, shell, env)` → `terminal_id` | Start PTY |
| `DestroySession(terminal_id)` | Kill PTY |
| `Resize(terminal_id, cols, rows)` | Change PTY dimensions |
| `Stream(bidi)` | `StreamRequest{terminal_id, bytes data}` ↔ `StreamResponse{oneof: bytes data \| int32 exit_code}` |

Key: all data is **raw bytes**, not UTF-8 strings.

---

## Required Changes to `Terminal.tsx`

### 1. Expose `cols`/`rows` for `CreateSession`

The backend needs initial dimensions. ghostty-web's Terminal exposes `term.cols` and `term.rows` after `open()` + `fit()`, but `TerminalHandle` doesn't expose them.

**Change:** Add `cols` and `rows` to `TerminalHandle`:

```typescript
export interface TerminalHandle {
  write: (data: string) => void;
  writeln: (data: string) => void;
  clear: () => void;
  focus: () => void;
  cols: number;   // NEW
  rows: number;   // NEW
}
```

Implementation reads from `terminalRef.current.cols` / `.rows` (both exist on the xterm.js-compatible API).

### 2. Add `onResize` callback for `Resize` RPC

When FitAddon resizes the terminal (browser window resize, panel drag, etc.), we need to send the new dimensions to the backend. ghostty-web fires a resize event via `terminal.onResize(({cols, rows}) => ...)`.

**Change:** Add `onResize` prop:

```typescript
interface TerminalProps {
  // ... existing props
  onResize?: (cols: number, rows: number) => void;  // NEW
}
```

Wire it up in the `setup()` function after `fitAddon.fit()`:

```typescript
if (onResize) {
  terminal.onResize(({ cols, rows }) => onResize(cols, rows));
}
```

### 3. Support `Uint8Array` writes for raw PTY output

The backend Stream sends `bytes data` (raw PTY output). The current `write(data: string)` works because ghostty-web's `Terminal.write()` accepts both `string` and `Uint8Array`. However, the `TerminalHandle` type only exposes `string`.

**Change:** Add a `writeBytes` method or widen the `write` signature:

```typescript
export interface TerminalHandle {
  write: (data: string | Uint8Array) => void;  // WIDENED
  // ...
}
```

This avoids a lossy `TextDecoder` round-trip for raw PTY bytes (which may contain partial UTF-8 sequences mid-chunk).

### 4. Add `onBinary` or ensure `onData` handles binary

User keystrokes go to the backend as `bytes data` in `StreamRequest`. The `onData` callback returns a `string`. For most keyboard input this is fine — the Connect client can encode it. But for binary sequences (paste with non-UTF-8, mouse protocol bytes), ghostty-web has `terminal.onBinary(callback)`.

**Recommendation:** Not needed for v1. `onData` returns strings which covers keyboard input. If binary paste issues arise later, add `onBinary` then.

### 5. Remove `initialContent` in connected mode

When connected to a real PTY, `initialContent` should not be used — the shell prompt comes from the backend stream. No code change needed, just don't pass the prop when using the live backend.

---

## No Changes Needed

| Aspect | Why it's fine |
|--------|---------------|
| FitAddon auto-resize | Already works; just need the `onResize` callback to forward dimensions |
| Theme support | Purely cosmetic, no backend interaction |
| Scrollback buffer | 10,000 lines is reasonable, PTY doesn't care |
| `onData` for input | Returns string which maps to `StreamRequest.data` bytes via `TextEncoder` |
| Cleanup on unmount | `dispose()` already tears down; parent hook will call `DestroySession` |
| Cursor style/blink | Cosmetic only |

---

## Summary of Changes

| # | What | Type | Effort |
|---|------|------|--------|
| 1 | Expose `cols`/`rows` on `TerminalHandle` | Add getter | Trivial |
| 2 | Add `onResize` prop + wire to `terminal.onResize()` | New prop | Small |
| 3 | Widen `write()` to accept `Uint8Array` | Type change | Trivial |
| 4 | _(deferred)_ `onBinary` for binary paste | — | — |

Total: ~15 lines of code changes to `Terminal.tsx`. The component is well-structured and ghostty-web's xterm.js-compatible API maps directly to our backend protocol.

The actual Connect RPC communication (hook that manages the stream lifecycle, calls CreateSession/DestroySession/Resize) is out of scope for this component — it belongs in a parent hook like `useTerminalSession`.
