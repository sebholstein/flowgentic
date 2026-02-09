// Types for the code review interface

export type FileChangeStatus = "added" | "modified" | "deleted" | "renamed";

export interface FileDiff {
  path: string;
  status: FileChangeStatus;
  oldPath?: string; // For renamed files
  language: string; // For syntax highlighting
  oldContent: string;
  newContent: string;
  additions: number;
  deletions: number;
}

export interface ExecutionDiff {
  executionId: string;
  taskId: string;
  agentName: string;
  agentId: string;
  files: FileDiff[];
  totalAdditions: number;
  totalDeletions: number;
  timestamp: string;
}

export type CommentAuthorType = "user" | "overseer" | "agent";

export interface CommentAuthor {
  type: CommentAuthorType;
  id: string;
  name: string;
  avatar?: string;
}

export type CommentActionType = "question" | "suggestion" | "request_change" | "approve";

export interface LineComment {
  id: string;
  executionId: string;
  filePath: string;
  lineNumber: number;
  lineRange?: { start: number; end: number };

  author: CommentAuthor;
  content: string;
  createdAt: string;

  // Threading
  parentId?: string;
  replies?: LineComment[];

  // Actions
  resolved: boolean;
  actionType?: CommentActionType;
}

export type DiffViewMode = "unified" | "split";

export interface PendingComment {
  filePath: string;
  lineNumber: number;
}

export interface CodeReviewState {
  executionId: string | null;
  files: FileDiff[];
  selectedFilePath: string | null;
  viewMode: DiffViewMode;
  comments: LineComment[];
  pendingComment: PendingComment | null;
}

// For comparing two executions
export interface ExecutionComparisonState {
  leftExecutionId: string | null;
  rightExecutionId: string | null;
  selectedFilePath: string | null;
}
