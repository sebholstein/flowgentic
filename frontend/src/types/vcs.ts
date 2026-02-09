// VCS Strategy - how changes are organized
export type VCSStrategy =
  | "none" // No VCS handling, raw changes in folder
  | "stacked_branches" // Stacked branches (each task branches from previous)
  | "worktree"; // Git worktrees (isolated directories)

// VCS context for a task
export interface TaskVCSContext {
  strategy: VCSStrategy;

  // Branch info (for stacked_branches strategy)
  branch?: string; // e.g., "feat/auth-login-ui"
  baseBranch?: string; // e.g., "feat/auth-setup" or "main"

  // Worktree info (for worktree strategy)
  worktree?: string; // e.g., "../project-auth-login"
  worktreePath?: string; // Full path to worktree

  // Common
  isStale?: boolean; // Branch/worktree needs rebase
  hasConflicts?: boolean; // Merge conflicts detected
  aheadBy?: number; // Commits ahead of base
  behindBy?: number; // Commits behind base
}

// VCS context at thread level
export interface ThreadVCSContext {
  strategy: VCSStrategy;

  // For stacked branches: the root branch all tasks branch from
  rootBranch?: string; // e.g., "main" or "develop"

  // For worktrees: the base directory
  worktreeBaseDir?: string; // e.g., "../project-worktrees"

  // Aggregate status
  totalBranches?: number;
  mergedBranches?: number;
  conflictedBranches?: number;
}

// Strategy display configuration
export interface VCSStrategyConfig {
  label: string;
  shortLabel: string;
  description: string;
  icon: "folder" | "git-branch" | "git-fork";
  color: string;
  bgColor: string;
}

// Legacy alias for backwards compatibility
export type IssueVCSContext = ThreadVCSContext;

export const VCS_STRATEGY_CONFIG: Record<VCSStrategy, VCSStrategyConfig> = {
  none: {
    label: "No VCS",
    shortLabel: "Raw",
    description: "Changes made directly in the working directory",
    icon: "folder",
    color: "text-slate-400",
    bgColor: "bg-slate-400/10",
  },
  stacked_branches: {
    label: "Stacked Branches",
    shortLabel: "Stacked",
    description: "Each task creates a branch stacked on the previous",
    icon: "git-branch",
    color: "text-cyan-400",
    bgColor: "bg-cyan-400/10",
  },
  worktree: {
    label: "Worktrees",
    shortLabel: "Worktree",
    description: "Each task uses an isolated git worktree",
    icon: "git-fork",
    color: "text-violet-400",
    bgColor: "bg-violet-400/10",
  },
};
