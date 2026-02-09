// Task status derived from executions
export type TaskStatus =
  | "pending" // no executions started
  | "running" // has running execution(s)
  | "needs_feedback" // multiple completed executions, needs selection
  | "completed" // execution selected and applied
  | "failed" // all executions failed
  | "blocked"; // dependencies not met

// Subtask within a task
export interface Subtask {
  id: string;
  name: string;
  completed: boolean;
}

// Token usage tracking
export interface TokenUsage {
  input: number;
  output: number;
  total: number;
}

// TaskExecution is an instance - one attempt at completing a task
export interface TaskExecution {
  id: string;
  taskId: string;
  agentId: string;
  agentName: string; // "claude-opus-4", "gpt-4", etc.
  status: "running" | "completed" | "failed";
  createdAt: string;
  startedAt?: string;
  completedAt?: string;
  duration?: string;

  // Token usage
  tokens?: TokenUsage;

  // The actual output
  output: {
    summary: string;
    code?: string;
    files?: { path: string; content: string }[];
    reasoning?: string;
  };

  // Evaluation (by overseer or automated)
  evaluation?: {
    score?: number;
    pros?: string[];
    cons?: string[];
  };

  // Resources produced by this execution
  producedResourceIds?: string[];
}

// Import resource types
import type {
  InputRequirement,
  OutputRequirement,
  ResourceRef,
  ResourceCompletionStatus,
} from "./resource";
import type { TaskVCSContext, ThreadVCSContext } from "./vcs";

// Task is the template - what should happen
export interface Task {
  id: string;
  name: string;
  description: string;
  dependencies: string[]; // other task IDs
  subtasks?: Subtask[];
  agent?: string; // default/preferred agent

  // Execution tracking
  executions?: TaskExecution[];
  selectedExecutionId?: string; // which execution was chosen

  // Derived status from executions
  status: TaskStatus;

  // Legacy fields for compatibility
  duration?: string;
  startedAt?: string;
  completedAt?: string;

  // Resource requirements
  inputRequirements?: InputRequirement[];
  outputRequirements?: OutputRequirement[];

  // Resolved available resources
  availableResources?: ResourceRef[];

  // Completion tracking
  resourceCompletionStatus?: ResourceCompletionStatus;

  // VCS context
  vcs?: TaskVCSContext;

  // Planning phase (optional)
  plannerPrompt?: string;
  planApproval?: "user" | "overseer" | "auto";
  planStatus?: import("./task").TaskPlanStatus;
  plan?: import("./task").TaskPlan;
  planner?: { id: string; name: string };
}

// Message in overseer conversation
export interface OverseerMessage {
  id: string;
  role: "overseer" | "agent" | "user";
  content: string;
  timestamp: string;
  agentId?: string;
  executionId?: string; // which execution this relates to
}

// Who is requesting attention
export type InboxSource =
  | "project_overseer" // Top-level project coordination
  | "thread_overseer" // Thread-level coordination (managing tasks within a thread)
  | "task_agent"; // Agent working on a specific task

// Inbox item types (renamed from FeedbackType)
export type InboxItemType =
  | "execution_selection" // Multiple executions need human selection
  | "thread_review" // Thread needs review/approval
  | "planning_approval" // Planning phase needs approval
  | "task_plan_approval" // Task-level plan needs user approval
  | "questionnaire" // Agent needs answers to questions
  | "decision_escalation" // Agent escalates blocking decision it can't make
  | "direction_clarification"; // Agent needs guidance on approach

// Questionnaire option
export interface QuestionOption {
  id: string;
  label: string;
  description?: string;
}

// Questionnaire question
export interface QuestionnaireQuestion {
  id: string;
  header: string;
  question: string;
  options: QuestionOption[];
  multiSelect: boolean;
  selectedOptionIds?: string[];
}

// Decision option for escalations
export interface DecisionOption {
  id: string;
  label: string;
  description?: string;
  recommended?: boolean;
  risks?: string[];
  benefits?: string[];
}

// Direction clarification context
export interface ClarificationContext {
  currentUnderstanding: string;
  relevantCode?: string;
  relevantFiles?: string[];
  approachOptions?: {
    id: string;
    description: string;
    tradeoffs?: string;
  }[];
}

// Inbox item status (renamed from FeedbackStatus)
export type InboxItemStatus = "pending" | "reviewed" | "resolved";

// Priority levels (renamed from FeedbackPriority)
export type InboxItemPriority = "low" | "medium" | "high";

// Unified inbox item (renamed from FeedbackItem)
export interface InboxItem {
  id: string;
  type: InboxItemType;
  status: InboxItemStatus;
  priority: InboxItemPriority;
  title: string;
  description: string;
  createdAt: string;

  // Who is requesting attention
  source: InboxSource;
  sourceName?: string; // e.g., "Alex Chen" for thread overseer, "Claude Opus" for task agent

  // Context references
  threadId?: string;
  threadName?: string; // e.g., "User Authentication Flow"
  taskId?: string;
  taskName?: string; // e.g., "Build Login UI"

  // For execution selection (multiple agents tackled same task)
  executionIds?: string[]; // references to TaskExecution
  selectedExecutionId?: string;
  decidedBy?: "user" | "overseer";

  // Overseer conversation history
  overseerMessages?: OverseerMessage[];

  // Questionnaire feedback
  questions?: QuestionnaireQuestion[];
  customResponse?: string;

  // Decision escalation fields
  decisionContext?: string; // What led to this decision point
  decisionOptions?: DecisionOption[]; // Options the agent has identified
  selectedDecisionId?: string; // Which decision was made
  decisionRationale?: string; // Why the decision was made

  // Direction clarification fields
  clarificationContext?: ClarificationContext;
  clarificationResponse?: string; // The guidance provided
  delegatedToIssueOverseer?: boolean; // Whether this was delegated up
}

// View mode for inbox
export type ViewMode = "user" | "overseer";

// Thread Overseer - manages a specific thread
export interface ThreadOverseer {
  id: string;
  name: string; // Human name like "Alex Chen"
  threadId: string;
}

// Legacy alias
export type IssueOverseer = ThreadOverseer;

// Task Agent - responsible for a task execution
export interface TaskAgent {
  id: string;
  agentId: string; // e.g., "claude-opus-4", "gpt-4"
  agentName: string; // Display name
  taskId: string;
  executionId?: string;
}

// Chat message for Issue Overseer or Task Agent conversations
export interface AgentChatMessage {
  id: string;
  role: "user" | "agent";
  content: string;
  timestamp: string;
}

// Chat session for Thread Overseer or Task Agent
export interface AgentChatSession {
  id: string;
  type: "thread_overseer" | "task_agent";
  entityId: string; // threadId or taskId
  messages: AgentChatMessage[];
}

// Type aliases for backwards compatibility during migration
export type FeedbackType = InboxItemType;
export type FeedbackStatus = InboxItemStatus;
export type FeedbackPriority = InboxItemPriority;
export type FeedbackItem = InboxItem;

// Legacy alias
export type ThreadId = string;

// Import additional resource types for Thread
import type { ThreadResource } from "./resource";

// Thread status types
export type ThreadStatus = "draft" | "pending" | "in_progress" | "completed" | "failed";

// Thread interface with resource support
export interface Thread {
  id: string;
  title: string;
  description: string;
  status: ThreadStatus;
  taskCount: number;
  completedTasks: number;
  createdAt: string;
  updatedAt: string;
  overseer: ThreadOverseer;
  memory?: string;
  projectId: string;

  // Resources available at thread level
  resources?: ThreadResource[];

  // VCS context at thread level
  vcs?: ThreadVCSContext;

  // Thread mode
  mode: "single_agent" | "orchestrated";
  model?: string;
}

// Legacy alias
export type Issue = Thread;
export type IssueStatus = ThreadStatus;
