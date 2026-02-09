import type {
  InputRequirement,
  OutputRequirement,
  ResourceRef,
  ResourceCompletionStatus,
} from "@/types/resource";
import type { TaskVCSContext } from "@/types/vcs";

export type TaskStatus =
  | "pending"
  | "running"
  | "completed"
  | "failed"
  | "blocked"
  | "needs_feedback";

export type InboxItemType =
  | "execution_selection"
  | "thread_review"
  | "planning_approval"
  | "task_plan_approval"
  | "questionnaire"
  | "decision_escalation"
  | "direction_clarification";

export type TaskPlanStatus =
  | "pending" // planning not started yet
  | "in_progress" // planner agent is working
  | "awaiting_approval" // plan ready, waiting for approval
  | "approved" // plan approved, ready for execution
  | "rejected" // plan rejected, needs re-planning
  | "skipped"; // no planning needed for this task

export interface TaskPlan {
  summary: string;
  steps: string[];
  approach?: string;
  considerations?: string[];
  estimatedComplexity?: "low" | "medium" | "high";
  agentId: string;
  agentName: string;
  createdAt: string;
}

export interface TaskCheckIn {
  id: string;
  type: InboxItemType;
  title: string;
  priority: "low" | "medium" | "high";
}

export interface Subtask {
  id: string;
  name: string;
  completed: boolean;
}

export interface TaskExecution {
  id: string;
  agentId: string;
  agentName: string;
  status: "running" | "completed" | "failed";
  duration?: string;
  tokens?: {
    input: number;
    output: number;
    total: number;
  };
}

export interface Task {
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
  executions?: TaskExecution[];
  selectedExecutionId?: string;
  feedbackItemId?: string;
  checkIn?: TaskCheckIn;
  inputRequirements?: InputRequirement[];
  outputRequirements?: OutputRequirement[];
  availableResources?: ResourceRef[];
  resourceCompletionStatus?: ResourceCompletionStatus;
  vcs?: TaskVCSContext;

  // Planning phase (optional)
  plannerPrompt?: string;
  planApproval?: "user" | "overseer" | "auto";
  planStatus?: TaskPlanStatus;
  plan?: TaskPlan;
  planner?: { id: string; name: string };
}
