import { cn } from "@/lib/utils";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Bot, User, CheckCircle2, RotateCcw } from "lucide-react";
import type { OverseerMessage, TaskExecution } from "@/types/inbox";

// Agent color mapping
const agentColors: Record<string, string> = {
  "claude-opus-4": "bg-orange-500",
  "claude-sonnet": "bg-amber-500",
  "gpt-4": "bg-emerald-500",
  "gpt-4o": "bg-teal-500",
  "gemini-pro": "bg-blue-500",
};

function getAgentColor(agentId?: string) {
  return agentId ? (agentColors[agentId] ?? "bg-purple-500") : "bg-purple-500";
}

interface MessageBubbleProps {
  message: OverseerMessage;
}

function MessageBubble({ message }: MessageBubbleProps) {
  const isOverseer = message.role === "overseer";
  const isUser = message.role === "user";

  return (
    <div className="flex items-start gap-3">
      <Avatar className="size-8 shrink-0">
        <AvatarFallback
          className={cn(
            "text-xs font-medium text-white",
            isOverseer ? "bg-purple-500" : isUser ? "bg-slate-500" : getAgentColor(message.agentId),
          )}
        >
          {isOverseer ? (
            <Bot className="size-4" />
          ) : isUser ? (
            <User className="size-4" />
          ) : (
            (message.agentId?.[0]?.toUpperCase() ?? "A")
          )}
        </AvatarFallback>
      </Avatar>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="font-medium text-sm">
            {isOverseer ? "Overseer" : isUser ? "You" : message.agentId}
          </span>
          {message.role === "agent" && (
            <Badge variant="outline" className="text-[0.6rem]">
              Agent
            </Badge>
          )}
          <span className="text-xs text-muted-foreground">{message.timestamp}</span>
        </div>
        <div className="mt-1 text-sm text-slate-300 whitespace-pre-wrap leading-relaxed">
          {message.content}
        </div>
      </div>
    </div>
  );
}

interface DecisionBoxProps {
  selectedExecutionId?: string;
  executions: TaskExecution[];
  decidedBy?: "user" | "overseer";
  onOverride?: () => void;
}

function DecisionBox({ selectedExecutionId, executions, decidedBy, onOverride }: DecisionBoxProps) {
  const selectedExecution = executions.find((e) => e.id === selectedExecutionId);

  if (!selectedExecution) return null;

  return (
    <div className="rounded-lg border border-emerald-500/30 bg-emerald-500/5 p-4">
      <div className="flex items-center gap-2 mb-2">
        <CheckCircle2 className="size-4 text-emerald-400" />
        <span className="text-sm font-medium text-emerald-400">
          {decidedBy === "overseer" ? "Approved by overseer" : "Selected by user"}
        </span>
      </div>
      <div className="text-sm text-slate-300 mb-3">
        Selected execution: <span className="font-medium">{selectedExecution.agentName}</span>
      </div>
      {decidedBy === "overseer" && onOverride && (
        <Button variant="outline" size="sm" onClick={onOverride} className="gap-1.5 text-xs">
          <RotateCcw className="size-3" />
          Override Selection
        </Button>
      )}
    </div>
  );
}

interface OverseerConversationProps {
  messages: OverseerMessage[];
  executions: TaskExecution[];
  selectedExecutionId?: string;
  decidedBy?: "user" | "overseer";
  onOverride?: () => void;
}

export function OverseerConversation({
  messages,
  executions,
  selectedExecutionId,
  decidedBy,
  onOverride,
}: OverseerConversationProps) {
  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-2 px-6 py-4 border-b">
        <Bot className="size-5 text-purple-400" />
        <h2 className="font-semibold">Overseer Decision History</h2>
      </div>

      <ScrollArea className="flex-1">
        <div className="px-6 py-4 space-y-6">
          {messages.map((message) => (
            <MessageBubble key={message.id} message={message} />
          ))}

          {selectedExecutionId && (
            <DecisionBox
              selectedExecutionId={selectedExecutionId}
              executions={executions}
              decidedBy={decidedBy}
              onOverride={onOverride}
            />
          )}
        </div>
      </ScrollArea>

      {/* Empty state */}
      {messages.length === 0 && (
        <div className="flex-1 flex flex-col items-center justify-center py-12 text-center">
          <Bot className="size-12 text-muted-foreground/50 mb-4" />
          <p className="text-muted-foreground">No overseer activity yet</p>
        </div>
      )}
    </div>
  );
}
