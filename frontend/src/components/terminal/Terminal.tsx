import { useEffect, useRef, useImperativeHandle, forwardRef, useState } from "react";
import { init, Terminal as GhosttyTerminal, FitAddon } from "ghostty-web";

export interface TerminalHandle {
  write: (data: string | Uint8Array) => void;
  writeln: (data: string) => void;
  clear: () => void;
  focus: () => void;
  cols: number;
  rows: number;
}

interface TerminalProps {
  className?: string;
  onData?: (data: string) => void;
  onResize?: (cols: number, rows: number) => void;
  initialContent?: string;
  fontFamily?: string;
  fontSize?: number;
  theme?: "dark" | "light";
}

// Custom dark theme matching the app's design
const darkTheme = {
  background: "#0a0a0f",
  foreground: "#e4e4e7",
  cursor: "#e4e4e7",
  cursorAccent: "#0a0a0f",
  selectionBackground: "#3f3f46",
  selectionForeground: "#e4e4e7",
  black: "#18181b",
  red: "#ef4444",
  green: "#22c55e",
  yellow: "#eab308",
  blue: "#3b82f6",
  magenta: "#a855f7",
  cyan: "#06b6d4",
  white: "#e4e4e7",
  brightBlack: "#52525b",
  brightRed: "#f87171",
  brightGreen: "#4ade80",
  brightYellow: "#facc15",
  brightBlue: "#60a5fa",
  brightMagenta: "#c084fc",
  brightCyan: "#22d3ee",
  brightWhite: "#fafafa",
};

const lightTheme = {
  background: "#fafafa",
  foreground: "#18181b",
  cursor: "#18181b",
  cursorAccent: "#fafafa",
  selectionBackground: "#d4d4d8",
  selectionForeground: "#18181b",
  black: "#18181b",
  red: "#dc2626",
  green: "#16a34a",
  yellow: "#ca8a04",
  blue: "#2563eb",
  magenta: "#9333ea",
  cyan: "#0891b2",
  white: "#e4e4e7",
  brightBlack: "#71717a",
  brightRed: "#ef4444",
  brightGreen: "#22c55e",
  brightYellow: "#eab308",
  brightBlue: "#3b82f6",
  brightMagenta: "#a855f7",
  brightCyan: "#06b6d4",
  brightWhite: "#fafafa",
};

// Module-level initialization promise
let initPromise: Promise<void> | null = null;

async function ensureInit() {
  if (!initPromise) {
    initPromise = init();
  }
  return initPromise;
}

export const Terminal = forwardRef<TerminalHandle, TerminalProps>(function Terminal(
  {
    className,
    onData,
    onResize,
    initialContent,
    fontFamily = "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace",
    fontSize = 13,
    theme = "dark",
  },
  ref,
) {
  const containerRef = useRef<HTMLDivElement>(null);
  const terminalRef = useRef<GhosttyTerminal | null>(null);
  const [_isReady, setIsReady] = useState(false);

  useImperativeHandle(ref, () => ({
    write: (data: string | Uint8Array) => terminalRef.current?.write(data),
    writeln: (data: string) => terminalRef.current?.write(data + "\r\n"),
    clear: () => terminalRef.current?.clear(),
    focus: () => terminalRef.current?.focus(),
    get cols() {
      return terminalRef.current?.cols ?? 80;
    },
    get rows() {
      return terminalRef.current?.rows ?? 24;
    },
  }));

  useEffect(() => {
    if (!containerRef.current) return;

    let terminal: GhosttyTerminal | null = null;
    let fitAddon: FitAddon | null = null;
    let disposed = false;

    async function setup() {
      await ensureInit();

      if (disposed || !containerRef.current) return;

      const selectedTheme = theme === "dark" ? darkTheme : lightTheme;

      terminal = new GhosttyTerminal({
        fontFamily,
        fontSize,
        theme: selectedTheme,
        cursorBlink: true,
        cursorStyle: "bar",
        scrollback: 10000,
      });

      // Create and load FitAddon for auto-resizing
      fitAddon = new FitAddon();
      terminal.loadAddon(fitAddon);

      terminal.open(containerRef.current);
      terminalRef.current = terminal;

      // Fit terminal to container and observe for resize changes
      fitAddon.fit();
      fitAddon.observeResize();

      // Forward resize events to parent
      if (onResize) {
        terminal.onResize(({ cols, rows }) => onResize(cols, rows));
      }

      // Handle user input
      if (onData) {
        terminal.onData(onData);
      }

      // Write initial content
      if (initialContent) {
        terminal.write(initialContent);
      }

      setIsReady(true);
    }

    setup();

    return () => {
      disposed = true;
      fitAddon?.dispose();
      terminal?.dispose();
      terminalRef.current = null;
    };
  }, [fontFamily, fontSize, theme, onData, onResize, initialContent]);

  return (
    <div
      ref={containerRef}
      className={className}
      style={{
        width: "100%",
        height: "100%",
        padding: "8px",
        paddingRight: 0,
        boxSizing: "border-box",
        backgroundColor: theme === "dark" ? "#0a0a0f" : "#fafafa",
        display: "flex",
        justifyContent: "flex-end",
      }}
    />
  );
});
