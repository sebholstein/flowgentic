import { memo, useMemo } from "react";
import { cn } from "@/lib/utils";
import {
  GitBranch,
  GitFork,
  Folder,
  ChevronDown,
  ChevronRight,
  Check,
  AlertTriangle,
  Clock,
} from "lucide-react";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Badge } from "@/components/ui/badge";
import type { ThreadVCSContext, TaskVCSContext, VCSStrategy } from "@/types/vcs";
import { VCS_STRATEGY_CONFIG } from "@/types/vcs";
import { useState } from "react";

// ============================================================================
// Strategy Icon
// ============================================================================

function StrategyIconLarge({ strategy }: { strategy: VCSStrategy }) {
  const Icon = {
    none: Folder,
    stacked_branches: GitBranch,
    worktree: GitFork,
  }[strategy];

  return <Icon className="size-5" />;
}

// ============================================================================
// Thread VCS Banner
// ============================================================================

interface TaskVCSInfo {
  taskId: string;
  taskName: string;
  taskStatus: "pending" | "running" | "completed" | "failed" | "blocked" | "needs_feedback";
  vcs: TaskVCSContext;
}

interface ThreadVCSBannerProps {
  threadVCS: ThreadVCSContext;
  taskVCSInfos: TaskVCSInfo[];
  className?: string;
}

function ThreadVCSBannerComponent({ threadVCS, taskVCSInfos, className }: ThreadVCSBannerProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const config = VCS_STRATEGY_CONFIG[threadVCS.strategy];

  // Calculate aggregate stats
  const stats = useMemo(() => {
    const merged = taskVCSInfos.filter((t) => t.taskStatus === "completed").length;
    const active = taskVCSInfos.filter((t) => t.taskStatus === "running").length;
    const pending = taskVCSInfos.filter(
      (t) => t.taskStatus === "pending" || t.taskStatus === "blocked",
    ).length;
    const conflicts = taskVCSInfos.filter((t) => t.vcs.hasConflicts).length;
    const stale = taskVCSInfos.filter((t) => t.vcs.isStale && !t.vcs.hasConflicts).length;

    return { merged, active, pending, conflicts, stale, total: taskVCSInfos.length };
  }, [taskVCSInfos]);

  if (threadVCS.strategy === "none") {
    return null; // Don't show banner for raw mode
  }

  return (
    <Collapsible open={isExpanded} onOpenChange={setIsExpanded}>
      <div
        className={cn(
          "rounded-lg border",
          config.bgColor,
          stats.conflicts > 0
            ? "border-red-500/40"
            : stats.stale > 0
              ? "border-amber-500/40"
              : "border-border",
          className,
        )}
      >
        {/* Header */}
        <CollapsibleTrigger asChild>
          <button className="flex w-full items-center gap-3 px-4 py-3 text-left hover:bg-muted/30 transition-colors rounded-lg">
            <div className={cn("rounded-md p-2", config.bgColor, config.color)}>
              <StrategyIconLarge strategy={threadVCS.strategy} />
            </div>

            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <span className={cn("font-medium text-sm", config.color)}>{config.label}</span>
                {threadVCS.rootBranch && (
                  <Badge variant="outline" className="text-[0.65rem] font-mono">
                    {threadVCS.rootBranch}
                  </Badge>
                )}
              </div>
              <p className="text-xs text-muted-foreground mt-0.5">{config.description}</p>
            </div>

            {/* Stats */}
            <div className="flex items-center gap-3 text-xs">
              {stats.conflicts > 0 && (
                <span className="flex items-center gap-1 text-red-400">
                  <AlertTriangle className="size-3.5" />
                  {stats.conflicts}
                </span>
              )}
              {stats.stale > 0 && (
                <span className="flex items-center gap-1 text-amber-400">
                  <Clock className="size-3.5" />
                  {stats.stale}
                </span>
              )}
              <span className="flex items-center gap-1 text-emerald-400">
                <Check className="size-3.5" />
                {stats.merged}/{stats.total}
              </span>
            </div>

            {isExpanded ? (
              <ChevronDown className="size-4 text-muted-foreground" />
            ) : (
              <ChevronRight className="size-4 text-muted-foreground" />
            )}
          </button>
        </CollapsibleTrigger>

        {/* Expanded: Branch visualization */}
        <CollapsibleContent>
          <div className="border-t border-border/50 px-4 py-3">
            <BranchStackVisualization
              rootBranch={threadVCS.rootBranch || "main"}
              tasks={taskVCSInfos}
            />
          </div>
        </CollapsibleContent>
      </div>
    </Collapsible>
  );
}

export const ThreadVCSBanner = memo(ThreadVCSBannerComponent);

// Legacy export for backwards compatibility
export const IssueVCSBanner = ThreadVCSBanner;

// ============================================================================
// Branch Stack Visualization
// ============================================================================

interface BranchStackVisualizationProps {
  rootBranch: string;
  tasks: TaskVCSInfo[];
}

function BranchStackVisualization({ rootBranch, tasks }: BranchStackVisualizationProps) {
  // Sort tasks by their branch stacking order (based on baseBranch relationships)
  const sortedTasks = useMemo(() => {
    // Simple sort: completed first, then running, then pending
    return [...tasks].sort((a, b) => {
      const order = {
        completed: 0,
        running: 1,
        needs_feedback: 2,
        pending: 3,
        blocked: 4,
        failed: 5,
      };
      return (order[a.taskStatus] ?? 99) - (order[b.taskStatus] ?? 99);
    });
  }, [tasks]);

  return (
    <div className="space-y-1">
      {/* Root branch */}
      <div className="flex items-center gap-2">
        <div className="flex items-center justify-center w-5">
          <div className="size-2.5 rounded-full bg-emerald-500 ring-2 ring-emerald-500/30" />
        </div>
        <GitBranch className="size-3.5 text-emerald-400" />
        <span className="font-mono text-xs text-emerald-400 font-medium">{rootBranch}</span>
        <span className="text-[0.65rem] text-muted-foreground">base</span>
      </div>

      {/* Task branches */}
      {sortedTasks.map((task, index) => {
        const branchName = task.vcs.branch || task.vcs.worktree;
        if (!branchName) return null;

        const statusColor = {
          completed: "bg-emerald-500",
          running: "bg-cyan-500",
          needs_feedback: "bg-purple-500",
          pending: "bg-slate-500",
          blocked: "bg-amber-500",
          failed: "bg-red-500",
        }[task.taskStatus];

        const textColor = {
          completed: "text-emerald-400",
          running: "text-cyan-400",
          needs_feedback: "text-purple-400",
          pending: "text-muted-foreground",
          blocked: "text-amber-400",
          failed: "text-red-400",
        }[task.taskStatus];

        return (
          <div key={task.taskId} className="flex items-center gap-2">
            {/* Connecting line and dot */}
            <div className="flex flex-col items-center w-5">
              <div className="w-px h-2 bg-border" />
              <div
                className={cn(
                  "size-2 rounded-full ring-2 ring-offset-1 ring-offset-background",
                  statusColor,
                  task.taskStatus === "running" && "animate-pulse",
                  task.vcs.hasConflicts && "ring-red-500",
                  !task.vcs.hasConflicts && "ring-current/30",
                )}
              />
              {index < sortedTasks.length - 1 && <div className="w-px h-2 bg-border" />}
            </div>

            {/* Branch info */}
            <div className="flex items-center gap-2 flex-1 min-w-0">
              <span className={cn("font-mono text-xs truncate", textColor)}>{branchName}</span>
              <span className="text-[0.65rem] text-muted-foreground truncate max-w-[140px]">
                {task.taskName}
              </span>
            </div>

            {/* Status indicators */}
            <div className="flex items-center gap-1.5">
              {task.vcs.hasConflicts && <AlertTriangle className="size-3 text-red-400" />}
              {task.vcs.isStale && !task.vcs.hasConflicts && (
                <Clock className="size-3 text-amber-400" />
              )}
              {task.vcs.aheadBy !== undefined && task.vcs.aheadBy > 0 && (
                <span className="text-[0.6rem] text-emerald-400 tabular-nums">
                  +{task.vcs.aheadBy}
                </span>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
