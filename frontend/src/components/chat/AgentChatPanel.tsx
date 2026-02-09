import { useState, useRef, useEffect, useCallback } from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import {
  Send,
  Bot,
  Loader2,
  X,
  Play,
  Pause,
  ImagePlus,
  Bell,
  Users,
  User,
  Folder,
  Cpu,
  ChevronsUpDown,
  Check,
} from "lucide-react";
import { getNameInitials } from "@/lib/names";
import { ToolCallBlock, type ToolCall, type ToolType, type ToolStatus } from "./ToolCallBlock";
import { ThinkingBlock, type ThinkingPhase } from "./ThinkingBlock";
import { InlineFeedbackCard } from "./InlineFeedbackCard";
import { ServerStatusDot } from "@/components/servers/ServerStatusDot";
import { Popover, PopoverTrigger, PopoverContent } from "@/components/ui/popover";
import {
  ClaudeIcon,
  CodexIcon,
  OpenCodeIcon,
  AmpIcon,
  GeminiIcon,
} from "@/components/icons/harness-icons";
import {
  Command,
  CommandInput,
  CommandList,
  CommandEmpty,
  CommandGroup,
  CommandItem,
} from "@/components/ui/command";
import type { InboxItem } from "@/types/inbox";
import type { Project } from "@/types/project";
import type { Worker } from "@/types/server";

interface BaseMessage {
  id: string;
  timestamp: string;
}

interface UserMessage extends BaseMessage {
  type: "user";
  content: string;
}

interface AgentMessage extends BaseMessage {
  type: "agent";
  content: string;
}

interface ToolMessage extends BaseMessage {
  type: "tool";
  tool: ToolCall;
}

interface ThinkingMessage extends BaseMessage {
  type: "thinking";
  thinking: ThinkingPhase;
}

type ChatMessage = UserMessage | AgentMessage | ToolMessage | ThinkingMessage;

export interface ChatTarget {
  type: "thread_overseer" | "task_agent";
  entityId: string;
  agentName: string;
  title: string;
  agentColor?: string;
  /** Current step being worked on (shown with spinner) */
  currentStep?: {
    name: string;
    current: number;
    total: number;
  };
}

interface AgentChatPanelProps {
  target: ChatTarget;
  onClose?: () => void;
  className?: string;
  /** Hide the header bar */
  hideHeader?: boolean;
  /** Enable simulation mode with tool calls and thinking */
  enableSimulation?: boolean;
  /** Pending feedback item to display inline */
  pendingFeedback?: InboxItem | null;
  /** Callback when feedback is submitted */
  onFeedbackSubmit?: (itemId: string, data: unknown) => void;
  /** Thread mode */
  threadMode?: "single_agent" | "orchestrated";
  /** Callback when mode changes */
  onModeChange?: (mode: "single_agent" | "orchestrated") => void;
  /** Selected model */
  threadModel?: string;
  /** Callback when model changes */
  onModelChange?: (model: string) => void;
  /** Current project */
  project?: Project;
  /** Available projects */
  projects?: Project[];
  /** Callback when project changes */
  onProjectChange?: (projectId: string) => void;
  /** Selected worker ID */
  workerId?: string;
  /** Available workers */
  workers?: Worker[];
  /** Callback when worker changes */
  onWorkerChange?: (workerId: string) => void;
  /** Selected harness */
  harness?: string;
  /** Callback when harness changes */
  onHarnessChange?: (harness: string) => void;
}

const availableModels = [
  { id: "default", name: "Default" },
  { id: "claude-opus-4", name: "Claude Opus 4" },
  { id: "claude-sonnet-4", name: "Claude Sonnet 4" },
  { id: "gpt-4", name: "GPT-4" },
  { id: "gpt-4o", name: "GPT-4o" },
  { id: "gemini-pro", name: "Gemini Pro" },
];

const availableHarnesses = [
  { id: "claude-code", name: "Claude Code", icon: ClaudeIcon },
  { id: "codex", name: "Codex", icon: CodexIcon },
  { id: "opencode", name: "OpenCode", icon: OpenCodeIcon },
  { id: "amp", name: "Amp", icon: AmpIcon },
  { id: "gemini", name: "Gemini", icon: GeminiIcon },
];

// Simulated agent workflow steps
interface WorkflowStep {
  type: "thinking" | "tool" | "message";
  delay: number;
  data: Partial<ThinkingPhase> | Partial<ToolCall> | { content: string };
}

const simulatedWorkflow: WorkflowStep[] = [
  {
    type: "thinking",
    delay: 3000,
    data: {
      content: `I need to set up the Stripe SDK for this project. Let me first understand the current project structure and check if there are any existing payment-related files.

Key things to consider:
1. Check package.json for existing dependencies
2. Look for any existing payment configuration
3. Determine the appropriate location for the new files
4. Follow the project's existing patterns for environment variables`,
    },
  },
  {
    type: "tool",
    delay: 800,
    data: {
      type: "read_file" as ToolType,
      title: "Read package.json",
      subtitle: "package.json",
      output: `{
  "name": "flowgentic-app",
  "dependencies": {
    "react": "^19.2.0",
    "@tanstack/react-router": "^1.15.0",
    "tailwindcss": "^4.0.0"
  }
}`,
      duration: 45,
    },
  },
  {
    type: "tool",
    delay: 600,
    data: {
      type: "list_directory" as ToolType,
      title: "List src/lib directory",
      subtitle: "src/lib/",
      output: `utils.ts
cn.ts
api.ts`,
      duration: 23,
    },
  },
  {
    type: "thinking",
    delay: 2500,
    data: {
      content: `Good, I can see the project structure. There's no existing Stripe setup. I'll need to:

1. Install the stripe and @stripe/stripe-js packages
2. Create src/lib/stripe.ts for server-side client
3. Create src/lib/stripe-client.ts for browser-side loader
4. Update .env.example with the required variables

Let me start by installing the dependencies.`,
    },
  },
  {
    type: "tool",
    delay: 2000,
    data: {
      type: "bash" as ToolType,
      title: "Install Stripe packages",
      subtitle: "npm install stripe @stripe/stripe-js",
      output: `added 2 packages in 1.8s

+ stripe@14.5.0
+ @stripe/stripe-js@2.4.0`,
      duration: 1823,
    },
  },
  {
    type: "tool",
    delay: 300,
    data: {
      type: "create_file" as ToolType,
      title: "Create server-side Stripe client",
      subtitle: "src/lib/stripe.ts",
      output: `Created file with 42 lines`,
      duration: 12,
    },
  },
  {
    type: "tool",
    delay: 300,
    data: {
      type: "create_file" as ToolType,
      title: "Create client-side Stripe loader",
      subtitle: "src/lib/stripe-client.ts",
      output: `Created file with 28 lines`,
      duration: 8,
    },
  },
  {
    type: "tool",
    delay: 200,
    data: {
      type: "edit_file" as ToolType,
      title: "Update environment template",
      subtitle: ".env.example",
      output: `Added 2 lines:
+ STRIPE_SECRET_KEY=sk_test_...
+ NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY=pk_test_...`,
      duration: 15,
    },
  },
  {
    type: "thinking",
    delay: 2000,
    data: {
      content: `Files created successfully. Now I should verify everything works by running the type checker and tests.`,
    },
  },
  {
    type: "tool",
    delay: 1500,
    data: {
      type: "bash" as ToolType,
      title: "Run type check",
      subtitle: "npm run typecheck",
      output: `✓ No type errors found`,
      duration: 1245,
    },
  },
  {
    type: "tool",
    delay: 1800,
    data: {
      type: "bash" as ToolType,
      title: "Run tests",
      subtitle: "npm test -- --grep stripe",
      output: `✓ stripe.test.ts › initializes Stripe client correctly (12ms)
✓ stripe.test.ts › handles missing API key gracefully (8ms)
✓ stripe-client.test.ts › loads Stripe.js asynchronously (45ms)

Tests: 3 passed, 3 total
Time: 0.892s`,
      duration: 1654,
    },
  },
  {
    type: "message",
    delay: 200,
    data: {
      content: `I've completed the Stripe SDK setup. Here's what I did:

**Installed packages:**
- \`stripe\` (v14.5.0) - Server-side SDK
- \`@stripe/stripe-js\` (v2.4.0) - Client-side loader

**Created files:**
- \`src/lib/stripe.ts\` - Server-side client with singleton pattern
- \`src/lib/stripe-client.ts\` - Lazy-loading browser client

**Updated:**
- \`.env.example\` - Added required environment variables

All type checks pass and tests are green. The implementation follows the project's existing patterns. Let me know if you'd like me to make any adjustments!`,
    },
  },
];

export function AgentChatPanel({
  target,
  onClose,
  className,
  hideHeader = false,
  enableSimulation = false,
  pendingFeedback,
  onFeedbackSubmit,
  threadMode,
  onModeChange,
  threadModel,
  onModelChange,
  project,
  projects,
  onProjectChange,
  workerId,
  workers,
  onWorkerChange,
  harness,
  onHarnessChange,
}: AgentChatPanelProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [inputValue, setInputValue] = useState("");
  const [isTyping, setIsTyping] = useState(false);
  const [isSimulating, setIsSimulating] = useState(false);
  const [simulationIndex, setSimulationIndex] = useState(0);
  const scrollRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const simulationRef = useRef<NodeJS.Timeout | null>(null);
  const streamingRef = useRef<NodeJS.Timeout | null>(null);

  const agentColor =
    target.agentColor ?? (target.type === "thread_overseer" ? "bg-violet-500" : "bg-orange-500");

  // Track session tokens (rough estimate: ~4 chars per token)
  const [sessionTokens, setSessionTokens] = useState(0);

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    requestAnimationFrame(() => {
      if (scrollRef.current) {
        scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
      }
    });
  }, [messages]);

  // Reset messages and focus when target changes
  useEffect(() => {
    setMessages([]);
    setSimulationIndex(0);
    setIsSimulating(false);
    setSessionTokens(0);
    if (simulationRef.current) clearTimeout(simulationRef.current);
    if (streamingRef.current) clearInterval(streamingRef.current);
    setTimeout(() => textareaRef.current?.focus(), 50);
  }, [target.entityId]);

  // Update session tokens when messages change
  useEffect(() => {
    const totalChars = messages.reduce((acc, msg) => {
      if (msg.type === "user" || msg.type === "agent") {
        return acc + msg.content.length;
      }
      if (msg.type === "thinking" && msg.thinking.content) {
        return acc + msg.thinking.content.length;
      }
      return acc;
    }, 0);
    setSessionTokens(Math.ceil(totalChars / 4));
  }, [messages]);

  // Run simulation step
  const runSimulationStep = useCallback(() => {
    if (simulationIndex >= simulatedWorkflow.length) {
      setIsSimulating(false);
      return;
    }

    const step = simulatedWorkflow[simulationIndex];
    const messageId = `msg-${Date.now()}`;
    const timestamp = new Date().toISOString();

    if (step.type === "thinking") {
      const thinkingData = step.data as Partial<ThinkingPhase>;
      const fullContent = thinkingData.content || "";
      let charIndex = 0;

      setMessages((prev) => [
        ...prev,
        {
          id: messageId,
          type: "thinking",
          timestamp,
          thinking: {
            id: messageId,
            status: "thinking",
            streamingContent: "",
          },
        },
      ]);

      const charsPerTick = 3;
      const tickInterval = 20;

      streamingRef.current = setInterval(() => {
        charIndex += charsPerTick;
        const currentContent = fullContent.slice(0, charIndex);

        setMessages((prev) =>
          prev.map((m) =>
            m.id === messageId && m.type === "thinking"
              ? { ...m, thinking: { ...m.thinking, streamingContent: currentContent } }
              : m,
          ),
        );

        if (charIndex >= fullContent.length) {
          if (streamingRef.current) clearInterval(streamingRef.current);
          setMessages((prev) =>
            prev.map((m) =>
              m.id === messageId && m.type === "thinking"
                ? {
                    ...m,
                    thinking: {
                      ...m.thinking,
                      status: "complete",
                      content: fullContent,
                      streamingContent: undefined,
                      duration: Math.floor(Math.random() * 500) + 200,
                    },
                  }
                : m,
            ),
          );
          setSimulationIndex((i) => i + 1);
        }
      }, tickInterval);
    } else if (step.type === "tool") {
      const toolData = step.data as Partial<ToolCall>;

      setMessages((prev) => [
        ...prev,
        {
          id: messageId,
          type: "tool",
          timestamp,
          tool: {
            id: messageId,
            type: toolData.type!,
            status: "running",
            title: toolData.title!,
            subtitle: toolData.subtitle,
          },
        },
      ]);

      simulationRef.current = setTimeout(() => {
        setMessages((prev) =>
          prev.map((m) =>
            m.id === messageId && m.type === "tool"
              ? {
                  ...m,
                  tool: {
                    ...m.tool,
                    status: "success" as ToolStatus,
                    output: toolData.output,
                    duration: toolData.duration,
                  },
                }
              : m,
          ),
        );
        setSimulationIndex((i) => i + 1);
      }, step.delay);
    } else if (step.type === "message") {
      const messageData = step.data as { content: string };
      setMessages((prev) => [
        ...prev,
        { id: messageId, type: "agent", timestamp, content: messageData.content },
      ]);
      setSimulationIndex((i) => i + 1);
    }
  }, [simulationIndex]);

  // Continue simulation
  useEffect(() => {
    if (isSimulating && simulationIndex < simulatedWorkflow.length) {
      simulationRef.current = setTimeout(runSimulationStep, simulationIndex === 0 ? 0 : 300);
    } else if (simulationIndex >= simulatedWorkflow.length) {
      setIsSimulating(false);
    }

    return () => {
      if (simulationRef.current) clearTimeout(simulationRef.current);
      if (streamingRef.current) clearInterval(streamingRef.current);
    };
  }, [isSimulating, simulationIndex, runSimulationStep]);

  const startSimulation = () => {
    setMessages([]);
    setSimulationIndex(0);
    setIsSimulating(true);
  };

  const pauseSimulation = () => {
    setIsSimulating(false);
    if (simulationRef.current) clearTimeout(simulationRef.current);
    if (streamingRef.current) clearInterval(streamingRef.current);
  };

  const handleSend = async () => {
    if (!inputValue.trim() || isSimulating) return;

    const userMessage: UserMessage = {
      id: `msg-${Date.now()}`,
      type: "user",
      content: inputValue.trim(),
      timestamp: new Date().toISOString(),
    };

    setMessages((prev) => [...prev, userMessage]);
    setInputValue("");
    setIsTyping(true);

    await new Promise((r) => setTimeout(r, 1000 + Math.random() * 1000));

    const agentResponse: AgentMessage = {
      id: `msg-${Date.now()}`,
      type: "agent",
      content: generateMockResponse(target.type),
      timestamp: new Date().toISOString(),
    };

    setMessages((prev) => [...prev, agentResponse]);
    setIsTyping(false);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      handleSend();
    }
  };

  const handleImageClick = () => fileInputRef.current?.click();

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (files && files.length > 0) {
      console.log("Selected files:", files);
      e.target.value = "";
    }
  };

  const getAgentInitials = () => {
    return target.type === "thread_overseer"
      ? getNameInitials(target.agentName)
      : target.agentName[0]?.toUpperCase();
  };

  return (
    <div className={cn("flex h-full flex-col", className)}>
      {/* Header */}
      {!hideHeader && (
        <div className="flex items-center gap-2 border-b px-3 py-2">
          {target.currentStep ? (
            <>
              <Loader2 className="size-3.5 animate-spin text-emerald-500 shrink-0" />
              <span className="text-xs text-muted-foreground">
                Step {target.currentStep.current}/{target.currentStep.total}
              </span>
              <span className="text-xs font-medium truncate flex-1">{target.currentStep.name}</span>
            </>
          ) : target.type === "thread_overseer" ? (
            <>
              <Bot className="size-4 text-violet-500 shrink-0" />
              <span className="text-xs font-medium flex-1">Overseer</span>
            </>
          ) : (
            <>
              <Avatar className={cn("h-6 w-6", agentColor)}>
                <AvatarFallback className="text-white text-[9px] font-medium">
                  {getAgentInitials()}
                </AvatarFallback>
              </Avatar>
              <div className="flex-1 min-w-0">
                <div className="text-xs font-medium truncate">{target.title}</div>
                <p className="text-[0.6rem] text-muted-foreground truncate">
                  Task Agent • {target.agentName}
                </p>
              </div>
            </>
          )}
          {pendingFeedback && (
            <Badge variant="outline" className="text-amber-400 border-amber-500/30 text-xs gap-1">
              <Bell className="size-3" />
              Feedback
            </Badge>
          )}
          {enableSimulation && (
            <Button
              variant="ghost"
              size="sm"
              onClick={isSimulating ? pauseSimulation : startSimulation}
              className="size-7 p-0 text-muted-foreground hover:text-foreground"
              title={isSimulating ? "Pause simulation" : "Start simulation"}
            >
              {isSimulating ? <Pause className="size-4" /> : <Play className="size-4" />}
            </Button>
          )}
          {onClose && (
            <Button
              variant="ghost"
              size="sm"
              onClick={onClose}
              className="size-7 p-0 text-muted-foreground hover:text-foreground"
            >
              <X className="size-4" />
            </Button>
          )}
        </div>
      )}

      {/* Messages */}
      <div
        className={cn(
          "flex-1 min-h-0 overflow-y-auto px-3",
          messages.length === 0 && !pendingFeedback && "flex flex-col",
        )}
        ref={scrollRef}
      >
        <div
          className={cn(
            "py-4 space-y-3",
            messages.length === 0 && !pendingFeedback && "flex-1 flex flex-col justify-center",
          )}
        >
          {/* Inline feedback card at top when present */}
          {pendingFeedback && (
            <InlineFeedbackCard
              inboxItem={pendingFeedback}
              onSubmit={(data) => onFeedbackSubmit?.(pendingFeedback.id, data)}
            />
          )}

          {messages.length === 0 && !pendingFeedback && (
            <div className="flex flex-col items-center text-center">
              <div className={cn("rounded-full p-3 mb-3", agentColor, "bg-opacity-10")}>
                <Bot className={cn("h-6 w-6", agentColor.replace("bg-", "text-"))} />
              </div>
              <p className="text-sm font-medium mb-1">
                {target.type === "thread_overseer" ? target.agentName : target.agentName}
              </p>
              <p className="text-xs text-muted-foreground max-w-[240px] mb-5">
                {target.type === "thread_overseer"
                  ? "Configure your thread and start a conversation."
                  : `Ask ${target.agentName} about this task.`}
              </p>

              {/* Thread config selectors */}
              {(onProjectChange ||
                onWorkerChange ||
                onModeChange ||
                onHarnessChange ||
                onModelChange) && (
                <div className="w-full max-w-[480px] grid grid-cols-2 gap-x-3 gap-y-3 mb-5 text-left">
                  {onProjectChange && project && projects && (
                    <div className="col-span-2">
                      <ConfigField label="Project">
                        <SearchableSelect
                          items={projects.map((p) => ({
                            id: p.id,
                            name: p.name,
                            icon: <Folder className={cn("size-3", p.color ?? "text-amber-400")} />,
                          }))}
                          selectedId={project.id}
                          onSelect={onProjectChange}
                          placeholder="Search projects…"
                        />
                      </ConfigField>
                    </div>
                  )}
                  {onModeChange && threadMode && (
                    <ConfigField label="Mode">
                      <SearchableSelect
                        items={[
                          {
                            id: "single_agent",
                            name: "Single Agent",
                            icon: <User className="size-3" />,
                          },
                          {
                            id: "orchestrated",
                            name: "Orchestrated",
                            icon: <Users className="size-3" />,
                          },
                        ]}
                        selectedId={threadMode}
                        onSelect={(id) => onModeChange(id as "single_agent" | "orchestrated")}
                        placeholder="Search modes…"
                      />
                    </ConfigField>
                  )}
                  {onWorkerChange && workerId && workers && workers.length > 1 && (
                    <ConfigField label="Worker">
                      <SearchableSelect
                        items={workers.map((w) => ({
                          id: w.id,
                          name: w.name,
                          icon: <Cpu className="size-3" />,
                          trailing: <ServerStatusDot status={w.status} />,
                        }))}
                        selectedId={workerId}
                        onSelect={onWorkerChange}
                        placeholder="Search workers…"
                      />
                    </ConfigField>
                  )}
                  {onHarnessChange && harness && (
                    <ConfigField label="Harness">
                      <SearchableSelect
                        items={availableHarnesses.map((h) => ({
                          id: h.id,
                          name: h.name,
                          icon: <h.icon className="size-3" />,
                        }))}
                        selectedId={harness}
                        onSelect={onHarnessChange}
                        placeholder="Search harnesses…"
                      />
                    </ConfigField>
                  )}
                  {onModelChange && threadModel && (
                    <ConfigField label="Model">
                      {harness === "claude-code" ? (
                        <input
                          type="text"
                          value={threadModel}
                          onChange={(e) => onModelChange(e.target.value)}
                          placeholder="e.g. claude-sonnet-4-20250514"
                          className="h-7 w-full rounded-md border border-input bg-input/20 dark:bg-input/30 px-2 text-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                        />
                      ) : (
                        <SearchableSelect
                          items={availableModels.map((m) => ({
                            id: m.id,
                            name: m.name,
                            icon: <Bot className="size-3" />,
                          }))}
                          selectedId={threadModel}
                          onSelect={onModelChange}
                          placeholder="Search models…"
                        />
                      )}
                    </ConfigField>
                  )}
                </div>
              )}

              {enableSimulation && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={startSimulation}
                  className="text-xs h-7"
                >
                  <Play className="size-3 mr-1" />
                  Start Demo
                </Button>
              )}
            </div>
          )}

          {messages.map((message) => {
            if (message.type === "user") {
              return (
                <div key={message.id} className="flex gap-2 flex-row-reverse">
                  <Avatar className="h-6 w-6 shrink-0 bg-primary">
                    <AvatarFallback className="text-white text-[9px] font-medium">U</AvatarFallback>
                  </Avatar>
                  <div className="rounded-lg px-2.5 py-1.5 max-w-[85%] text-xs bg-primary text-primary-foreground">
                    {message.content}
                  </div>
                </div>
              );
            }

            if (message.type === "agent") {
              return (
                <div key={message.id} className="flex gap-2">
                  <Avatar className={cn("h-6 w-6 shrink-0", agentColor)}>
                    <AvatarFallback className="text-white text-[9px] font-medium">
                      {getAgentInitials()}
                    </AvatarFallback>
                  </Avatar>
                  <div className="rounded-lg px-2.5 py-1.5 max-w-[85%] text-xs bg-muted whitespace-pre-wrap">
                    {message.content}
                  </div>
                </div>
              );
            }

            if (message.type === "tool") {
              return <ToolCallBlock key={message.id} tool={message.tool} />;
            }

            if (message.type === "thinking") {
              return <ThinkingBlock key={message.id} thinking={message.thinking} />;
            }

            return null;
          })}

          {isTyping && (
            <div className="flex gap-2">
              <Avatar className={cn("h-6 w-6 shrink-0", agentColor)}>
                <AvatarFallback className="text-white text-[9px] font-medium">
                  {getAgentInitials()}
                </AvatarFallback>
              </Avatar>
              <div className="rounded-lg bg-muted px-2.5 py-1.5">
                <div className="flex items-center gap-1">
                  <span
                    className="h-1.5 w-1.5 rounded-full bg-muted-foreground/40 animate-bounce"
                    style={{ animationDelay: "0ms" }}
                  />
                  <span
                    className="h-1.5 w-1.5 rounded-full bg-muted-foreground/40 animate-bounce"
                    style={{ animationDelay: "150ms" }}
                  />
                  <span
                    className="h-1.5 w-1.5 rounded-full bg-muted-foreground/40 animate-bounce"
                    style={{ animationDelay: "300ms" }}
                  />
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Input */}
      <div className="border-t px-3 py-2">
        {/* Session token count */}
        <div className="flex justify-end mb-0.5">
          <span className="text-[9px] text-muted-foreground/60">
            ~{sessionTokens.toLocaleString()} tokens this session
          </span>
        </div>

        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          onChange={handleFileChange}
          className="hidden"
          multiple
        />

        {/* Textarea with buttons inside */}
        <div className="relative rounded-lg border border-input bg-background dark:bg-muted/50 focus-within:ring-1 focus-within:ring-ring">
          <textarea
            ref={textareaRef}
            placeholder={`Message ${target.agentName}... (⌘+Enter to send)`}
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyDown={handleKeyDown}
            disabled={isTyping || isSimulating}
            rows={4}
            className="w-full resize-none bg-transparent px-3 pt-2 pb-10 text-xs placeholder:text-muted-foreground focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
          />
          <div className="absolute bottom-2 right-2 flex items-center gap-1">
            <Button
              variant="ghost"
              size="icon"
              className="h-7 w-7"
              onClick={handleImageClick}
              disabled={isTyping || isSimulating}
              title="Attach image"
            >
              <ImagePlus className="h-3.5 w-3.5 text-muted-foreground" />
            </Button>
            <Button
              size="icon"
              className="h-7 w-7"
              onClick={handleSend}
              disabled={!inputValue.trim() || isTyping || isSimulating}
            >
              {isTyping ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <Send className="h-3.5 w-3.5" />
              )}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}

function ConfigField({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <span className="text-[0.6rem] font-medium uppercase tracking-wider text-muted-foreground/60">
        {label}
      </span>
      {children}
    </div>
  );
}

interface SearchableSelectItem {
  id: string;
  name: string;
  icon?: React.ReactNode;
  trailing?: React.ReactNode;
}

function SearchableSelect({
  items,
  selectedId,
  onSelect,
  placeholder = "Search…",
}: {
  items: SearchableSelectItem[];
  selectedId: string;
  onSelect: (id: string) => void;
  placeholder?: string;
}) {
  const [open, setOpen] = useState(false);
  const selected = items.find((item) => item.id === selectedId);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="w-full h-7 justify-between text-xs font-normal bg-input/20 dark:bg-input/30 border-input px-2"
        >
          {selected ? (
            <span className="flex items-center gap-2 truncate">
              {selected.icon}
              <span className="truncate">{selected.name}</span>
              {selected.trailing}
            </span>
          ) : (
            <span className="text-muted-foreground">{placeholder}</span>
          )}
          <ChevronsUpDown className="ml-auto size-3 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[--radix-popover-trigger-width] p-0">
        <Command className="p-0">
          <CommandInput placeholder={placeholder} className="h-7 text-xs" />
          <CommandList>
            <CommandEmpty>No results.</CommandEmpty>
            <CommandGroup className="p-1">
              {items.map((item) => (
                <CommandItem
                  key={item.id}
                  value={item.id}
                  keywords={[item.name]}
                  onSelect={(val) => {
                    onSelect(val);
                    setOpen(false);
                  }}
                  className="py-1 px-2 gap-1.5"
                >
                  <Check
                    className={cn(
                      "size-3 shrink-0",
                      selectedId === item.id ? "opacity-100" : "opacity-0",
                    )}
                  />
                  {item.icon}
                  <span className="flex-1 truncate">{item.name}</span>
                  {item.trailing}
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}

function generateMockResponse(agentType: "thread_overseer" | "task_agent"): string {
  const overseerResponses = [
    `Based on my analysis of the current thread progress, I can see that we're making good headway. The team has been focused on the core functionality.`,
    `I've reviewed the recent task completions and they align well with our objectives. There are a few areas where we might need to adjust our approach.`,
    `That's a great question. From my perspective overseeing this thread, the key challenge right now is ensuring proper coordination between dependent tasks.`,
    `I understand your concern. Let me check the status of the related tasks and provide a more detailed assessment.`,
    `Looking at the bigger picture for this thread, I think we're in a good position. The completed tasks have laid a solid foundation.`,
  ];

  const taskAgentResponses = [
    `I've been working on this task and can provide some insights. The implementation approach focuses on maintainability and performance.`,
    `Good question! In my execution of this task, I considered several approaches before settling on the current solution.`,
    `The output I generated addresses the core requirements of the task. I followed the established conventions in the codebase.`,
    `I can explain my reasoning here. The decisions were based on the task requirements and context from the thread overseer.`,
    `During my work on this task, I identified a few edge cases that might need additional attention.`,
  ];

  const responses = agentType === "thread_overseer" ? overseerResponses : taskAgentResponses;
  return responses[Math.floor(Math.random() * responses.length)];
}
