import { useRef, useEffect, type ReactNode } from "react";
import { Bot, User } from "lucide-react";
import { cn } from "@/lib/utils";
import { ChatInput } from "./ChatInput";
import type { InboxItem, OverseerMessage } from "@/types/inbox";

const agentColors: Record<string, string> = {
  "claude-opus-4": "bg-orange-500",
  "claude-sonnet": "bg-amber-500",
  "gpt-4": "bg-emerald-500",
  "gpt-4o": "bg-teal-500",
};

interface ChatMessageProps {
  message: OverseerMessage;
}

function ChatMessage({ message }: ChatMessageProps) {
  const isOverseer = message.role === "overseer";
  const isUser = message.role === "user";
  const bgColor = isOverseer
    ? "bg-purple-500"
    : isUser
      ? "bg-slate-500"
      : (agentColors[message.agentId ?? ""] ?? "bg-blue-500");

  return (
    <div className="flex gap-2">
      <div
        className={cn(
          "flex size-5 shrink-0 items-center justify-center rounded-full text-white",
          bgColor,
        )}
      >
        {isOverseer ? (
          <Bot className="size-3" />
        ) : isUser ? (
          <User className="size-3" />
        ) : (
          <span className="text-[0.5rem] font-medium">
            {message.agentId?.[0]?.toUpperCase() ?? "A"}
          </span>
        )}
      </div>
      <div className="min-w-0 flex-1 space-y-0.5">
        <div className="flex items-baseline gap-1.5">
          <span className="text-xs font-medium">
            {isOverseer ? "Overseer" : isUser ? "You" : message.agentId}
          </span>
          <span className="text-[0.6rem] text-muted-foreground">{message.timestamp}</span>
        </div>
        <p className="text-xs text-muted-foreground leading-relaxed">{message.content}</p>
      </div>
    </div>
  );
}

interface ChatHistoryProps {
  messages: OverseerMessage[];
}

function ChatHistory({ messages }: ChatHistoryProps) {
  if (messages.length === 0) return null;

  return (
    <div className="space-y-3">
      {messages.map((message) => (
        <ChatMessage key={message.id} message={message} />
      ))}
    </div>
  );
}

interface InboxChatLayoutProps {
  inboxItem: InboxItem;
  children: ReactNode;
  onSendMessage?: (message: string) => void;
  header?: ReactNode;
}

export function InboxChatLayout({
  inboxItem,
  children,
  onSendMessage,
  header,
}: InboxChatLayoutProps) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    if (contentRef.current) {
      contentRef.current.scrollIntoView({ behavior: "smooth", block: "end" });
    }
  }, [inboxItem.overseerMessages?.length]);

  const handleSend = (message: string) => {
    onSendMessage?.(message);
  };

  const messages = inboxItem.overseerMessages ?? [];

  return (
    <div className="flex h-full flex-col overflow-hidden">
      {header && <div className="shrink-0 sticky top-0 z-10 bg-background">{header}</div>}

      <div className="flex-1 min-h-0 overflow-y-auto" ref={scrollRef}>
        <div className="p-4 space-y-4" ref={contentRef}>
          {/* Chat history */}
          {messages.length > 0 && <ChatHistory messages={messages} />}

          {/* Inline content card (questionnaire, decision, etc.) */}
          {children}
        </div>
      </div>

      {/* Persistent chat input - always visible at bottom */}
      {onSendMessage && (
        <div className="shrink-0">
          <ChatInput onSend={handleSend} placeholder="Type a message..." />
        </div>
      )}
    </div>
  );
}
