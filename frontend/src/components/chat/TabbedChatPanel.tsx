import { useState } from "react";
import { cn } from "@/lib/utils";
import { Bot, Code2 } from "lucide-react";
import { AgentChatPanel, type ChatTarget } from "./AgentChatPanel";
import type { InboxItem } from "@/types/inbox";

type ChatTab = "planner" | "worker";

interface TabbedChatPanelProps {
  /** Entity ID (thread or task) */
  entityId: string;
  /** Entity type for context */
  entityType: "thread" | "task";
  /** Display title for the entity */
  entityTitle: string;
  /** Optional pending feedback to show in planner chat */
  pendingFeedback?: InboxItem | null;
  /** Callback when feedback is submitted */
  onFeedbackSubmit?: (itemId: string, data: unknown) => void;
  /** Enable simulation mode */
  enableSimulation?: boolean;
  /** Current step the worker is working on */
  currentStep?: {
    name: string;
    current: number;
    total: number;
  };
  /** Selected model */
  threadModel?: string;
  /** Callback when model changes */
  onModelChange?: (model: string) => void;
  className?: string;
}

export function TabbedChatPanel({
  entityId,
  entityType,
  entityTitle,
  pendingFeedback,
  onFeedbackSubmit,
  enableSimulation = false,
  currentStep,
  threadModel,
  onModelChange,
  className,
}: TabbedChatPanelProps) {
  const [activeTab, setActiveTab] = useState<ChatTab>("planner");

  // Define chat targets for planner and worker
  const plannerTarget: ChatTarget = {
    type: entityType === "thread" ? "thread_overseer" : "task_agent",
    entityId: `${entityId}-planner`,
    agentName: "Overseer",
    title: entityTitle,
    agentColor: "bg-violet-500",
  };

  const workerTarget: ChatTarget = {
    type: "task_agent",
    entityId: `${entityId}-worker`,
    agentName: "Worker",
    title: entityTitle,
    agentColor: "bg-emerald-500",
    currentStep,
  };

  return (
    <div className={cn("flex h-full flex-col", className)}>
      {/* Tab switcher */}
      <div className="flex border-b bg-muted/30">
        <button
          type="button"
          onClick={() => setActiveTab("planner")}
          className={cn(
            "flex-1 flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium transition-colors border-b-2 cursor-pointer",
            activeTab === "planner"
              ? "bg-background text-foreground border-violet-500"
              : "text-muted-foreground hover:text-foreground hover:bg-muted/50 border-transparent",
          )}
        >
          <Bot className="size-3.5" />
          Overseer
        </button>
        <button
          type="button"
          onClick={() => setActiveTab("worker")}
          className={cn(
            "flex-1 flex items-center justify-center gap-1.5 px-3 py-2 text-xs font-medium transition-colors border-b-2 cursor-pointer",
            activeTab === "worker"
              ? "bg-background text-foreground border-emerald-500"
              : "text-muted-foreground hover:text-foreground hover:bg-muted/50 border-transparent",
          )}
        >
          <Code2 className="size-3.5" />
          Worker
        </button>
      </div>

      {/* Chat panels - render both but only show active one to preserve state */}
      <div className="flex-1 min-h-0 relative">
        <div className={cn("absolute inset-0", activeTab !== "planner" && "hidden")}>
          <AgentChatPanel
            target={plannerTarget}
            hideHeader
            pendingFeedback={pendingFeedback}
            onFeedbackSubmit={onFeedbackSubmit}
            threadModel={threadModel}
            onModelChange={onModelChange}
          />
        </div>
        <div className={cn("absolute inset-0", activeTab !== "worker" && "hidden")}>
          <AgentChatPanel
            target={workerTarget}
            enableSimulation={enableSimulation}
            threadModel={threadModel}
            onModelChange={onModelChange}
          />
        </div>
      </div>
    </div>
  );
}
