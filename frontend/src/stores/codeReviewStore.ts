import { create } from "zustand";
import type {
  FileDiff,
  LineComment,
  DiffViewMode,
  PendingComment,
  CommentAuthor,
} from "@/types/code-review";

interface CodeReviewState {
  // Current review state
  executionId: string | null;
  files: FileDiff[];
  selectedFilePath: string | null;
  viewMode: DiffViewMode;

  // Comments
  comments: LineComment[];
  pendingComment: PendingComment | null;

  // Collapsed unchanged regions
  collapsedRegions: boolean;
}

interface CodeReviewActions {
  // Setup
  setExecution: (executionId: string, files: FileDiff[]) => void;
  clearReview: () => void;

  // File selection
  selectFile: (path: string | null) => void;

  // View mode
  setViewMode: (mode: DiffViewMode) => void;
  toggleViewMode: () => void;

  // Collapsed regions
  setCollapsedRegions: (collapsed: boolean) => void;

  // Comment management
  setComments: (comments: LineComment[]) => void;
  addComment: (comment: Omit<LineComment, "id" | "createdAt">) => void;
  resolveComment: (commentId: string) => void;
  unresolveComment: (commentId: string) => void;
  deleteComment: (commentId: string) => void;
  replyToComment: (parentId: string, content: string, author: CommentAuthor) => void;

  // Pending comment (when user clicks to add a new comment)
  startComment: (filePath: string, lineNumber: number) => void;
  cancelComment: () => void;
}

type CodeReviewStore = CodeReviewState & CodeReviewActions;

const initialState: CodeReviewState = {
  executionId: null,
  files: [],
  selectedFilePath: null,
  viewMode: "unified",
  comments: [],
  pendingComment: null,
  collapsedRegions: true,
};

export const useCodeReviewStore = create<CodeReviewStore>((set, get) => ({
  ...initialState,

  setExecution: (executionId, files) => {
    set({
      executionId,
      files,
      selectedFilePath: files.length > 0 ? files[0].path : null,
    });
  },

  clearReview: () => {
    set(initialState);
  },

  selectFile: (path) => {
    set({ selectedFilePath: path, pendingComment: null });
  },

  setViewMode: (mode) => {
    set({ viewMode: mode });
  },

  toggleViewMode: () => {
    set((state) => ({
      viewMode: state.viewMode === "unified" ? "split" : "unified",
    }));
  },

  setCollapsedRegions: (collapsed) => {
    set({ collapsedRegions: collapsed });
  },

  setComments: (comments) => {
    set({ comments });
  },

  addComment: (commentData) => {
    const newComment: LineComment = {
      ...commentData,
      id: `comment-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
      createdAt: new Date().toISOString(),
    };

    set((state) => ({
      comments: [...state.comments, newComment],
      pendingComment: null,
    }));
  },

  resolveComment: (commentId) => {
    set((state) => ({
      comments: state.comments.map((c) => (c.id === commentId ? { ...c, resolved: true } : c)),
    }));
  },

  unresolveComment: (commentId) => {
    set((state) => ({
      comments: state.comments.map((c) => (c.id === commentId ? { ...c, resolved: false } : c)),
    }));
  },

  deleteComment: (commentId) => {
    set((state) => ({
      comments: state.comments.filter((c) => c.id !== commentId && c.parentId !== commentId),
    }));
  },

  replyToComment: (parentId, content, author) => {
    const state = get();
    const parentComment = state.comments.find((c) => c.id === parentId);
    if (!parentComment) return;

    const reply: LineComment = {
      id: `reply-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
      executionId: parentComment.executionId,
      filePath: parentComment.filePath,
      lineNumber: parentComment.lineNumber,
      parentId,
      author,
      content,
      createdAt: new Date().toISOString(),
      resolved: false,
    };

    set((state) => ({
      comments: state.comments.map((c) =>
        c.id === parentId ? { ...c, replies: [...(c.replies || []), reply] } : c,
      ),
    }));
  },

  startComment: (filePath, lineNumber) => {
    set({ pendingComment: { filePath, lineNumber } });
  },

  cancelComment: () => {
    set({ pendingComment: null });
  },
}));

// Selectors for computed values
export const selectCommentsForFile = (state: CodeReviewState, filePath: string) =>
  state.comments.filter((c) => c.filePath === filePath && !c.parentId);

export const selectUnresolvedCommentsCount = (state: CodeReviewState) =>
  state.comments.filter((c) => !c.resolved && !c.parentId).length;

export const selectCommentsCountForLine = (
  state: CodeReviewState,
  filePath: string,
  lineNumber: number,
) =>
  state.comments.filter(
    (c) => c.filePath === filePath && c.lineNumber === lineNumber && !c.parentId,
  ).length;
