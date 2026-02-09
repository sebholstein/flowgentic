import { Link } from "@tanstack/react-router";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger, TooltipProvider } from "@/components/ui/tooltip";
import {
  ChevronRight,
  ChevronDown,
  Calendar,
  ExternalLink,
  GitCompare,
  Timer,
  AlertTriangle,
  Compass,
  MessageCircle,
  ArrowUpRight,
  ClipboardList,
} from "lucide-react";
import type { Task, TaskStatus } from "@/types/task";
import { ResourceRequirementList } from "@/components/resources/ResourceRequirementList";
import { VCSBadgeFull } from "@/components/vcs/VCSBadge";

// Status config type - imported from parent or defined here
interface StatusConfig {
  icon: React.ComponentType<{ className?: string }>;
  color: string;
  bgColor: string;
  label: string;
}

const taskStatusConfig: Record<TaskStatus, StatusConfig> = {
  pending: {
    icon: () => <div className="size-3 rounded-full border-2 border-slate-400" />,
    color: "text-slate-400",
    bgColor: "bg-slate-400/10",
    label: "Pending",
  },
  running: {
    icon: ({ className }: { className?: string }) => (
      <div
        className={cn(
          "size-3 rounded-full border-2 border-blue-400 border-t-transparent animate-spin",
          className,
        )}
      />
    ),
    color: "text-blue-400",
    bgColor: "bg-blue-400/10",
    label: "Running",
  },
  completed: {
    icon: ({ className }: { className?: string }) => (
      <svg
        className={cn("size-3", className)}
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="3"
      >
        <path d="M20 6L9 17l-5-5" />
      </svg>
    ),
    color: "text-emerald-400",
    bgColor: "bg-emerald-400/10",
    label: "Completed",
  },
  failed: {
    icon: ({ className }: { className?: string }) => (
      <svg
        className={cn("size-3", className)}
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="3"
      >
        <circle cx="12" cy="12" r="10" />
        <path d="M12 8v4M12 16h.01" />
      </svg>
    ),
    color: "text-red-400",
    bgColor: "bg-red-400/10",
    label: "Failed",
  },
  blocked: {
    icon: ({ className }: { className?: string }) => (
      <svg
        className={cn("size-3", className)}
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
      >
        <circle cx="12" cy="12" r="10" />
        <polyline points="12 6 12 12 16 14" />
      </svg>
    ),
    color: "text-amber-400",
    bgColor: "bg-amber-400/10",
    label: "Blocked",
  },
  needs_feedback: {
    icon: MessageCircle,
    color: "text-purple-400",
    bgColor: "bg-purple-400/10",
    label: "Needs Feedback",
  },
};

interface TaskDetailCardProps {
  task: Task;
  tasks: Task[];
  threadId: string;
  isExpanded: boolean;
  onToggle: () => void;
  onSelectTask: (taskId: string) => void;
}

/**
 * Expandable card showing task details in the thread detail view.
 * Shows status, progress, dependencies, and links to full task details.
 */
export function TaskDetailCard({
  task,
  tasks,
  threadId,
  isExpanded,
  onToggle,
  onSelectTask,
}: TaskDetailCardProps) {
  const statusConfig = taskStatusConfig[task.status];
  const StatusIcon = statusConfig.icon;
  const subtasksDone = task.subtasks?.filter((s) => s.completed).length ?? 0;
  const subtasksTotal = task.subtasks?.length ?? 0;
  const hasSubtasks = subtasksTotal > 0;
  const dependencyTasks = task.dependencies
    .map((id) => tasks.find((t) => t.id === id))
    .filter(Boolean) as Task[];
  const dependentTasks = tasks.filter((t) => t.dependencies.includes(task.id));
  const hasExecutions = (task.executions?.length ?? 0) > 0;
  const hasResources =
    (task.inputRequirements?.length ?? 0) > 0 || (task.outputRequirements?.length ?? 0) > 0;

  return (
    <div
      className={cn(
        "rounded-lg border border-border transition-all",
        isExpanded && "ring-1 ring-border",
      )}
    >
      {/* Task Header */}
      <button
        onClick={onToggle}
        className="flex w-full items-center gap-2.5 px-3 py-2.5 text-left hover:bg-muted/50 rounded-t-lg transition-colors"
      >
        <StatusIcon
          className={cn(
            "size-3.5 shrink-0",
            statusConfig.color,
            task.status === "running" && "animate-spin",
          )}
        />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-1.5 flex-wrap">
            <span className="font-medium text-sm">{task.name}</span>
            {task.agent && (
              <Badge variant="outline" className="text-[0.6rem] py-0 h-4">
                {task.agent}
              </Badge>
            )}
            {hasExecutions && (
              <Badge className="bg-purple-500/20 text-purple-400 border-purple-500/30 text-[0.6rem] py-0 h-4">
                <GitCompare className="size-2.5 mr-0.5" />
                {task.executions?.length}
              </Badge>
            )}
            {/* Plan status badge */}
            {task.planStatus && task.planStatus !== "skipped" && task.planStatus !== "pending" && (
              <Badge
                className={cn(
                  "text-[0.6rem] py-0 h-4 gap-0.5",
                  task.planStatus === "in_progress" &&
                    "bg-blue-500/20 text-blue-400 border-blue-500/30",
                  task.planStatus === "awaiting_approval" &&
                    "bg-orange-500/20 text-orange-400 border-orange-500/30",
                  task.planStatus === "approved" &&
                    "bg-emerald-500/20 text-emerald-400 border-emerald-500/30",
                  task.planStatus === "rejected" && "bg-red-500/20 text-red-400 border-red-500/30",
                )}
              >
                <ClipboardList className="size-2.5" />
                {task.planStatus === "in_progress" && "Planning..."}
                {task.planStatus === "awaiting_approval" && "Plan ready"}
                {task.planStatus === "approved" && "Plan approved"}
                {task.planStatus === "rejected" && "Plan rejected"}
              </Badge>
            )}
            {/* Check-in indicator badge */}
            {task.checkIn && (
              <TooltipProvider delayDuration={300}>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Link
                      to="/app/inbox/$itemId"
                      params={{ itemId: task.checkIn.id }}
                      onClick={(e) => e.stopPropagation()}
                      className={cn(
                        "flex items-center gap-1 text-[0.6rem] py-0 h-4 px-1.5 rounded-md border",
                        task.checkIn.type === "decision_escalation" &&
                          "bg-amber-500/20 text-amber-400 border-amber-500/30",
                        task.checkIn.type === "direction_clarification" &&
                          "bg-blue-500/20 text-blue-400 border-blue-500/30",
                        task.checkIn.type === "questionnaire" &&
                          "bg-violet-500/20 text-violet-400 border-violet-500/30",
                        ![
                          "decision_escalation",
                          "direction_clarification",
                          "questionnaire",
                        ].includes(task.checkIn.type) &&
                          "bg-purple-500/20 text-purple-400 border-purple-500/30",
                      )}
                    >
                      {task.checkIn.type === "decision_escalation" && (
                        <AlertTriangle className="size-2.5" />
                      )}
                      {task.checkIn.type === "direction_clarification" && (
                        <Compass className="size-2.5" />
                      )}
                      {task.checkIn.type === "questionnaire" && (
                        <MessageCircle className="size-2.5" />
                      )}
                      {![
                        "decision_escalation",
                        "direction_clarification",
                        "questionnaire",
                      ].includes(task.checkIn.type) && <MessageCircle className="size-2.5" />}
                      <span>
                        {task.checkIn.type === "decision_escalation"
                          ? "Decision needed"
                          : task.checkIn.type === "direction_clarification"
                            ? "Needs guidance"
                            : "Check-in"}
                      </span>
                    </Link>
                  </TooltipTrigger>
                  <TooltipContent side="top">
                    <p>{task.checkIn.title}</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          {hasSubtasks && (
            <div className="flex items-center gap-1.5">
              <div className="w-12 h-1 bg-muted rounded-full overflow-hidden">
                <div
                  className={cn(
                    "h-full rounded-full transition-all",
                    subtasksDone === subtasksTotal ? "bg-emerald-500" : "bg-blue-500",
                  )}
                  style={{ width: `${(subtasksDone / subtasksTotal) * 100}%` }}
                />
              </div>
              <span className="text-[0.65rem] text-muted-foreground tabular-nums">
                {subtasksDone}/{subtasksTotal}
              </span>
            </div>
          )}
          {task.duration && (
            <span className="text-[0.65rem] text-muted-foreground">{task.duration}</span>
          )}
          {/* Quick jump to task detail */}
          <TooltipProvider delayDuration={300}>
            <Tooltip>
              <TooltipTrigger asChild>
                <Link
                  to="/app/tasks/$threadId/$taskId"
                  params={{ threadId, taskId: task.id }}
                  onClick={(e) => e.stopPropagation()}
                  className="p-1 rounded hover:bg-muted transition-colors"
                >
                  <ArrowUpRight className="size-3.5 text-muted-foreground hover:text-foreground" />
                </Link>
              </TooltipTrigger>
              <TooltipContent side="top">
                <p>Open task details</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
          {isExpanded ? (
            <ChevronDown className="size-3.5 text-muted-foreground" />
          ) : (
            <ChevronRight className="size-3.5 text-muted-foreground" />
          )}
        </div>
      </button>

      {/* Expanded Content - Compact Overview */}
      {isExpanded && (
        <div className="border-t border-border/50 px-3 py-2.5 space-y-2.5">
          {/* Description */}
          <p className="text-sm text-muted-foreground line-clamp-2">{task.description}</p>

          {/* Plan summary (when plan exists) */}
          {task.plan && (
            <div className="rounded-md border border-border bg-muted/30 p-2.5">
              <div className="flex items-center gap-1.5 mb-1">
                <ClipboardList className="size-3 text-muted-foreground" />
                <span className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
                  Plan
                </span>
              </div>
              <p className="text-xs text-muted-foreground line-clamp-2">{task.plan.summary}</p>
            </div>
          )}

          {/* VCS Context */}
          {task.vcs && <VCSBadgeFull vcs={task.vcs} />}

          {/* Quick Stats Row */}
          <div className="flex flex-wrap items-center gap-2 text-xs">
            {task.startedAt && (
              <span className="flex items-center gap-1 text-muted-foreground">
                <Calendar className="size-3" />
                {task.startedAt}
              </span>
            )}
            {task.duration && (
              <span className="flex items-center gap-1 text-muted-foreground">
                <Timer className="size-3" />
                {task.duration}
              </span>
            )}
            {/* Resource compact badges */}
            {hasResources && (
              <ResourceRequirementList
                inputRequirements={task.inputRequirements}
                outputRequirements={task.outputRequirements}
                availableResources={task.availableResources}
                compact
              />
            )}
          </div>

          {/* Dependencies - compact */}
          {(dependencyTasks.length > 0 || dependentTasks.length > 0) && (
            <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
              {dependencyTasks.length > 0 && (
                <div className="flex items-center gap-1">
                  <span className="text-muted-foreground">Depends on:</span>
                  {dependencyTasks.slice(0, 2).map((dep) => {
                    const depStatus = taskStatusConfig[dep.status];
                    const DepIcon = depStatus.icon;
                    return (
                      <button
                        key={dep.id}
                        onClick={(e) => {
                          e.stopPropagation();
                          onSelectTask(dep.id);
                        }}
                        className="flex items-center gap-1 rounded border border-border px-1.5 py-0.5 hover:bg-muted transition-colors"
                      >
                        <DepIcon className={cn("size-2.5", depStatus.color)} />
                        <span className="max-w-[80px] truncate">{dep.name}</span>
                      </button>
                    );
                  })}
                  {dependencyTasks.length > 2 && (
                    <span className="text-muted-foreground">+{dependencyTasks.length - 2}</span>
                  )}
                </div>
              )}
              {dependentTasks.length > 0 && (
                <span className="text-muted-foreground">
                  Blocks {dependentTasks.length} task{dependentTasks.length > 1 ? "s" : ""}
                </span>
              )}
            </div>
          )}

          {/* View Details Link */}
          <Link
            to="/app/tasks/$threadId/$taskId"
            params={{ threadId, taskId: task.id }}
            onClick={(e) => e.stopPropagation()}
            className="flex items-center justify-center gap-1.5 w-full rounded border border-border bg-muted/50 px-2 py-1.5 text-xs text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
          >
            <span>View details</span>
            <ExternalLink className="size-3" />
          </Link>
        </div>
      )}
    </div>
  );
}
