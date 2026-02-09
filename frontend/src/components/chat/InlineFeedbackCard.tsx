import { useState, useMemo } from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  ChevronUp,
  ChevronDown,
  HelpCircle,
  AlertTriangle,
  Compass,
  FileText,
  FileSearch,
  GitCompare,
  Bot,
} from "lucide-react";
import { QuestionnaireView } from "@/components/inbox/QuestionnaireView";
import { DecisionEscalation } from "@/components/inbox/DecisionEscalation";
import { DirectionClarification } from "@/components/inbox/DirectionClarification";
import { PlanningApproval } from "@/components/inbox/PlanningApproval";
import { ThreadReview } from "@/components/inbox/ThreadReview";
import { ExecutionComparison } from "@/components/inbox/ExecutionComparison";
import {
  planningDataByItemId,
  threadReviewDataByItemId,
  executionsData,
} from "@/data/mockInboxData";
import type { InboxItem, InboxItemType, OverseerMessage } from "@/types/inbox";

// Type icons mapping
const typeIcons: Record<InboxItemType, React.ElementType> = {
  questionnaire: HelpCircle,
  decision_escalation: AlertTriangle,
  direction_clarification: Compass,
  planning_approval: FileText,
  task_plan_approval: FileText,
  thread_review: FileSearch,
  execution_selection: GitCompare,
};

const typeColors: Record<InboxItemType, { bg: string; text: string; border: string }> = {
  questionnaire: {
    bg: "bg-violet-500/10",
    text: "text-violet-400",
    border: "border-violet-500/30",
  },
  decision_escalation: {
    bg: "bg-amber-500/10",
    text: "text-amber-400",
    border: "border-amber-500/30",
  },
  direction_clarification: {
    bg: "bg-blue-500/10",
    text: "text-blue-400",
    border: "border-blue-500/30",
  },
  planning_approval: {
    bg: "bg-indigo-500/10",
    text: "text-indigo-400",
    border: "border-indigo-500/30",
  },
  task_plan_approval: {
    bg: "bg-orange-500/10",
    text: "text-orange-400",
    border: "border-orange-500/30",
  },
  thread_review: { bg: "bg-cyan-500/10", text: "text-cyan-400", border: "border-cyan-500/30" },
  execution_selection: {
    bg: "bg-purple-500/10",
    text: "text-purple-400",
    border: "border-purple-500/30",
  },
};

const typeLabels: Record<InboxItemType, string> = {
  questionnaire: "Questionnaire",
  decision_escalation: "Decision Needed",
  direction_clarification: "Guidance Needed",
  planning_approval: "Planning Review",
  task_plan_approval: "Task Plan Review",
  thread_review: "Thread Review",
  execution_selection: "Choose Execution",
};

interface InlineFeedbackCardProps {
  inboxItem: InboxItem;
  onSubmit: (data?: unknown) => void;
}

export function InlineFeedbackCard({ inboxItem, onSubmit }: InlineFeedbackCardProps) {
  const [isCollapsed, setIsCollapsed] = useState(false);

  const TypeIcon = typeIcons[inboxItem.type];
  const colors = typeColors[inboxItem.type];
  const typeLabel = typeLabels[inboxItem.type];

  // Handle submission for different types
  const handleSubmit = (data?: unknown) => {
    onSubmit(data);
  };

  // Render conversation history if present
  const conversationHistory = useMemo(() => {
    if (!inboxItem.overseerMessages || inboxItem.overseerMessages.length === 0) {
      return null;
    }

    return (
      <div className="space-y-2 mb-4 pb-4 border-b border-border">
        <span className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
          Conversation
        </span>
        <div className="space-y-2">
          {inboxItem.overseerMessages.map((msg) => (
            <ConversationMessage key={msg.id} message={msg} />
          ))}
        </div>
      </div>
    );
  }, [inboxItem.overseerMessages]);

  // Render the appropriate detail view based on type
  const detailView = useMemo(() => {
    switch (inboxItem.type) {
      case "questionnaire":
        return (
          <QuestionnaireView
            inboxItem={inboxItem}
            onSubmit={(answers, otherAnswers, context) =>
              handleSubmit({ answers, otherAnswers, context })
            }
          />
        );
      case "decision_escalation":
        return (
          <DecisionEscalation
            inboxItem={inboxItem}
            onSubmit={(decisionId, rationale) => handleSubmit({ decisionId, rationale })}
          />
        );
      case "direction_clarification":
        return (
          <DirectionClarification
            inboxItem={inboxItem}
            onSubmit={(response, delegated) => handleSubmit({ response, delegated })}
          />
        );
      case "planning_approval": {
        const planningData = planningDataByItemId[inboxItem.id];
        if (!planningData) return null;
        return (
          <PlanningApproval
            inboxItem={inboxItem}
            planningData={planningData}
            onApprove={() => handleSubmit({ approved: true })}
            onRequestChanges={(feedback) => handleSubmit({ approved: false, feedback })}
          />
        );
      }
      case "thread_review": {
        const reviewData = threadReviewDataByItemId[inboxItem.id];
        if (!reviewData) return null;
        return (
          <ThreadReview
            inboxItem={inboxItem}
            reviewData={reviewData}
            onApprove={() => handleSubmit({ approved: true })}
            onRequestChanges={(feedback) => handleSubmit({ approved: false, feedback })}
          />
        );
      }
      case "execution_selection": {
        const executions =
          inboxItem.executionIds
            ?.map((id) => executionsData.find((e) => e.id === id))
            .filter((e) => e !== undefined) ?? [];
        return (
          <ExecutionComparison
            title={inboxItem.title}
            taskName={inboxItem.taskName}
            executions={executions}
            selectedExecutionId={inboxItem.selectedExecutionId}
            onSelectExecution={(executionId) => handleSubmit({ executionId })}
          />
        );
      }
      default:
        return null;
    }
  }, [inboxItem]);

  return (
    <Card className={cn("border-2", colors.border, "bg-card/50")}>
      <CardHeader className="p-3 pb-2">
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-2 min-w-0">
            <div className={cn("rounded-md p-1.5", colors.bg)}>
              <TypeIcon className={cn("size-4", colors.text)} />
            </div>
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium truncate">{inboxItem.title}</span>
                <Badge className={cn("text-xs shrink-0", colors.bg, colors.text, colors.border)}>
                  {typeLabel}
                </Badge>
              </div>
              <p className="text-xs text-muted-foreground truncate">
                {inboxItem.sourceName} • {inboxItem.threadName ?? inboxItem.taskName}
              </p>
            </div>
          </div>
          <Button
            variant="ghost"
            size="sm"
            className="size-7 p-0 shrink-0"
            onClick={() => setIsCollapsed(!isCollapsed)}
          >
            {isCollapsed ? <ChevronDown className="size-4" /> : <ChevronUp className="size-4" />}
          </Button>
        </div>
      </CardHeader>

      {!isCollapsed && (
        <CardContent className="p-3 pt-0">
          {conversationHistory}
          {detailView}
        </CardContent>
      )}
    </Card>
  );
}

// Conversation message component
function ConversationMessage({ message }: { message: OverseerMessage }) {
  const isAgent = message.role === "agent";
  const isOverseer = message.role === "overseer";

  return (
    <div className={cn("flex gap-2", isAgent && "flex-row-reverse")}>
      <Avatar
        className={cn(
          "h-6 w-6 shrink-0",
          isOverseer ? "bg-violet-500" : isAgent ? "bg-orange-500" : "bg-primary",
        )}
      >
        <AvatarFallback className="text-white text-[9px] font-medium">
          {isOverseer ? (
            <Bot className="size-3" />
          ) : isAgent ? (
            (message.agentId?.[0]?.toUpperCase() ?? "A")
          ) : (
            "U"
          )}
        </AvatarFallback>
      </Avatar>
      <div
        className={cn(
          "rounded-lg px-2.5 py-1.5 max-w-[85%] text-xs",
          isAgent
            ? "bg-muted"
            : isOverseer
              ? "bg-violet-500/10 text-foreground"
              : "bg-primary text-primary-foreground",
        )}
      >
        {message.content}
      </div>
    </div>
  );
}
