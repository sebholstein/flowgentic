import { useState } from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  CheckCircle2,
  CircleDot,
  MessageSquare,
  HelpCircle,
  Lightbulb,
  AlertCircle,
  ThumbsUp,
  CornerDownRight,
  X,
} from "lucide-react";
import type { LineComment, CommentActionType, CommentAuthor } from "@/types/code-review";

interface CommentThreadProps {
  comment: LineComment;
  onResolve?: (commentId: string) => void;
  onUnresolve?: (commentId: string) => void;
  onReply?: (parentId: string, content: string) => void;
  onDelete?: (commentId: string) => void;
  currentUser?: CommentAuthor;
  className?: string;
}

function getActionIcon(actionType?: CommentActionType) {
  switch (actionType) {
    case "question":
      return HelpCircle;
    case "suggestion":
      return Lightbulb;
    case "request_change":
      return AlertCircle;
    case "approve":
      return ThumbsUp;
    default:
      return MessageSquare;
  }
}

function getActionLabel(actionType?: CommentActionType) {
  switch (actionType) {
    case "question":
      return "Question";
    case "suggestion":
      return "Suggestion";
    case "request_change":
      return "Change requested";
    case "approve":
      return "Approved";
    default:
      return null;
  }
}

function getActionColor(actionType?: CommentActionType) {
  switch (actionType) {
    case "question":
      return "text-blue-400";
    case "suggestion":
      return "text-amber-400";
    case "request_change":
      return "text-red-400";
    case "approve":
      return "text-emerald-400";
    default:
      return "text-muted-foreground";
  }
}

function getAuthorColor(type: CommentAuthor["type"]) {
  switch (type) {
    case "user":
      return "bg-blue-500";
    case "overseer":
      return "bg-purple-500";
    case "agent":
      return "bg-orange-500";
    default:
      return "bg-slate-500";
  }
}

function getInitials(name: string) {
  return name
    .split(" ")
    .map((n) => n[0])
    .join("")
    .toUpperCase()
    .slice(0, 2);
}

function formatDate(dateString: string) {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return "just now";
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;

  return date.toLocaleDateString();
}

function SingleComment({
  comment,
  isReply = false,
  onDelete: _onDelete,
}: {
  comment: LineComment;
  isReply?: boolean;
  onDelete?: (commentId: string) => void;
}) {
  const ActionIcon = getActionIcon(comment.actionType);
  const actionLabel = getActionLabel(comment.actionType);
  const actionColor = getActionColor(comment.actionType);

  return (
    <div className={cn("flex gap-2", isReply && "ml-6")}>
      {isReply && <CornerDownRight className="size-3.5 text-muted-foreground mt-1 shrink-0" />}
      <Avatar className="size-6 shrink-0">
        <AvatarFallback className={cn("text-xs text-white", getAuthorColor(comment.author.type))}>
          {getInitials(comment.author.name)}
        </AvatarFallback>
      </Avatar>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="font-medium text-sm">{comment.author.name}</span>
          {comment.author.type !== "user" && (
            <span className="text-xs text-muted-foreground capitalize">
              ({comment.author.type})
            </span>
          )}
          <span className="text-xs text-muted-foreground">{formatDate(comment.createdAt)}</span>
          {actionLabel && (
            <span className={cn("text-xs flex items-center gap-1", actionColor)}>
              <ActionIcon className="size-3" />
              {actionLabel}
            </span>
          )}
        </div>
        <p className="text-sm text-foreground/90 mt-1 whitespace-pre-wrap">{comment.content}</p>
      </div>
    </div>
  );
}

export function CommentThread({
  comment,
  onResolve,
  onUnresolve,
  onReply,
  onDelete,
  currentUser: _currentUser,
  className,
}: CommentThreadProps) {
  const [isReplying, setIsReplying] = useState(false);
  const [replyContent, setReplyContent] = useState("");

  const handleSubmitReply = () => {
    if (!replyContent.trim() || !onReply) return;
    onReply(comment.id, replyContent.trim());
    setReplyContent("");
    setIsReplying(false);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      handleSubmitReply();
    }
    if (e.key === "Escape") {
      setIsReplying(false);
      setReplyContent("");
    }
  };

  return (
    <div
      className={cn(
        "rounded-lg border p-3 space-y-3",
        comment.resolved ? "bg-muted/30 border-muted" : "bg-background border-border",
        className,
      )}
    >
      {/* Main comment */}
      <SingleComment comment={comment} onDelete={onDelete} />

      {/* Replies */}
      {comment.replies && comment.replies.length > 0 && (
        <div className="space-y-3 pt-2 border-t border-dashed">
          {comment.replies.map((reply) => (
            <SingleComment key={reply.id} comment={reply} isReply onDelete={onDelete} />
          ))}
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center gap-2 pt-2 border-t">
        {!isReplying && (
          <>
            <Button
              variant="ghost"
              size="sm"
              className="h-7 text-xs gap-1.5"
              onClick={() => setIsReplying(true)}
            >
              <MessageSquare className="size-3.5" />
              Reply
            </Button>
            {comment.resolved ? (
              <Button
                variant="ghost"
                size="sm"
                className="h-7 text-xs gap-1.5 text-muted-foreground"
                onClick={() => onUnresolve?.(comment.id)}
              >
                <CircleDot className="size-3.5" />
                Unresolve
              </Button>
            ) : (
              <Button
                variant="ghost"
                size="sm"
                className="h-7 text-xs gap-1.5 text-emerald-400 hover:text-emerald-300"
                onClick={() => onResolve?.(comment.id)}
              >
                <CheckCircle2 className="size-3.5" />
                Resolve
              </Button>
            )}
          </>
        )}
      </div>

      {/* Reply input */}
      {isReplying && (
        <div className="space-y-2">
          <textarea
            autoFocus
            value={replyContent}
            onChange={(e) => setReplyContent(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Write a reply..."
            className="w-full min-h-[60px] p-2 text-sm rounded-md border bg-background resize-none focus:outline-none focus:ring-2 focus:ring-primary/50"
          />
          <div className="flex items-center justify-between">
            <span className="text-xs text-muted-foreground">Press ⌘+Enter to submit</span>
            <div className="flex gap-2">
              <Button
                variant="ghost"
                size="sm"
                className="h-7"
                onClick={() => {
                  setIsReplying(false);
                  setReplyContent("");
                }}
              >
                Cancel
              </Button>
              <Button
                size="sm"
                className="h-7"
                onClick={handleSubmitReply}
                disabled={!replyContent.trim()}
              >
                Reply
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// Inline comment display for showing in the diff view
export function InlineCommentThread({
  comment,
  onResolve,
  onUnresolve,
  onReply,
  onCollapse,
  className,
}: CommentThreadProps & { onCollapse?: () => void }) {
  return (
    <div
      className={cn("border-l-2 border-purple-500/50 bg-purple-500/5 ml-12 mr-4 my-1", className)}
    >
      <div className="flex items-center justify-between px-3 py-1 border-b border-purple-500/20">
        <span className="text-xs text-muted-foreground">Line {comment.lineNumber}</span>
        {onCollapse && (
          <Button variant="ghost" size="sm" className="h-5 w-5 p-0" onClick={onCollapse}>
            <X className="size-3" />
          </Button>
        )}
      </div>
      <div className="p-3">
        <CommentThread
          comment={comment}
          onResolve={onResolve}
          onUnresolve={onUnresolve}
          onReply={onReply}
          className="border-0 bg-transparent p-0"
        />
      </div>
    </div>
  );
}
