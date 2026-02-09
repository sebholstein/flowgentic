import { useState } from "react";
import {
  FileSearch,
  Check,
  MessageSquare,
  Send,
  GitBranch,
  Clock,
  Layers,
  Bot,
  ChevronRight,
  ChevronDown,
  CheckCircle2,
  Circle,
  PlayCircle,
  AlertCircle,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import type { InboxItem } from "@/types/inbox";

// Task in the proposed thread plan
interface ProposedTask {
  id: string;
  name: string;
  description: string;
  agent: string;
  dependencies: string[];
  status?: "pending" | "running" | "completed" | "failed";
}

// Thread review data attached to inbox item
export interface ThreadReviewData {
  threadTitle: string;
  threadDescription: string;
  threadStatus: "draft" | "pending" | "in_progress" | "completed" | "failed";
  totalTasks: number;
  parallelGroups: number;
  estimatedDuration?: string;
  tasks: ProposedTask[];
  planSummary?: string;
  securityConsiderations?: string[];
  technicalNotes?: string[];
}

interface ThreadReviewProps {
  inboxItem: InboxItem;
  reviewData: ThreadReviewData;
  onApprove?: () => void;
  onRequestChanges?: (feedback: string) => void;
}

const statusConfig = {
  draft: { icon: Circle, color: "text-slate-400", label: "Draft" },
  pending: { icon: Circle, color: "text-muted-foreground", label: "Pending Review" },
  in_progress: { icon: PlayCircle, color: "text-blue-400", label: "In Progress" },
  completed: { icon: CheckCircle2, color: "text-emerald-400", label: "Completed" },
  failed: { icon: AlertCircle, color: "text-red-400", label: "Failed" },
};

function TaskDependencyGraph({ tasks }: { tasks: ProposedTask[] }) {
  // Group tasks by their dependency level
  const levels: ProposedTask[][] = [];
  const placed = new Set<string>();

  // First pass: find root tasks (no dependencies)
  const rootTasks = tasks.filter((t) => t.dependencies.length === 0);
  if (rootTasks.length > 0) {
    levels.push(rootTasks);
    rootTasks.forEach((t) => placed.add(t.id));
  }

  // Subsequent passes: find tasks whose dependencies are all placed
  let safety = 0;
  while (placed.size < tasks.length && safety < 20) {
    const nextLevel: ProposedTask[] = [];
    for (const task of tasks) {
      if (placed.has(task.id)) continue;
      const allDepsMet = task.dependencies.every((d) => placed.has(d));
      if (allDepsMet) {
        nextLevel.push(task);
      }
    }
    if (nextLevel.length > 0) {
      levels.push(nextLevel);
      nextLevel.forEach((t) => placed.add(t.id));
    }
    safety++;
  }

  const [expandedLevel, setExpandedLevel] = useState<number | null>(null);

  return (
    <div className="space-y-2">
      {levels.map((level, levelIndex) => (
        <div key={levelIndex} className="space-y-1.5">
          <button
            type="button"
            onClick={() => setExpandedLevel(expandedLevel === levelIndex ? null : levelIndex)}
            className="flex items-center gap-1.5 text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide hover:text-foreground transition-colors"
          >
            {expandedLevel === levelIndex ? (
              <ChevronDown className="size-3" />
            ) : (
              <ChevronRight className="size-3" />
            )}
            {levelIndex === 0 ? "Start" : `Step ${levelIndex}`}
            <span className="text-muted-foreground font-normal">
              · {level.length} {level.length === 1 ? "task" : "parallel tasks"}
            </span>
          </button>

          <div className="grid gap-1.5 pl-4">
            {level.map((task) => (
              <div
                key={task.id}
                className={cn(
                  "flex items-start gap-2.5 rounded-md border border-border bg-card/50 px-3 py-2 transition-all",
                  expandedLevel === levelIndex ? "ring-1 ring-primary/20" : "",
                )}
              >
                <div className="flex size-5 shrink-0 items-center justify-center rounded bg-muted text-[0.6rem] font-medium text-muted-foreground mt-0.5">
                  {task.id.replace("t", "")}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium truncate">{task.name}</span>
                    <Badge variant="outline" className="text-[0.6rem] px-1.5 py-0 h-4 shrink-0">
                      {task.agent}
                    </Badge>
                  </div>
                  {expandedLevel === levelIndex && (
                    <p className="text-xs text-muted-foreground mt-1">{task.description}</p>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}

export function ThreadReview({
  inboxItem,
  reviewData,
  onApprove,
  onRequestChanges,
}: ThreadReviewProps) {
  const [showFeedbackInput, setShowFeedbackInput] = useState(false);
  const [feedback, setFeedback] = useState("");
  const [isSubmitted, setIsSubmitted] = useState(inboxItem.status === "resolved");
  const [approvedState, setApprovedState] = useState<"approved" | "changes_requested" | null>(null);

  const StatusIcon = statusConfig[reviewData.threadStatus].icon;

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
        <div className="rounded-md bg-cyan-500/10 p-1.5">
          <FileSearch className="size-4 text-cyan-400" />
        </div>
        <div className="min-w-0 flex-1">
          <h3 className="text-sm font-semibold truncate">{inboxItem.title}</h3>
          <p className="text-xs text-muted-foreground truncate">{inboxItem.description}</p>
        </div>
        <Badge className="bg-cyan-500/20 text-cyan-400 border-cyan-500/30 text-xs">
          Thread Review
        </Badge>
      </div>

      {/* Source info */}
      <div className="flex items-center gap-1.5">
        <Bot className="size-3.5 text-muted-foreground" />
        <span className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
          {inboxItem.source === "project_overseer" ? "Project Overseer" : "Thread Overseer"}:{" "}
          {inboxItem.sourceName}
        </span>
      </div>

      {/* Thread details */}
      <div className="rounded-lg border border-border bg-muted/30 p-3 space-y-2">
        <div className="flex items-start justify-between gap-3">
          <h4 className="font-semibold text-sm">{reviewData.threadTitle}</h4>
          <Badge
            variant="outline"
            className={cn("shrink-0 text-[0.6rem]", statusConfig[reviewData.threadStatus].color)}
          >
            <StatusIcon className="size-2.5 mr-1" />
            {statusConfig[reviewData.threadStatus].label}
          </Badge>
        </div>
        <div className="text-xs text-muted-foreground leading-relaxed max-w-none">
          {reviewData.threadDescription.split("\n").map((line, i) => {
            if (line.startsWith("### ")) {
              return (
                <h5 key={i} className="text-xs font-semibold text-foreground mt-2 mb-1">
                  {line.replace("### ", "")}
                </h5>
              );
            }
            if (line.startsWith("**") && line.endsWith("**")) {
              return (
                <p key={i} className="font-medium text-foreground mt-1.5 mb-0.5">
                  {line.replace(/\*\*/g, "")}
                </p>
              );
            }
            if (line.startsWith("- ")) {
              return (
                <div key={i} className="flex items-start gap-1.5 ml-2">
                  <span className="text-primary mt-0.5">•</span>
                  <span>{line.replace("- ", "")}</span>
                </div>
              );
            }
            if (line.startsWith("> ")) {
              return (
                <blockquote
                  key={i}
                  className="border-l-2 border-primary/50 pl-2 italic text-muted-foreground my-1.5"
                >
                  {line.replace("> ", "")}
                </blockquote>
              );
            }
            if (line.match(/^\d+\. /)) {
              return (
                <div key={i} className="flex items-start gap-1.5 ml-2">
                  <span className="text-muted-foreground tabular-nums">
                    {line.match(/^(\d+)\./)?.[1]}.
                  </span>
                  <span>{line.replace(/^\d+\. /, "")}</span>
                </div>
              );
            }
            if (line.trim() === "") return <div key={i} className="h-1.5" />;
            return <p key={i}>{line}</p>;
          })}
        </div>
      </div>

      {/* Plan summary */}
      {reviewData.planSummary && (
        <div className="space-y-1.5">
          <span className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
            Plan Summary
          </span>
          <div className="rounded-md border border-border bg-muted/30 p-3">
            <p className="text-xs text-foreground leading-relaxed">{reviewData.planSummary}</p>
          </div>
        </div>
      )}

      {/* Plan stats */}
      <div className="grid grid-cols-3 gap-2">
        <div className="rounded-lg border border-border bg-muted/30 p-2 text-center">
          <div className="flex items-center justify-center gap-1 text-muted-foreground mb-0.5">
            <Layers className="size-3" />
          </div>
          <div className="text-lg font-semibold">{reviewData.totalTasks}</div>
          <div className="text-[0.6rem] text-muted-foreground uppercase">Tasks</div>
        </div>
        <div className="rounded-lg border border-border bg-muted/30 p-2 text-center">
          <div className="flex items-center justify-center gap-1 text-muted-foreground mb-0.5">
            <GitBranch className="size-3" />
          </div>
          <div className="text-lg font-semibold">{reviewData.parallelGroups}</div>
          <div className="text-[0.6rem] text-muted-foreground uppercase">Steps</div>
        </div>
        <div className="rounded-lg border border-border bg-muted/30 p-2 text-center">
          <div className="flex items-center justify-center gap-1 text-muted-foreground mb-0.5">
            <Clock className="size-3" />
          </div>
          <div className="text-lg font-semibold">{reviewData.estimatedDuration ?? "—"}</div>
          <div className="text-[0.6rem] text-muted-foreground uppercase">Est.</div>
        </div>
      </div>

      {/* Security considerations */}
      {reviewData.securityConsiderations && reviewData.securityConsiderations.length > 0 && (
        <div className="space-y-1.5">
          <span className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
            Security Considerations
          </span>
          <ul className="space-y-1">
            {reviewData.securityConsiderations.map((item, i) => (
              <li key={i} className="flex items-start gap-2 text-xs text-muted-foreground">
                <span className="text-amber-400 mt-0.5">!</span>
                <span>{item}</span>
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* Technical notes */}
      {reviewData.technicalNotes && reviewData.technicalNotes.length > 0 && (
        <div className="space-y-1.5">
          <span className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
            Technical Notes
          </span>
          <ul className="space-y-1">
            {reviewData.technicalNotes.map((item, i) => (
              <li key={i} className="flex items-start gap-2 text-xs text-muted-foreground">
                <span className="text-primary mt-0.5">•</span>
                <span>{item}</span>
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* Task breakdown */}
      <div className="space-y-2">
        <span className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
          Proposed Task Breakdown
        </span>
        <TaskDependencyGraph tasks={reviewData.tasks} />
      </div>

      {/* Feedback input */}
      {showFeedbackInput && !isSubmitted && (
        <div className="space-y-1.5 pt-2 border-t">
          <label className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
            Feedback for changes
          </label>
          <Textarea
            placeholder="Describe what changes you'd like to see in this thread plan..."
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
                  Thread approved — tasks will begin execution
                </span>
              </>
            ) : (
              <>
                <MessageSquare className="size-3.5" />
                <span className="text-xs font-medium">Changes requested — awaiting revision</span>
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
                Approve Thread
              </Button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// Legacy exports for backwards compatibility
export type IssueReviewData = ThreadReviewData;
export const IssueReview = ThreadReview;
