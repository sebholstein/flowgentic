import { useState, useEffect, useMemo, useCallback } from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Tooltip, TooltipContent, TooltipTrigger, TooltipProvider } from "@/components/ui/tooltip";
import {
  Rows3,
  Columns2,
  ChevronDown,
  ChevronUp,
  MessageSquare,
  XCircle,
  ThumbsUp,
  FileCode2,
} from "lucide-react";
import { FileTree } from "./FileTree";
import { DiffViewer } from "./DiffViewer";
import { CommentThread } from "./CommentThread";
import { CommentPopover } from "./CommentPopover";
import {
  useCodeReviewStore,
  selectCommentsForFile,
  selectUnresolvedCommentsCount,
} from "@/stores/codeReviewStore";
import type { ExecutionDiff, LineComment, CommentAuthor } from "@/types/code-review";

interface CodeReviewViewProps {
  execution: ExecutionDiff;
  comments?: LineComment[];
  onApprove?: () => void;
  onRequestChanges?: () => void;
  onDismiss?: () => void;
  className?: string;
}

// Default mock user for comments
const defaultUser: CommentAuthor = {
  type: "user",
  id: "current-user",
  name: "You",
};

export function CodeReviewView({
  execution,
  comments: initialComments = [],
  onApprove,
  onRequestChanges,
  onDismiss,
  className,
}: CodeReviewViewProps) {
  const {
    selectedFilePath,
    viewMode,
    comments,
    pendingComment,
    collapsedRegions,
    setExecution,
    selectFile,
    setViewMode,
    setComments,
    setCollapsedRegions,
    addComment,
    resolveComment,
    unresolveComment,
    replyToComment,
    startComment,
    cancelComment,
  } = useCodeReviewStore();

  // Sidebar resize state (using the custom pattern per CLAUDE.md)
  const [sidebarWidth, setSidebarWidth] = useState(280);

  // Initialize store with execution data
  useEffect(() => {
    setExecution(execution.executionId, execution.files);
    setComments(initialComments);
  }, [execution, initialComments, setExecution, setComments]);

  // Get the selected file
  const selectedFile = useMemo(
    () => execution.files.find((f) => f.path === selectedFilePath),
    [execution.files, selectedFilePath],
  );

  // Get comments for the selected file
  const fileComments = useMemo(
    () =>
      selectedFilePath
        ? selectCommentsForFile(
            {
              comments,
              files: [],
              selectedFilePath,
              viewMode,
              executionId: execution.executionId,
              pendingComment,
              collapsedRegions,
            },
            selectedFilePath,
          )
        : [],
    [comments, selectedFilePath, viewMode, execution.executionId, pendingComment, collapsedRegions],
  );

  const unresolvedCount = selectUnresolvedCommentsCount({
    comments,
    files: [],
    selectedFilePath,
    viewMode,
    executionId: execution.executionId,
    pendingComment,
    collapsedRegions,
  });

  // Sidebar resize handler
  const handleMouseDown = (e: React.MouseEvent) => {
    e.preventDefault();
    const startX = e.clientX;
    const startWidth = sidebarWidth;

    const handleMouseMove = (moveEvent: MouseEvent) => {
      const newWidth = startWidth + (moveEvent.clientX - startX);
      setSidebarWidth(Math.min(400, Math.max(200, newWidth)));
    };

    const handleMouseUp = () => {
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
    };

    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);
  };

  const handleAddComment = useCallback(
    (data: { content: string; actionType?: LineComment["actionType"]; author: CommentAuthor }) => {
      if (!pendingComment) return;
      addComment({
        executionId: execution.executionId,
        filePath: pendingComment.filePath,
        lineNumber: pendingComment.lineNumber,
        author: data.author,
        content: data.content,
        actionType: data.actionType,
        resolved: false,
      });
    },
    [addComment, execution.executionId, pendingComment],
  );

  const handleReply = useCallback(
    (parentId: string, content: string) => {
      replyToComment(parentId, content, defaultUser);
    },
    [replyToComment],
  );

  return (
    <div className={cn("flex flex-col h-full", className)}>
      {/* Header */}
      <div className="flex items-center justify-between gap-4 px-4 py-3 border-b bg-muted/30">
        <div className="flex items-center gap-3 min-w-0">
          <div className="flex items-center gap-2">
            <FileCode2 className="size-4 text-muted-foreground" />
            <span className="font-medium text-sm">{execution.agentName}</span>
          </div>
          <div className="flex items-center gap-2 text-sm">
            <span className="text-emerald-400 tabular-nums">+{execution.totalAdditions}</span>
            <span className="text-red-400 tabular-nums">-{execution.totalDeletions}</span>
            <span className="text-muted-foreground">
              ({execution.files.length} {execution.files.length === 1 ? "file" : "files"})
            </span>
          </div>
          {unresolvedCount > 0 && (
            <Badge variant="outline" className="gap-1 text-purple-400 border-purple-500/30">
              <MessageSquare className="size-3" />
              {unresolvedCount} unresolved
            </Badge>
          )}
        </div>

        <div className="flex items-center gap-2">
          {/* View mode toggle */}
          <TooltipProvider>
            <div className="flex items-center border rounded-md">
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="sm"
                    className={cn("h-8 px-2 rounded-r-none", viewMode === "unified" && "bg-muted")}
                    onClick={() => setViewMode("unified")}
                  >
                    <Rows3 className="size-4" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Unified view</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="sm"
                    className={cn(
                      "h-8 px-2 rounded-l-none border-l",
                      viewMode === "split" && "bg-muted",
                    )}
                    onClick={() => setViewMode("split")}
                  >
                    <Columns2 className="size-4" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Side-by-side view</TooltipContent>
              </Tooltip>
            </div>
          </TooltipProvider>

          {/* Collapse toggle */}
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="outline"
                  size="sm"
                  className="h-8 gap-1.5"
                  onClick={() => setCollapsedRegions(!collapsedRegions)}
                >
                  {collapsedRegions ? (
                    <>
                      <ChevronDown className="size-3.5" />
                      Expand
                    </>
                  ) : (
                    <>
                      <ChevronUp className="size-3.5" />
                      Collapse
                    </>
                  )}
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                {collapsedRegions ? "Expand unchanged regions" : "Collapse unchanged regions"}
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </div>
      </div>

      {/* Main content */}
      <div className="flex flex-1 min-h-0">
        {/* File tree sidebar */}
        <div className="flex-shrink-0 border-r overflow-hidden" style={{ width: sidebarWidth }}>
          <FileTree
            files={execution.files}
            selectedPath={selectedFilePath}
            onSelectFile={selectFile}
            comments={comments}
          />
        </div>

        {/* Resize handle - wide hit area, thin visual line */}
        <div
          className="w-3 -ml-[6px] -mr-[5px] flex-shrink-0 cursor-col-resize flex justify-center group relative z-10"
          onMouseDown={handleMouseDown}
        >
          <div className="w-px h-full bg-border group-hover:bg-primary/30 transition-colors pointer-events-none" />
        </div>

        {/* Diff viewer and comments */}
        <div className="flex-1 min-w-0 flex flex-col">
          {selectedFile ? (
            <>
              {/* File header */}
              <div className="flex items-center justify-between px-4 py-2 border-b bg-muted/20">
                <div className="flex items-center gap-2 min-w-0">
                  <span className="font-mono text-sm truncate">{selectedFile.path}</span>
                  <Badge
                    variant="outline"
                    className={cn(
                      "text-xs",
                      selectedFile.status === "added" && "text-emerald-400 border-emerald-500/30",
                      selectedFile.status === "modified" && "text-amber-400 border-amber-500/30",
                      selectedFile.status === "deleted" && "text-red-400 border-red-500/30",
                      selectedFile.status === "renamed" && "text-blue-400 border-blue-500/30",
                    )}
                  >
                    {selectedFile.status}
                  </Badge>
                </div>
                <div className="text-sm text-muted-foreground tabular-nums">
                  <span className="text-emerald-400">+{selectedFile.additions}</span>
                  {" / "}
                  <span className="text-red-400">-{selectedFile.deletions}</span>
                </div>
              </div>

              {/* Diff view */}
              <div className="flex-1 min-h-0 relative">
                <DiffViewer
                  file={selectedFile}
                  viewMode={viewMode}
                  comments={fileComments}
                  onLineClick={(line) => startComment(selectedFile.path, line)}
                />
              </div>

              {/* Comments panel */}
              {fileComments.length > 0 && (
                <div className="border-t bg-muted/10">
                  <div className="px-4 py-2 border-b">
                    <span className="text-sm font-medium">Comments ({fileComments.length})</span>
                  </div>
                  <ScrollArea className="max-h-[300px]">
                    <div className="p-4 space-y-3">
                      {fileComments.map((comment) => (
                        <CommentThread
                          key={comment.id}
                          comment={comment}
                          onResolve={resolveComment}
                          onUnresolve={unresolveComment}
                          onReply={handleReply}
                          currentUser={defaultUser}
                        />
                      ))}
                    </div>
                  </ScrollArea>
                </div>
              )}
            </>
          ) : (
            <div className="flex-1 flex items-center justify-center text-muted-foreground">
              Select a file to view changes
            </div>
          )}
        </div>
      </div>

      {/* Pending comment popover */}
      {pendingComment && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <CommentPopover
            filePath={pendingComment.filePath}
            lineNumber={pendingComment.lineNumber}
            onSubmit={handleAddComment}
            onCancel={cancelComment}
            currentUser={defaultUser}
          />
        </div>
      )}

      {/* Footer actions */}
      {(onApprove || onRequestChanges || onDismiss) && (
        <div className="flex items-center justify-end gap-3 px-4 py-3 border-t bg-muted/30">
          {onDismiss && (
            <Button variant="ghost" onClick={onDismiss}>
              Dismiss
            </Button>
          )}
          {onRequestChanges && (
            <Button variant="outline" className="gap-1.5" onClick={onRequestChanges}>
              <XCircle className="size-4" />
              Request Changes
            </Button>
          )}
          {onApprove && (
            <Button className="gap-1.5 bg-emerald-600 hover:bg-emerald-700" onClick={onApprove}>
              <ThumbsUp className="size-4" />
              Approve & Select
            </Button>
          )}
        </div>
      )}
    </div>
  );
}
