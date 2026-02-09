import { useState } from "react";
import {
  ClipboardList,
  Check,
  MessageSquare,
  Send,
  Bot,
  ChevronDown,
  ChevronRight,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import type { InboxItem } from "@/types/inbox";
import type { TaskPlan } from "@/types/task";
import { threadTasks } from "@/data/mockTasksData";

interface TaskPlanApprovalProps {
  inboxItem: InboxItem;
  onApprove?: () => void;
  onRequestChanges?: (feedback: string) => void;
}

export function TaskPlanApproval({
  inboxItem,
  onApprove,
  onRequestChanges,
}: TaskPlanApprovalProps) {
  const [showFeedbackInput, setShowFeedbackInput] = useState(false);
  const [feedback, setFeedback] = useState("");
  const [isSubmitted, setIsSubmitted] = useState(inboxItem.status === "resolved");
  const [approvedState, setApprovedState] = useState<"approved" | "changes_requested" | null>(null);
  const [stepsExpanded, setStepsExpanded] = useState(false);

  // Find the task data to get the plan
  const task =
    inboxItem.threadId && inboxItem.taskId
      ? threadTasks[inboxItem.threadId]?.find((t) => t.id === inboxItem.taskId)
      : undefined;

  const plan: TaskPlan | undefined = task?.plan;
  const plannerPrompt = task?.plannerPrompt;

  const handleApprove = () => {
    setIsSubmitted(true);
    setApprovedState("approved");
    onApprove?.();
  };

  const handleRequestChanges = () => {
    if (feedback.trim()) {
      setIsSubmitted(true);
      setApprovedState("changes_requested");
      onRequestChanges?.(feedback);
    }
  };

  return (
    <div className="rounded-lg border border-border bg-card/50 p-4 space-y-4">
      {/* Card header */}
      <div className="flex items-center gap-2">
        <div className="rounded-md bg-orange-500/10 p-1.5">
          <ClipboardList className="size-4 text-orange-400" />
        </div>
        <div className="min-w-0 flex-1">
          <h3 className="text-sm font-semibold truncate">
            {inboxItem.taskName ?? inboxItem.title}
          </h3>
          <p className="text-xs text-muted-foreground truncate">{inboxItem.description}</p>
        </div>
        <Badge className="bg-orange-500/20 text-orange-400 border-orange-500/30 text-xs">
          Task Plan Review
        </Badge>
      </div>

      {/* Context */}
      <div className="space-y-1.5">
        <div className="flex items-center gap-1.5">
          <Bot className="size-3.5 text-muted-foreground" />
          <span className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
            Planned by: {plan?.agentName ?? "Agent"}
          </span>
        </div>
        {inboxItem.threadName && (
          <p className="text-xs text-muted-foreground">Thread: {inboxItem.threadName}</p>
        )}
      </div>

      {/* Planner Prompt */}
      {plannerPrompt && (
        <div className="space-y-1.5">
          <span className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
            Planner Instructions
          </span>
          <div className="rounded-md border border-border bg-muted/30 p-3">
            <p className="text-xs text-foreground leading-relaxed">{plannerPrompt}</p>
          </div>
        </div>
      )}

      {/* Plan Output */}
      {plan && (
        <>
          {/* Summary */}
          <div className="space-y-1.5">
            <span className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
              Plan Summary
            </span>
            <div className="rounded-md border border-border bg-muted/30 p-3">
              <p className="text-xs text-foreground leading-relaxed">{plan.summary}</p>
            </div>
          </div>

          {/* Steps */}
          <div className="space-y-1.5">
            <button
              type="button"
              onClick={() => setStepsExpanded(!stepsExpanded)}
              className="flex items-center gap-1.5 text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide hover:text-foreground transition-colors"
            >
              {stepsExpanded ? (
                <ChevronDown className="size-3" />
              ) : (
                <ChevronRight className="size-3" />
              )}
              Steps ({plan.steps.length})
            </button>
            {stepsExpanded && (
              <ol className="space-y-1.5 pl-4">
                {plan.steps.map((step, i) => (
                  <li key={i} className="flex items-start gap-2 text-xs text-muted-foreground">
                    <span className="flex size-5 shrink-0 items-center justify-center rounded bg-muted text-[0.6rem] font-medium text-muted-foreground mt-0.5">
                      {i + 1}
                    </span>
                    <span className="leading-relaxed">{step}</span>
                  </li>
                ))}
              </ol>
            )}
          </div>

          {/* Approach */}
          {plan.approach && (
            <div className="space-y-1.5">
              <span className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
                Approach
              </span>
              <p className="text-xs text-muted-foreground leading-relaxed">{plan.approach}</p>
            </div>
          )}

          {/* Considerations */}
          {plan.considerations && plan.considerations.length > 0 && (
            <div className="space-y-1.5">
              <span className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
                Considerations
              </span>
              <ul className="space-y-1">
                {plan.considerations.map((item, i) => (
                  <li key={i} className="flex items-start gap-2 text-xs text-muted-foreground">
                    <span className="text-primary mt-0.5">•</span>
                    <span>{item}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {/* Complexity */}
          {plan.estimatedComplexity && (
            <div className="flex items-center gap-2">
              <span className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
                Complexity:
              </span>
              <Badge
                variant="outline"
                className={cn(
                  "text-[0.6rem] px-1.5 py-0 h-4",
                  plan.estimatedComplexity === "low" && "text-emerald-400 border-emerald-500/30",
                  plan.estimatedComplexity === "medium" && "text-amber-400 border-amber-500/30",
                  plan.estimatedComplexity === "high" && "text-red-400 border-red-500/30",
                )}
              >
                {plan.estimatedComplexity}
              </Badge>
            </div>
          )}
        </>
      )}

      {/* Feedback input */}
      {showFeedbackInput && !isSubmitted && (
        <div className="space-y-1.5 pt-2 border-t">
          <label className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
            Feedback for changes
          </label>
          <Textarea
            placeholder="Describe what changes you'd like to the task plan..."
            value={feedback}
            onChange={(e) => setFeedback(e.target.value)}
            rows={3}
            className="text-xs resize-none"
            autoFocus
          />
        </div>
      )}

      {/* Submitted state or action buttons */}
      {isSubmitted ? (
        <div
          className={cn(
            "rounded-md border px-3 py-2",
            approvedState === "approved"
              ? "border-emerald-500/30 bg-emerald-500/10"
              : "border-amber-500/30 bg-amber-500/10",
          )}
        >
          <div
            className={cn(
              "flex items-center gap-1.5",
              approvedState === "approved" ? "text-emerald-400" : "text-amber-400",
            )}
          >
            {approvedState === "approved" ? (
              <>
                <Check className="size-3.5" />
                <span className="text-xs font-medium">
                  Plan approved — task execution will begin shortly
                </span>
              </>
            ) : (
              <>
                <MessageSquare className="size-3.5" />
                <span className="text-xs font-medium">
                  Changes requested — awaiting re-planning
                </span>
              </>
            )}
          </div>
          {approvedState === "changes_requested" && feedback && (
            <p className="text-xs text-muted-foreground mt-1">{feedback}</p>
          )}
        </div>
      ) : (
        <div className="space-y-2">
          {showFeedbackInput ? (
            <div className="flex gap-2">
              <Button
                variant="outline"
                onClick={() => {
                  setShowFeedbackInput(false);
                  setFeedback("");
                }}
                className="flex-1"
                size="sm"
              >
                Cancel
              </Button>
              <Button
                onClick={handleRequestChanges}
                disabled={!feedback.trim()}
                variant="secondary"
                className="flex-1"
                size="sm"
              >
                <Send className="size-3.5" />
                Send Feedback
              </Button>
            </div>
          ) : (
            <div className="flex gap-2">
              <Button
                variant="outline"
                onClick={() => setShowFeedbackInput(true)}
                className="flex-1"
                size="sm"
              >
                <MessageSquare className="size-3.5" />
                Request Changes
              </Button>
              <Button onClick={handleApprove} className="flex-1" size="sm">
                <Check className="size-3.5" />
                Approve Plan
              </Button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
