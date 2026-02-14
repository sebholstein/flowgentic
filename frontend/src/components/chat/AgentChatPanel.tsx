import { useState, useRef, useEffect } from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Bot, Loader2, X, ImagePlus, Bell, ArrowUp } from "lucide-react";
import { ToolCallBlock } from "./ToolCallBlock";
import { McpToolCallBlock } from "./McpToolCallBlock";
import { ThinkingBlock } from "./ThinkingBlock";
import { InlineFeedbackCard } from "./InlineFeedbackCard";
import { Markdown } from "@/components/ui/markdown";
import type { InboxItem } from "@/types/inbox";
import type { ChatMessage } from "@/lib/session-event-mapper";

export type { ChatMessage };

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
  /** Whether to render the setup/config form when chat is empty */
  showSetupOnEmpty?: boolean;
  /** Optional content rendered inside empty state */
  emptyStateContent?: React.ReactNode;
  /** Pending feedback item to display inline */
  pendingFeedback?: InboxItem | null;
  /** Callback when feedback is submitted */
  onFeedbackSubmit?: (itemId: string, data: unknown) => void;
  /** Callback when user sends a message — if provided, replaces mock response logic */
  onSend?: (message: string) => void;
  /** External messages from streaming hook — when provided, replaces internal state */
  externalMessages?: ChatMessage[];
  /** Streaming agent text (not yet finalized into a message) */
  pendingAgentText?: string;
  /** Streaming thought text (not yet finalized into a message) */
  pendingThoughtText?: string;
  /** Whether the stream is actively connected / producing output */
  isStreaming?: boolean;
}

export function AgentChatPanel({
  target,
  onClose,
  className,
  hideHeader = false,
  showSetupOnEmpty = true,
  emptyStateContent,
  pendingFeedback,
  onFeedbackSubmit,
  onSend,
  externalMessages,
  pendingAgentText = "",
  pendingThoughtText = "",
  isStreaming = false,
}: AgentChatPanelProps) {
  const [internalMessages, setInternalMessages] = useState<ChatMessage[]>([]);
  const [inputValue, setInputValue] = useState("");
  const [isTyping, setIsTyping] = useState(false);
  const scrollRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Use external messages when provided, otherwise fall back to internal state
  const messages = externalMessages ?? internalMessages;

  // Track session tokens (rough estimate: ~4 chars per token)
  const [sessionTokens, setSessionTokens] = useState(0);

  // Auto-scroll to bottom when new messages arrive or pending text changes
  useEffect(() => {
    requestAnimationFrame(() => {
      if (scrollRef.current) {
        scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
      }
    });
  }, [messages, pendingAgentText, pendingThoughtText]);

  // Reset messages and focus when target changes
  useEffect(() => {
    if (!externalMessages) {
      setInternalMessages([]);
    }
    setSessionTokens(0);
    setTimeout(() => textareaRef.current?.focus(), 50);
  }, [target.entityId, externalMessages]);

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

  const handleSend = async () => {
    if (!inputValue.trim()) return;

    const text = inputValue.trim();
    const userMessage: ChatMessage = {
      id: `msg-${Date.now()}`,
      type: "user",
      content: text,
      timestamp: new Date().toISOString(),
    };

    if (!externalMessages) {
      setInternalMessages((prev) => [...prev, userMessage]);
    }
    setInputValue("");

    if (onSend) {
      onSend(text);
      return;
    }

    // Fallback: no-op when no onSend provided and using external messages
    if (externalMessages) return;

    setIsTyping(true);
    await new Promise((r) => setTimeout(r, 1000 + Math.random() * 1000));
    const agentResponse: ChatMessage = {
      id: `msg-${Date.now()}`,
      type: "agent",
      content: "Message received.",
      timestamp: new Date().toISOString(),
    };
    setInternalMessages((prev) => [...prev, agentResponse]);
    setIsTyping(false);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing) {
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

  const showTypingIndicator = isTyping || isStreaming;
  const showEmptyState =
    messages.length === 0 &&
    !pendingFeedback &&
    !pendingAgentText &&
    !pendingThoughtText &&
    !showTypingIndicator;
  const showSetupForm = showEmptyState && showSetupOnEmpty;

  return (
    <div className={cn("flex h-full flex-col", className)}>
      {/* Header */}
      {!hideHeader && (
        <div className="flex items-center gap-2 border-b px-4 py-2">
          {target.currentStep ? (
            <>
              <Loader2 className="size-3.5 animate-spin text-emerald-500 shrink-0" />
              <span className="text-xs text-muted-foreground">
                Step {target.currentStep.current}/{target.currentStep.total}
              </span>
              <span className="text-xs font-medium truncate flex-1">{target.currentStep.name}</span>
            </>
          ) : (
            <>
              <Bot className="size-4 text-muted-foreground shrink-0" />
              <span className="text-xs font-medium flex-1">{target.agentName}</span>
            </>
          )}
          {pendingFeedback && (
            <Badge variant="outline" className="text-amber-400 border-amber-500/30 text-xs gap-1">
              <Bell className="size-3" />
              Feedback
            </Badge>
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
        className={cn("flex-1 min-h-0 overflow-y-auto", showSetupForm && "flex flex-col")}
        ref={scrollRef}
      >
        <div
          className={cn(
            "py-4 space-y-4 w-full px-6 lg:px-10",
            showSetupForm && "flex-1 flex flex-col justify-center",
          )}
        >
          {/* Inline feedback card at top when present */}
          {pendingFeedback && (
            <InlineFeedbackCard
              inboxItem={pendingFeedback}
              onSubmit={(data) => onFeedbackSubmit?.(pendingFeedback.id, data)}
            />
          )}

          {showSetupForm && (
            <div className="flex flex-col items-center text-center">
              <Bot className="h-8 w-8 text-muted-foreground/40 mb-3" />
              <p className="text-sm font-medium mb-1">{target.agentName}</p>
              <p className="text-xs text-muted-foreground max-w-[280px] mb-5">
                {target.type === "thread_overseer"
                  ? "Configure your thread and start a conversation."
                  : `Ask ${target.agentName} about this task.`}
              </p>
              {emptyStateContent}
            </div>
          )}

          {messages.map((message) => {
            if (message.type === "user") {
              return (
                <div key={message.id} className="flex justify-end">
                  <div className="rounded-2xl px-4 py-2.5 max-w-[85%] text-sm bg-muted">
                    {message.content}
                  </div>
                </div>
              );
            }

            if (message.type === "agent") {
              return (
                <div key={message.id} className="flex items-start gap-2">
                  <span className="mt-2 h-1.5 w-1.5 rounded-full bg-blue-400 shrink-0" />
                  <div className="min-w-0 flex-1">
                    <Markdown className="text-sm">{message.content}</Markdown>
                  </div>
                </div>
              );
            }

            if (message.type === "tool") {
              if (message.tool.type === "mcp") {
                if (/flowgentic[._]+set_topic/.test(message.tool.title)) return null;
                return <McpToolCallBlock key={message.id} tool={message.tool} />;
              }
              return <ToolCallBlock key={message.id} tool={message.tool} />;
            }

            if (message.type === "thinking") {
              return <ThinkingBlock key={message.id} thinking={message.thinking} />;
            }

            return null;
          })}

          {/* Pending streaming thought */}
          {pendingThoughtText && (
            <ThinkingBlock
              thinking={{
                id: "pending-thought",
                status: "thinking",
                streamingContent: pendingThoughtText,
              }}
            />
          )}

          {/* Pending streaming agent text */}
          {pendingAgentText && (
            <div className="flex items-start gap-2">
              <span className="mt-2 h-1.5 w-1.5 rounded-full bg-blue-400 shrink-0" />
              <div className="min-w-0 flex-1">
                <Markdown className="text-sm">{pendingAgentText}</Markdown>
                <span className="inline-block w-1.5 h-4 bg-foreground/70 animate-pulse ml-0.5 -mb-0.5" />
              </div>
            </div>
          )}

          {showTypingIndicator && !pendingAgentText && !pendingThoughtText && (
            <div className="flex items-center gap-2">
              <span className="h-1.5 w-1.5 rounded-full bg-blue-400 shrink-0" />
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
          )}
        </div>
      </div>

      {/* Input */}
      <div className="px-6 lg:px-10 pb-3 pt-2">
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          onChange={handleFileChange}
          className="hidden"
          multiple
        />

        <div className="relative rounded-xl border border-input bg-muted/30 focus-within:ring-1 focus-within:ring-ring">
          <textarea
            ref={textareaRef}
            placeholder={`How can I help you today?`}
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyDown={handleKeyDown}
            disabled={isTyping}
            rows={1}
            className="w-full resize-none bg-transparent px-4 pt-3 pb-10 text-sm placeholder:text-muted-foreground/50 focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
          />
          <div className="absolute bottom-2 left-2 right-2 flex items-center justify-between">
            <div className="flex items-center gap-1">
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7 rounded-lg"
                onClick={handleImageClick}
                disabled={isTyping}
                title="Attach image"
              >
                <ImagePlus className="h-3.5 w-3.5 text-muted-foreground" />
              </Button>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-[10px] text-muted-foreground/40">
                ~{sessionTokens.toLocaleString()} tokens
              </span>
              <Button
                size="icon"
                className="h-7 w-7 rounded-lg"
                onClick={handleSend}
                disabled={!inputValue.trim() || isTyping}
              >
                {isTyping ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                ) : (
                  <ArrowUp className="h-3.5 w-3.5" />
                )}
              </Button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
