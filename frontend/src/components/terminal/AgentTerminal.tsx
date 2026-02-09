import { useEffect, useRef, useState, useCallback } from "react";
import { Terminal, type TerminalHandle } from "./Terminal";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Play,
  Pause,
  RotateCcw,
  Maximize2,
  Minimize2,
  Terminal as TerminalIcon,
  Circle,
} from "lucide-react";

interface AgentTerminalProps {
  className?: string;
  agentName?: string;
  taskName?: string;
  status?: "idle" | "running" | "completed" | "failed";
  onMaximize?: () => void;
  isMaximized?: boolean;
}

// ANSI escape codes for colors
const ANSI = {
  reset: "\x1b[0m",
  bold: "\x1b[1m",
  dim: "\x1b[2m",
  red: "\x1b[31m",
  green: "\x1b[32m",
  yellow: "\x1b[33m",
  blue: "\x1b[34m",
  magenta: "\x1b[35m",
  cyan: "\x1b[36m",
  white: "\x1b[37m",
  brightBlack: "\x1b[90m",
  brightGreen: "\x1b[92m",
  brightYellow: "\x1b[93m",
  brightCyan: "\x1b[96m",
  bgBlue: "\x1b[44m",
  bgGreen: "\x1b[42m",
};

// Simulated tmux-style output showing agent progress
const demoOutputSequence = [
  // Tmux header
  `${ANSI.bgBlue}${ANSI.white}${ANSI.bold} claude-opus-4 │ stripe-setup │ flowgentic ${ANSI.reset}\r\n`,
  `${ANSI.brightBlack}────────────────────────────────────────────────────────────────${ANSI.reset}\r\n\r\n`,

  // Agent starting
  `${ANSI.cyan}▶${ANSI.reset} ${ANSI.bold}Starting task:${ANSI.reset} Stripe SDK Setup\r\n`,
  `${ANSI.brightBlack}  Agent: claude-opus-4 | Model: claude-opus-4-5-20251101${ANSI.reset}\r\n\r\n`,

  // Step 1: Reading package.json
  `${ANSI.yellow}◐${ANSI.reset} Reading project configuration...\r\n`,
  `${ANSI.brightBlack}  → cat package.json${ANSI.reset}\r\n`,
  `${ANSI.green}✓${ANSI.reset} Found package.json with dependencies\r\n\r\n`,

  // Step 2: Installing Stripe
  `${ANSI.yellow}◐${ANSI.reset} Installing Stripe SDK...\r\n`,
  `${ANSI.brightBlack}  → pnpm add stripe @stripe/stripe-js${ANSI.reset}\r\n`,
  `${ANSI.dim}Packages: +2${ANSI.reset}\r\n`,
  `${ANSI.dim}++${ANSI.reset}\r\n`,
  `${ANSI.dim}dependencies:${ANSI.reset}\r\n`,
  `${ANSI.dim}+ stripe 17.7.0${ANSI.reset}\r\n`,
  `${ANSI.dim}+ @stripe/stripe-js 5.7.0${ANSI.reset}\r\n`,
  `${ANSI.green}✓${ANSI.reset} Dependencies installed\r\n\r\n`,

  // Step 3: Creating config file
  `${ANSI.yellow}◐${ANSI.reset} Creating Stripe configuration...\r\n`,
  `${ANSI.brightBlack}  → Write src/lib/stripe.ts${ANSI.reset}\r\n`,
  `${ANSI.dim}import Stripe from 'stripe';${ANSI.reset}\r\n`,
  `${ANSI.dim}${ANSI.reset}\r\n`,
  `${ANSI.dim}export const stripe = new Stripe(${ANSI.reset}\r\n`,
  `${ANSI.dim}  process.env.STRIPE_SECRET_KEY!,${ANSI.reset}\r\n`,
  `${ANSI.dim}  { apiVersion: '2025-01-27' }${ANSI.reset}\r\n`,
  `${ANSI.dim});${ANSI.reset}\r\n`,
  `${ANSI.green}✓${ANSI.reset} Created src/lib/stripe.ts\r\n\r\n`,

  // Step 4: Creating client config
  `${ANSI.yellow}◐${ANSI.reset} Creating client-side configuration...\r\n`,
  `${ANSI.brightBlack}  → Write src/lib/stripe-client.ts${ANSI.reset}\r\n`,
  `${ANSI.dim}import { loadStripe } from '@stripe/stripe-js';${ANSI.reset}\r\n`,
  `${ANSI.dim}${ANSI.reset}\r\n`,
  `${ANSI.dim}export const stripePromise = loadStripe(${ANSI.reset}\r\n`,
  `${ANSI.dim}  process.env.NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY!${ANSI.reset}\r\n`,
  `${ANSI.dim});${ANSI.reset}\r\n`,
  `${ANSI.green}✓${ANSI.reset} Created src/lib/stripe-client.ts\r\n\r\n`,

  // Step 5: Environment variables
  `${ANSI.yellow}◐${ANSI.reset} Adding environment variables template...\r\n`,
  `${ANSI.brightBlack}  → Edit .env.example${ANSI.reset}\r\n`,
  `${ANSI.dim}STRIPE_SECRET_KEY=sk_test_...${ANSI.reset}\r\n`,
  `${ANSI.dim}NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_test_...${ANSI.reset}\r\n`,
  `${ANSI.dim}STRIPE_WEBHOOK_SECRET=whsec_...${ANSI.reset}\r\n`,
  `${ANSI.green}✓${ANSI.reset} Updated .env.example\r\n\r\n`,

  // Task complete
  `${ANSI.brightBlack}────────────────────────────────────────────────────────────────${ANSI.reset}\r\n`,
  `${ANSI.bgGreen}${ANSI.white}${ANSI.bold} COMPLETED ${ANSI.reset} ${ANSI.green}Task finished successfully${ANSI.reset}\r\n`,
  `${ANSI.brightBlack}  Duration: 1m 05s | Tokens: 12,450 (in: 8,200, out: 4,250)${ANSI.reset}\r\n\r\n`,

  // Summary
  `${ANSI.cyan}${ANSI.bold}Summary:${ANSI.reset}\r\n`,
  `${ANSI.green}✓${ANSI.reset} Installed stripe and @stripe/stripe-js packages\r\n`,
  `${ANSI.green}✓${ANSI.reset} Created server-side Stripe client (src/lib/stripe.ts)\r\n`,
  `${ANSI.green}✓${ANSI.reset} Created client-side Stripe loader (src/lib/stripe-client.ts)\r\n`,
  `${ANSI.green}✓${ANSI.reset} Added environment variable templates\r\n\r\n`,

  `${ANSI.brightBlack}Ready for next task: Payment Intent API${ANSI.reset}\r\n`,
];

export function AgentTerminal({
  className,
  agentName = "claude-opus-4",
  taskName = "Stripe SDK Setup",
  status = "running",
  onMaximize,
  isMaximized = false,
}: AgentTerminalProps) {
  const terminalRef = useRef<TerminalHandle>(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentLine, setCurrentLine] = useState(0);
  const [isReady, setIsReady] = useState(false);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Wait for terminal to be ready before starting playback
  useEffect(() => {
    const checkReady = setInterval(() => {
      if (terminalRef.current) {
        setIsReady(true);
        setIsPlaying(true);
        clearInterval(checkReady);
      }
    }, 100);

    return () => clearInterval(checkReady);
  }, []);

  const writeNextLine = useCallback(() => {
    if (currentLine >= demoOutputSequence.length) {
      setIsPlaying(false);
      return;
    }

    if (!terminalRef.current) return;

    terminalRef.current.write(demoOutputSequence[currentLine]);
    setCurrentLine((prev) => prev + 1);

    // Variable delay for realistic feel
    const delay = Math.random() * 150 + 50;
    timeoutRef.current = setTimeout(writeNextLine, delay);
  }, [currentLine]);

  useEffect(() => {
    if (isReady && isPlaying && currentLine < demoOutputSequence.length) {
      timeoutRef.current = setTimeout(writeNextLine, 100);
    }

    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
    };
  }, [isReady, isPlaying, writeNextLine, currentLine]);

  const handlePlayPause = () => {
    setIsPlaying(!isPlaying);
  };

  const handleReset = () => {
    terminalRef.current?.clear();
    setCurrentLine(0);
    setIsPlaying(true);
  };

  const statusColor =
    status === "running"
      ? "text-blue-400"
      : status === "completed"
        ? "text-emerald-400"
        : status === "failed"
          ? "text-red-400"
          : "text-muted-foreground";

  return (
    <div className={cn("flex flex-col h-full w-full bg-[#0a0a0f] overflow-hidden", className)}>
      {/* Terminal header */}
      <div className="flex items-center justify-between px-3 py-2 border-b border-zinc-800 bg-zinc-900/50">
        <div className="flex items-center gap-2">
          <TerminalIcon className="size-4 text-muted-foreground" />
          <span className="text-sm font-medium">{taskName}</span>
          <Badge variant="outline" className="text-xs">
            {agentName}
          </Badge>
          <Circle
            className={cn(
              "size-2 fill-current",
              statusColor,
              status === "running" && "animate-pulse",
            )}
          />
        </div>
        <div className="flex items-center gap-1">
          <Button variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={handlePlayPause}>
            {isPlaying ? <Pause className="size-3.5" /> : <Play className="size-3.5" />}
          </Button>
          <Button variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={handleReset}>
            <RotateCcw className="size-3.5" />
          </Button>
          {onMaximize && (
            <Button variant="ghost" size="sm" className="h-7 w-7 p-0" onClick={onMaximize}>
              {isMaximized ? (
                <Minimize2 className="size-3.5" />
              ) : (
                <Maximize2 className="size-3.5" />
              )}
            </Button>
          )}
        </div>
      </div>

      {/* Terminal content */}
      <div className="flex-1 min-h-0 w-full">
        <Terminal ref={terminalRef} theme="dark" className="h-full w-full" />
      </div>
    </div>
  );
}
