import { useState } from "react";
import {
  FileText,
  Check,
  MessageSquare,
  Send,
  GitBranch,
  Clock,
  Layers,
  Bot,
  ChevronRight,
  ChevronDown,
  ClipboardList,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { InboxItem } from "@/types/inbox";

// Worker assigned to a task (specific model that will execute)
export interface TaskWorker {
  id: string; // e.g., "claude-opus-4", "gpt-4"
  name: string; // e.g., "Claude Opus 4", "GPT-4"
}

// Task in the proposed plan
export interface PlannedTask {
  id: string;
  name: string;
  description: string;
  agent: string; // Agent type categorization
  workers?: TaskWorker[]; // Specific models to run this task (execution phase)
  dependencies: string[];
  estimatedDuration?: string;

  // Planning phase (optional — if plannerPrompt is set, task goes through planning before execution)
  plannerPrompt?: string;
  planApproval?: "user" | "overseer" | "auto";
  planner?: TaskWorker; // Agent/model for the planning phase
}

// Planning data attached to inbox item
export interface PlanningData {
  issueTitle: string;
  issueDescription: string;
  totalTasks: number;
  parallelGroups: number;
  estimatedDuration?: string;
  tasks: PlannedTask[];
  planSummary?: string;
  considerations?: string[];
}

interface PlanningApprovalProps {
  inboxItem: InboxItem;
  planningData: PlanningData;
  onApprove?: () => void;
  onRequestChanges?: (feedback: string) => void;
}

// Worker/model color mapping (consistent with ExecutionCard)
const workerStyles: Record<string, { icon: string; color: string }> = {
  "claude-opus-4": { icon: "C", color: "bg-orange-500" },
  "claude-sonnet": { icon: "C", color: "bg-amber-500" },
  "gpt-4": { icon: "G", color: "bg-emerald-500" },
  "gpt-4o": { icon: "G", color: "bg-teal-500" },
  "gemini-pro": { icon: "G", color: "bg-blue-500" },
  default: { icon: "A", color: "bg-slate-500" },
};

function getWorkerStyle(workerId: string) {
  return workerStyles[workerId] ?? workerStyles.default;
}

function WorkerBadges({ workers }: { workers?: TaskWorker[] }) {
  if (!workers || workers.length === 0) return null;

  if (workers.length === 1) {
    const style = getWorkerStyle(workers[0].id);
    return (
      <div className="flex items-center gap-1">
        <div
          className={cn(
            "flex size-4 items-center justify-center rounded text-[0.5rem] font-bold text-white",
            style.color,
          )}
        >
          {style.icon}
        </div>
        <span className="text-[0.6rem] text-muted-foreground">{workers[0].name}</span>
      </div>
    );
  }

  // Multiple workers - show stacked badges with tooltip
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <div className="flex items-center">
          <div className="flex -space-x-1">
            {workers.slice(0, 3).map((worker, idx) => {
              const style = getWorkerStyle(worker.id);
              return (
                <div
                  key={worker.id}
                  className={cn(
                    "flex size-4 items-center justify-center rounded text-[0.5rem] font-bold text-white ring-1 ring-background",
                    style.color,
                  )}
                  style={{ zIndex: workers.length - idx }}
                >
                  {style.icon}
                </div>
              );
            })}
          </div>
          <span className="ml-1.5 text-[0.6rem] text-muted-foreground">
            {workers.length} workers
          </span>
        </div>
      </TooltipTrigger>
      <TooltipContent side="top" className="max-w-xs">
        <p className="text-xs font-medium mb-1">Assigned Workers</p>
        <div className="flex flex-col gap-1">
          {workers.map((worker) => {
            const style = getWorkerStyle(worker.id);
            return (
              <div key={worker.id} className="flex items-center gap-1.5">
                <div
                  className={cn(
                    "flex size-3.5 items-center justify-center rounded text-[0.45rem] font-bold text-white",
                    style.color,
                  )}
                >
                  {style.icon}
                </div>
                <span className="text-xs">{worker.name}</span>
              </div>
            );
          })}
        </div>
      </TooltipContent>
    </Tooltip>
  );
}

function TaskDependencyGraph({ tasks }: { tasks: PlannedTask[] }) {
  // Group tasks by their dependency level
  const levels: PlannedTask[][] = [];
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
    const nextLevel: PlannedTask[] = [];
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
  const [expandedTaskId, setExpandedTaskId] = useState<string | null>(null);

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
            {level.map((task) => {
              const isTaskExpanded = expandedTaskId === task.id;
              const hasPlannerPrompt = !!task.plannerPrompt;

              return (
                <div
                  key={task.id}
                  className={cn(
                    "rounded-md border border-border bg-card/50 transition-all",
                    expandedLevel === levelIndex ? "ring-1 ring-primary/20" : "",
                  )}
                >
                  <button
                    type="button"
                    onClick={() => setExpandedTaskId(isTaskExpanded ? null : task.id)}
                    className="flex items-start gap-2.5 px-3 py-2 w-full text-left"
                  >
                    <div className="flex size-5 shrink-0 items-center justify-center rounded bg-muted text-[0.6rem] font-medium text-muted-foreground mt-0.5">
                      {task.id.replace("t", "")}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className="text-sm font-medium truncate">{task.name}</span>
                        <Badge variant="outline" className="text-[0.6rem] px-1.5 py-0 h-4 shrink-0">
                          {task.agent}
                        </Badge>
                        {hasPlannerPrompt && (
                          <Badge
                            variant="outline"
                            className="text-[0.6rem] px-1.5 py-0 h-4 shrink-0 text-orange-400 border-orange-500/30 bg-orange-500/10 gap-0.5"
                          >
                            <ClipboardList className="size-2.5" />
                            Plans
                          </Badge>
                        )}
                        {hasPlannerPrompt && task.planApproval && (
                          <span className="text-[0.55rem] text-muted-foreground">
                            {task.planApproval === "user"
                              ? "User approves"
                              : task.planApproval === "overseer"
                                ? "Overseer approves"
                                : "Auto"}
                          </span>
                        )}
                        {task.workers && task.workers.length > 0 && (
                          <div className="flex items-center ml-auto">
                            <WorkerBadges workers={task.workers} />
                          </div>
                        )}
                      </div>
                      {expandedLevel === levelIndex && !isTaskExpanded && (
                        <p className="text-xs text-muted-foreground mt-1">{task.description}</p>
                      )}
                    </div>
                    {hasPlannerPrompt &&
                      (isTaskExpanded ? (
                        <ChevronDown className="size-3 text-muted-foreground mt-1.5 shrink-0" />
                      ) : (
                        <ChevronRight className="size-3 text-muted-foreground mt-1.5 shrink-0" />
                      ))}
                  </button>

                  {/* Expanded task details */}
                  {isTaskExpanded && (
                    <div className="border-t border-border/50 px-3 py-2 space-y-2">
                      <p className="text-xs text-muted-foreground">{task.description}</p>

                      {hasPlannerPrompt && (
                        <>
                          {/* Planner prompt */}
                          <div className="rounded-md border border-border bg-muted/30 p-2.5">
                            <span className="text-[0.6rem] font-medium text-muted-foreground uppercase tracking-wide block mb-1">
                              Planner Instructions
                            </span>
                            <p className="text-xs text-foreground leading-relaxed">
                              {task.plannerPrompt}
                            </p>
                          </div>

                          {/* Planning vs Execution agents */}
                          <div className="flex flex-col gap-1.5 text-xs">
                            {task.planner && (
                              <div className="flex items-center gap-2">
                                <span className="text-muted-foreground text-[0.6rem] font-medium uppercase tracking-wide w-16">
                                  Planning:
                                </span>
                                <WorkerBadges workers={[task.planner]} />
                              </div>
                            )}
                            {task.workers && task.workers.length > 0 && (
                              <div className="flex items-center gap-2">
                                <span className="text-muted-foreground text-[0.6rem] font-medium uppercase tracking-wide w-16">
                                  Execution:
                                </span>
                                <WorkerBadges workers={task.workers} />
                              </div>
                            )}
                          </div>
                        </>
                      )}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      ))}
    </div>
  );
}

export function PlanningApproval({
  inboxItem,
  planningData,
  onApprove,
  onRequestChanges,
}: PlanningApprovalProps) {
  const [showFeedbackInput, setShowFeedbackInput] = useState(false);
  const [feedback, setFeedback] = useState("");
  const [isSubmitted, setIsSubmitted] = useState(inboxItem.status === "resolved");
  const [approvedState, setApprovedState] = useState<"approved" | "changes_requested" | null>(null);

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
        <div className="rounded-md bg-indigo-500/10 p-1.5">
          <FileText className="size-4 text-indigo-400" />
        </div>
        <div className="min-w-0 flex-1">
          <h3 className="text-sm font-semibold truncate">{inboxItem.title}</h3>
          <p className="text-xs text-muted-foreground truncate">{inboxItem.description}</p>
        </div>
        <Badge className="bg-indigo-500/20 text-indigo-400 border-indigo-500/30 text-xs">
          Planning Review
        </Badge>
      </div>

      {/* Issue context */}
      <div className="space-y-1.5">
        <div className="flex items-center gap-1.5">
          <Bot className="size-3.5 text-muted-foreground" />
          <span className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
            Issue Overseer: {inboxItem.sourceName}
          </span>
        </div>
        <div className="rounded-lg border border-border bg-muted/30 p-3 space-y-2">
          <h4 className="font-semibold text-sm">{planningData.issueTitle}</h4>
          <p className="text-xs text-muted-foreground leading-relaxed">
            {planningData.issueDescription}
          </p>
        </div>
      </div>

      {/* Plan summary */}
      {planningData.planSummary && (
        <div className="space-y-1.5">
          <span className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
            Plan Summary
          </span>
          <div className="rounded-md border border-border bg-muted/30 p-3">
            <p className="text-xs text-foreground leading-relaxed">{planningData.planSummary}</p>
          </div>
        </div>
      )}

      {/* Plan stats */}
      <div className="grid grid-cols-3 gap-2">
        <div className="rounded-lg border border-border bg-muted/30 p-2 text-center">
          <div className="flex items-center justify-center gap-1 text-muted-foreground mb-0.5">
            <Layers className="size-3" />
          </div>
          <div className="text-lg font-semibold">{planningData.totalTasks}</div>
          <div className="text-[0.6rem] text-muted-foreground uppercase">Tasks</div>
        </div>
        <div className="rounded-lg border border-border bg-muted/30 p-2 text-center">
          <div className="flex items-center justify-center gap-1 text-muted-foreground mb-0.5">
            <GitBranch className="size-3" />
          </div>
          <div className="text-lg font-semibold">{planningData.parallelGroups}</div>
          <div className="text-[0.6rem] text-muted-foreground uppercase">Steps</div>
        </div>
        <div className="rounded-lg border border-border bg-muted/30 p-2 text-center">
          <div className="flex items-center justify-center gap-1 text-muted-foreground mb-0.5">
            <Clock className="size-3" />
          </div>
          <div className="text-lg font-semibold">{planningData.estimatedDuration ?? "—"}</div>
          <div className="text-[0.6rem] text-muted-foreground uppercase">Est.</div>
        </div>
      </div>

      {/* Key considerations */}
      {planningData.considerations && planningData.considerations.length > 0 && (
        <div className="space-y-1.5">
          <span className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
            Key Considerations
          </span>
          <ul className="space-y-1">
            {planningData.considerations.map((item, i) => (
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
        <TaskDependencyGraph tasks={planningData.tasks} />
      </div>

      {/* Feedback input */}
      {showFeedbackInput && !isSubmitted && (
        <div className="space-y-1.5 pt-2 border-t">
          <label className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
            Feedback for changes
          </label>
          <Textarea
            placeholder="Describe what changes you'd like to see in this plan..."
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
                  Plan approved — execution will begin shortly
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
                Approve Plan
              </Button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
