import { cn } from "@/lib/utils";
import { Bot } from "lucide-react";
import { ToolCallBlock } from "./ToolCallBlock";
import { McpToolCallBlock } from "./McpToolCallBlock";
import { ThinkingBlock } from "./ThinkingBlock";
import { InlineFeedbackCard } from "./InlineFeedbackCard";
import { Markdown } from "@/components/ui/markdown";
import { StreamingSpinner } from "./StreamingSpinner";
import type { ChatTarget, ChatMessage } from "./chat-types";
import type { InboxItem } from "@/types/inbox";

export function ChatMessageList({
  messages,
  target,
  pendingFeedback,
  onFeedbackSubmit,
  pendingAgentText,
  pendingThoughtText,
  showTypingIndicator,
  showSetupForm,
  emptyStateContent,
}: {
  messages: ChatMessage[];
  target: ChatTarget;
  pendingFeedback?: InboxItem | null;
  onFeedbackSubmit?: (itemId: string, data: unknown) => void;
  pendingAgentText: string;
  pendingThoughtText: string;
  showTypingIndicator: boolean;
  showSetupForm: boolean;
  emptyStateContent?: React.ReactNode;
}) {
  return (
    <div
      className={cn(
        "py-4 space-y-4 w-full px-6 lg:px-10",
        showSetupForm && "flex-1 flex flex-col justify-center",
      )}
    >
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

      {pendingThoughtText && (
        <ThinkingBlock
          thinking={{
            id: "pending-thought",
            status: "thinking",
            streamingContent: pendingThoughtText,
          }}
        />
      )}

      {pendingAgentText && (
        <div className="flex items-start gap-2">
          <span className="mt-2 h-1.5 w-1.5 rounded-full bg-blue-400 shrink-0" />
          <div className="min-w-0 flex-1">
            <Markdown className="text-sm">{pendingAgentText}</Markdown>
          </div>
        </div>
      )}

      {showTypingIndicator && !pendingAgentText && !pendingThoughtText && (
        <StreamingSpinner />
      )}
    </div>
  );
}
