import { memo } from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import {
  Circle,
  GitCompare,
  ArrowDownToLine,
  ArrowUpFromLine,
  ClipboardList,
  ClipboardCheck,
  ClipboardX,
  Loader2,
} from "lucide-react";
import type { ResourceCompletionStatus, ResourceRef } from "@/types/resource";
import type { TaskVCSContext } from "@/types/vcs";
import { ResourceTypeIcon } from "@/components/resources/ResourceBadge";
import { VCSBadgeCompact } from "@/components/vcs/VCSBadge";

type TaskStatus = "pending" | "running" | "completed" | "failed" | "blocked" | "needs_feedback";

interface Subtask {
  id: string;
  name: string;
  completed: boolean;
}

type TaskPlanStatus =
  | "pending"
  | "in_progress"
  | "awaiting_approval"
  | "approved"
  | "rejected"
  | "skipped";

interface Task {
  id: string;
  name: string;
  description: string;
  status: TaskStatus;
  duration?: string;
  dependencies: string[];
  agent?: string;
  startedAt?: string;
  completedAt?: string;
  subtasks?: Subtask[];
  executionCount?: number;
  // Resource support
  resourceCompletionStatus?: ResourceCompletionStatus;
  availableResources?: ResourceRef[];
  // VCS context
  vcs?: TaskVCSContext;
  // Planning phase
  planStatus?: TaskPlanStatus;
}

interface StatusConfig {
  icon: typeof Circle;
  color: string;
  bgColor: string;
  label: string;
}

export interface TaskNodeData {
  task: Task;
  statusConfig: StatusConfig;
}

function TaskNodeComponent({ data, selected }: NodeProps<TaskNodeData>) {
  const { task, statusConfig } = data;
  const StatusIcon = statusConfig.icon;

  const borderColor = {
    pending: "border-border",
    running: "border-blue-500 shadow-blue-500/20 shadow-lg",
    completed: "border-emerald-500/50",
    failed: "border-red-500/50",
    blocked: "border-amber-500/50",
    needs_feedback: "border-purple-500 shadow-purple-500/20 shadow-lg",
  }[task.status];

  const subtasksDone = task.subtasks?.filter((s: Subtask) => s.completed).length ?? 0;
  const subtasksTotal = task.subtasks?.length ?? 0;
  const hasSubtasks = subtasksTotal > 0;
  const hasMultipleExecutions = (task.executionCount ?? 0) > 1;

  // Resource completion status
  const resourceStatus = task.resourceCompletionStatus;
  const hasResourceRequirements =
    resourceStatus && (resourceStatus.inputsTotal > 0 || resourceStatus.outputsTotal > 0);

  // Get unique resource types for display
  const resourceTypes: ResourceRef["type"][] = task.availableResources
    ? [...new Set(task.availableResources.map((r: ResourceRef) => r.type))]
    : [];

  return (
    <>
      <Handle
        type="target"
        position={Position.Top}
        className="!w-3 !h-3 !bg-muted-foreground !border-2 !border-border !-top-1.5"
      />
      <div
        className={cn(
          "w-[280px] rounded-lg border bg-card/95 backdrop-blur transition-all cursor-pointer shadow-sm",
          borderColor,
          selected && "ring-2 ring-blue-400 ring-offset-2 ring-offset-background",
        )}
      >
        <div className="flex items-start gap-2.5 p-3">
          <StatusIcon
            className={cn(
              "size-4 mt-0.5 flex-shrink-0",
              statusConfig.color,
              task.status === "running" && "animate-spin",
            )}
          />
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span className="font-medium text-sm text-foreground truncate">{task.name}</span>
              {hasMultipleExecutions && (
                <Badge className="bg-purple-500/20 text-purple-400 border-purple-500/30 text-[0.55rem] px-1 h-4">
                  <GitCompare className="size-2.5 mr-0.5" />
                  {task.executionCount}
                </Badge>
              )}
            </div>
            <p className="text-[0.65rem] text-muted-foreground line-clamp-2 mt-0.5 leading-relaxed">
              {task.description}
            </p>
          </div>
        </div>
        {hasSubtasks && (
          <div className="px-3 pb-2">
            <div className="flex items-center gap-2">
              <div className="flex-1 h-1.5 bg-muted rounded-full overflow-hidden">
                <div
                  className={cn(
                    "h-full rounded-full transition-all",
                    subtasksDone === subtasksTotal ? "bg-emerald-500" : "bg-blue-500",
                  )}
                  style={{ width: `${(subtasksDone / subtasksTotal) * 100}%` }}
                />
              </div>
              <span className="text-[0.6rem] text-muted-foreground tabular-nums">
                {subtasksDone}/{subtasksTotal}
              </span>
            </div>
          </div>
        )}
        <div className="flex items-center justify-between gap-2 border-t border-border px-3 py-2">
          <div className="flex items-center gap-1.5">
            {/* Plan status indicator */}
            {task.planStatus &&
              task.planStatus !== "skipped" &&
              (() => {
                const planIcons: Record<
                  string,
                  { icon: typeof ClipboardList; color: string; title: string }
                > = {
                  pending: { icon: ClipboardList, color: "text-slate-400", title: "Plan pending" },
                  in_progress: {
                    icon: ClipboardList,
                    color: "text-blue-400",
                    title: "Planning...",
                  },
                  awaiting_approval: {
                    icon: ClipboardList,
                    color: "text-orange-400",
                    title: "Plan ready",
                  },
                  approved: {
                    icon: ClipboardCheck,
                    color: "text-emerald-400",
                    title: "Plan approved",
                  },
                  rejected: { icon: ClipboardX, color: "text-red-400", title: "Plan rejected" },
                };
                const config = planIcons[task.planStatus];
                if (!config) return null;
                const PlanIcon = config.icon;
                return (
                  <div className="flex items-center gap-0.5" title={config.title}>
                    <PlanIcon className={cn("size-3", config.color)} />
                    {task.planStatus === "in_progress" && (
                      <Loader2 className="size-2.5 text-blue-400 animate-spin" />
                    )}
                    {task.planStatus === "awaiting_approval" && (
                      <span className="size-1.5 rounded-full bg-orange-400" />
                    )}
                  </div>
                );
              })()}
            {/* VCS context badge */}
            {task.vcs && task.vcs.strategy !== "none" && <VCSBadgeCompact vcs={task.vcs} />}
            {task.agent && (
              <Badge variant="outline" className="text-[0.6rem]">
                {task.agent}
              </Badge>
            )}
            {/* Resource requirement badges */}
            {hasResourceRequirements && (
              <>
                {resourceStatus.inputsTotal > 0 && (
                  <div
                    className={cn(
                      "flex items-center gap-0.5 text-[0.55rem] px-1 py-0.5 rounded border",
                      resourceStatus.inputsFulfilled === resourceStatus.inputsTotal
                        ? "text-emerald-400 bg-emerald-400/10 border-emerald-500/30"
                        : resourceStatus.inputsFulfilled > 0
                          ? "text-amber-400 bg-amber-400/10 border-amber-500/30"
                          : "text-red-400 bg-red-400/10 border-red-500/30",
                    )}
                    title={`Inputs: ${resourceStatus.inputsFulfilled}/${resourceStatus.inputsTotal}`}
                  >
                    <ArrowDownToLine className="size-2.5" />
                    <span className="tabular-nums">
                      {resourceStatus.inputsFulfilled}/{resourceStatus.inputsTotal}
                    </span>
                  </div>
                )}
                {resourceStatus.outputsTotal > 0 && (
                  <div
                    className={cn(
                      "flex items-center gap-0.5 text-[0.55rem] px-1 py-0.5 rounded border",
                      resourceStatus.outputsFulfilled === resourceStatus.outputsTotal
                        ? "text-emerald-400 bg-emerald-400/10 border-emerald-500/30"
                        : resourceStatus.outputsFulfilled > 0
                          ? "text-amber-400 bg-amber-400/10 border-amber-500/30"
                          : "text-slate-400 bg-slate-400/10 border-slate-500/30",
                    )}
                    title={`Outputs: ${resourceStatus.outputsFulfilled}/${resourceStatus.outputsTotal}`}
                  >
                    <ArrowUpFromLine className="size-2.5" />
                    <span className="tabular-nums">
                      {resourceStatus.outputsFulfilled}/{resourceStatus.outputsTotal}
                    </span>
                  </div>
                )}
              </>
            )}
            {/* Resource type icons */}
            {resourceTypes.length > 0 && (
              <div className="flex items-center gap-0.5">
                {resourceTypes.slice(0, 3).map((type) => (
                  <ResourceTypeIcon key={type} type={type} size="sm" />
                ))}
                {resourceTypes.length > 3 && (
                  <span className="text-[0.5rem] text-muted-foreground">
                    +{resourceTypes.length - 3}
                  </span>
                )}
              </div>
            )}
          </div>
          {task.duration ? (
            <span className="text-[0.6rem] text-muted-foreground ml-auto">{task.duration}</span>
          ) : task.status === "running" && task.startedAt ? (
            <span className="text-[0.6rem] text-blue-400 ml-auto animate-pulse">
              Started {task.startedAt}
            </span>
          ) : task.status === "needs_feedback" ? (
            <span className="text-[0.6rem] text-purple-400 ml-auto">Awaiting review</span>
          ) : (
            <span className="text-[0.6rem] text-muted-foreground ml-auto capitalize">
              {task.status.replace("_", " ")}
            </span>
          )}
        </div>
      </div>
      <Handle
        type="source"
        position={Position.Bottom}
        className="!w-3 !h-3 !bg-muted-foreground !border-2 !border-border !-bottom-1.5"
      />
    </>
  );
}

export const TaskNode = memo(TaskNodeComponent);
