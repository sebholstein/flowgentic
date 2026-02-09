import { memo } from "react";
import { cn } from "@/lib/utils";
import {
  GitBranch,
  GitFork,
  Folder,
  ArrowRight,
  AlertTriangle,
  RefreshCw,
  ArrowUp,
  ArrowDown,
} from "lucide-react";
import { Tooltip, TooltipContent, TooltipTrigger, TooltipProvider } from "@/components/ui/tooltip";
import type { TaskVCSContext, VCSStrategy } from "@/types/vcs";
import { VCS_STRATEGY_CONFIG } from "@/types/vcs";

// ============================================================================
// Strategy Icon
// ============================================================================

interface StrategyIconProps {
  strategy: VCSStrategy;
  className?: string;
}

export function StrategyIcon({ strategy, className }: StrategyIconProps) {
  const config = VCS_STRATEGY_CONFIG[strategy];
  const Icon = {
    folder: Folder,
    "git-branch": GitBranch,
    "git-fork": GitFork,
  }[config.icon];

  return <Icon className={cn("size-3.5", config.color, className)} />;
}

// ============================================================================
// Compact VCS Badge (for TaskNode in graph view)
// ============================================================================

interface VCSBadgeCompactProps {
  vcs: TaskVCSContext;
  className?: string;
}

function VCSBadgeCompactComponent({ vcs, className }: VCSBadgeCompactProps) {
  const config = VCS_STRATEGY_CONFIG[vcs.strategy];

  if (vcs.strategy === "none") {
    return null; // Don't show anything for "none" strategy in compact view
  }

  const branchName = vcs.branch || vcs.worktree;
  const displayName = branchName
    ? branchName.length > 16
      ? branchName.slice(0, 14) + "…"
      : branchName
    : null;

  return (
    <TooltipProvider delayDuration={300}>
      <Tooltip>
        <TooltipTrigger asChild>
          <div
            className={cn(
              "flex items-center gap-1 text-[0.55rem] px-1.5 py-0.5 rounded border",
              config.bgColor,
              config.color,
              vcs.hasConflicts && "border-red-500/50",
              vcs.isStale && !vcs.hasConflicts && "border-amber-500/50",
              !vcs.hasConflicts && !vcs.isStale && "border-current/30",
              className,
            )}
          >
            <StrategyIcon strategy={vcs.strategy} className="size-2.5" />
            {displayName && <span className="font-mono truncate max-w-[80px]">{displayName}</span>}
            {vcs.hasConflicts && <AlertTriangle className="size-2.5 text-red-400" />}
            {vcs.isStale && !vcs.hasConflicts && <RefreshCw className="size-2.5 text-amber-400" />}
          </div>
        </TooltipTrigger>
        <TooltipContent side="bottom" className="max-w-xs">
          <VCSTooltipContent vcs={vcs} />
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

export const VCSBadgeCompact = memo(VCSBadgeCompactComponent);

// ============================================================================
// Full VCS Badge (for TaskDetailCard)
// ============================================================================

interface VCSBadgeFullProps {
  vcs: TaskVCSContext;
  className?: string;
}

function VCSBadgeFullComponent({ vcs, className }: VCSBadgeFullProps) {
  const config = VCS_STRATEGY_CONFIG[vcs.strategy];

  return (
    <div
      className={cn(
        "flex items-center gap-2 rounded-md border px-2.5 py-1.5",
        config.bgColor,
        vcs.hasConflicts && "border-red-500/40",
        vcs.isStale && !vcs.hasConflicts && "border-amber-500/40",
        !vcs.hasConflicts && !vcs.isStale && "border-border",
        className,
      )}
    >
      {/* Strategy indicator */}
      <div className={cn("flex items-center gap-1.5", config.color)}>
        <StrategyIcon strategy={vcs.strategy} />
        <span className="text-xs font-medium">{config.shortLabel}</span>
      </div>

      {/* Branch flow visualization */}
      {(vcs.branch || vcs.worktree) && (
        <>
          <div className="h-4 w-px bg-border" />
          <div className="flex items-center gap-1.5 text-xs">
            {/* Base branch */}
            {vcs.baseBranch && (
              <>
                <span className="font-mono text-muted-foreground">{vcs.baseBranch}</span>
                <ArrowRight className="size-3 text-muted-foreground" />
              </>
            )}
            {/* Current branch */}
            <span className="font-mono text-foreground font-medium">
              {vcs.branch || vcs.worktree}
            </span>
          </div>
        </>
      )}

      {/* Status indicators */}
      {(vcs.aheadBy || vcs.behindBy || vcs.hasConflicts || vcs.isStale) && (
        <>
          <div className="h-4 w-px bg-border" />
          <div className="flex items-center gap-2">
            {vcs.aheadBy !== undefined && vcs.aheadBy > 0 && (
              <span className="flex items-center gap-0.5 text-[0.65rem] text-emerald-400">
                <ArrowUp className="size-3" />
                {vcs.aheadBy}
              </span>
            )}
            {vcs.behindBy !== undefined && vcs.behindBy > 0 && (
              <span className="flex items-center gap-0.5 text-[0.65rem] text-amber-400">
                <ArrowDown className="size-3" />
                {vcs.behindBy}
              </span>
            )}
            {vcs.hasConflicts && (
              <span className="flex items-center gap-1 text-[0.65rem] text-red-400">
                <AlertTriangle className="size-3" />
                Conflicts
              </span>
            )}
            {vcs.isStale && !vcs.hasConflicts && (
              <span className="flex items-center gap-1 text-[0.65rem] text-amber-400">
                <RefreshCw className="size-3" />
                Needs rebase
              </span>
            )}
          </div>
        </>
      )}
    </div>
  );
}

export const VCSBadgeFull = memo(VCSBadgeFullComponent);

// ============================================================================
// Branch Flow Visualization (shows stacking relationship)
// ============================================================================

interface BranchFlowProps {
  branches: Array<{
    name: string;
    taskId: string;
    taskName: string;
    status: "pending" | "active" | "merged" | "conflict";
  }>;
  baseBranch: string;
  className?: string;
}

export function BranchFlow({ branches, baseBranch, className }: BranchFlowProps) {
  return (
    <div className={cn("flex flex-col gap-1", className)}>
      {/* Base branch */}
      <div className="flex items-center gap-2 text-xs">
        <GitBranch className="size-3.5 text-emerald-400" />
        <span className="font-mono text-muted-foreground">{baseBranch}</span>
        <span className="text-muted-foreground/70">base</span>
      </div>

      {/* Stacked branches */}
      {branches.map((branch, index) => (
        <div key={branch.taskId} className="flex items-center gap-2 ml-3">
          {/* Connecting line */}
          <div className="flex flex-col items-center">
            <div className="w-px h-2 bg-border" />
            <div
              className={cn(
                "size-2 rounded-full border-2",
                branch.status === "merged" && "bg-emerald-500 border-emerald-500",
                branch.status === "active" && "bg-cyan-500 border-cyan-500",
                branch.status === "conflict" && "bg-red-500 border-red-500",
                branch.status === "pending" && "bg-transparent border-muted-foreground",
              )}
            />
            {index < branches.length - 1 && <div className="w-px h-2 bg-border" />}
          </div>

          {/* Branch info */}
          <div className="flex items-center gap-2 text-xs">
            <span
              className={cn(
                "font-mono",
                branch.status === "active" && "text-cyan-400 font-medium",
                branch.status === "merged" && "text-emerald-400",
                branch.status === "conflict" && "text-red-400",
                branch.status === "pending" && "text-muted-foreground",
              )}
            >
              {branch.name}
            </span>
            <span className="text-muted-foreground/70 truncate max-w-[120px]">
              {branch.taskName}
            </span>
          </div>
        </div>
      ))}
    </div>
  );
}

// ============================================================================
// Tooltip Content
// ============================================================================

function VCSTooltipContent({ vcs }: { vcs: TaskVCSContext }) {
  const config = VCS_STRATEGY_CONFIG[vcs.strategy];

  return (
    <div className="space-y-2 text-xs">
      <div className="flex items-center gap-2">
        <StrategyIcon strategy={vcs.strategy} />
        <span className="font-medium">{config.label}</span>
      </div>

      {(vcs.branch || vcs.worktree) && (
        <div className="space-y-1">
          {vcs.baseBranch && (
            <div className="flex items-center gap-1.5 text-muted-foreground">
              <span>From:</span>
              <code className="bg-muted px-1 rounded">{vcs.baseBranch}</code>
            </div>
          )}
          <div className="flex items-center gap-1.5">
            <span className="text-muted-foreground">
              {vcs.strategy === "worktree" ? "Worktree:" : "Branch:"}
            </span>
            <code className="bg-muted px-1 rounded font-medium">{vcs.branch || vcs.worktree}</code>
          </div>
          {vcs.worktreePath && (
            <div className="text-muted-foreground/70 font-mono text-[0.65rem]">
              {vcs.worktreePath}
            </div>
          )}
        </div>
      )}

      {(vcs.aheadBy || vcs.behindBy) && (
        <div className="flex items-center gap-3 pt-1 border-t border-border/50">
          {vcs.aheadBy !== undefined && vcs.aheadBy > 0 && (
            <span className="text-emerald-400">{vcs.aheadBy} ahead</span>
          )}
          {vcs.behindBy !== undefined && vcs.behindBy > 0 && (
            <span className="text-amber-400">{vcs.behindBy} behind</span>
          )}
        </div>
      )}

      {vcs.hasConflicts && (
        <div className="flex items-center gap-1.5 text-red-400 pt-1 border-t border-border/50">
          <AlertTriangle className="size-3" />
          Has merge conflicts
        </div>
      )}

      {vcs.isStale && !vcs.hasConflicts && (
        <div className="flex items-center gap-1.5 text-amber-400 pt-1 border-t border-border/50">
          <RefreshCw className="size-3" />
          Needs rebase from base
        </div>
      )}
    </div>
  );
}
