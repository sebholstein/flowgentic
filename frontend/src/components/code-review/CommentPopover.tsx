import { useState } from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { HelpCircle, Lightbulb, AlertCircle, ThumbsUp, MessageSquare, X } from "lucide-react";
import type { CommentActionType, CommentAuthor } from "@/types/code-review";

interface CommentPopoverProps {
  filePath: string;
  lineNumber: number;
  onSubmit: (data: {
    content: string;
    actionType?: CommentActionType;
    author: CommentAuthor;
  }) => void;
  onCancel: () => void;
  currentUser: CommentAuthor;
  className?: string;
}

const actionTypes: {
  type: CommentActionType;
  icon: typeof MessageSquare;
  label: string;
  color: string;
}[] = [
  {
    type: "question",
    icon: HelpCircle,
    label: "Question",
    color: "text-blue-400 hover:bg-blue-500/20",
  },
  {
    type: "suggestion",
    icon: Lightbulb,
    label: "Suggestion",
    color: "text-amber-400 hover:bg-amber-500/20",
  },
  {
    type: "request_change",
    icon: AlertCircle,
    label: "Request change",
    color: "text-red-400 hover:bg-red-500/20",
  },
  {
    type: "approve",
    icon: ThumbsUp,
    label: "Approve",
    color: "text-emerald-400 hover:bg-emerald-500/20",
  },
];

export function CommentPopover({
  filePath,
  lineNumber,
  onSubmit,
  onCancel,
  currentUser,
  className,
}: CommentPopoverProps) {
  const [content, setContent] = useState("");
  const [selectedAction, setSelectedAction] = useState<CommentActionType | undefined>();

  const handleSubmit = () => {
    if (!content.trim()) return;
    onSubmit({
      content: content.trim(),
      actionType: selectedAction,
      author: currentUser,
    });
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      handleSubmit();
    }
    if (e.key === "Escape") {
      onCancel();
    }
  };

  const fileName = filePath.split("/").pop();

  return (
    <div className={cn("rounded-lg border bg-background shadow-lg w-[400px]", className)}>
      <div className="flex items-center justify-between px-3 py-2 border-b bg-muted/30">
        <div className="flex items-center gap-2 text-sm">
          <MessageSquare className="size-4 text-muted-foreground" />
          <span className="font-medium">Add comment</span>
          <span className="text-muted-foreground">
            {fileName}:{lineNumber}
          </span>
        </div>
        <Button variant="ghost" size="sm" className="h-6 w-6 p-0" onClick={onCancel}>
          <X className="size-4" />
        </Button>
      </div>

      <div className="p-3 space-y-3">
        <textarea
          autoFocus
          value={content}
          onChange={(e) => setContent(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Write a comment..."
          className="w-full min-h-[80px] p-2 text-sm rounded-md border bg-background resize-none focus:outline-none focus:ring-2 focus:ring-primary/50"
        />

        <div className="flex items-center gap-1">
          <span className="text-xs text-muted-foreground mr-2">Type:</span>
          {actionTypes.map(({ type, icon: Icon, label, color }) => (
            <Button
              key={type}
              variant="ghost"
              size="sm"
              className={cn(
                "h-7 gap-1 text-xs",
                selectedAction === type ? cn(color, "bg-muted") : "text-muted-foreground",
              )}
              onClick={() => setSelectedAction(selectedAction === type ? undefined : type)}
            >
              <Icon className="size-3.5" />
              {label}
            </Button>
          ))}
        </div>

        <div className="flex items-center justify-between pt-2 border-t">
          <span className="text-xs text-muted-foreground">Press ⌘+Enter to submit</span>
          <div className="flex gap-2">
            <Button variant="ghost" size="sm" onClick={onCancel}>
              Cancel
            </Button>
            <Button size="sm" onClick={handleSubmit} disabled={!content.trim()}>
              Add Comment
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}

// Simple inline comment form (for use within the diff view)
export function InlineCommentForm({
  onSubmit,
  onCancel,
  placeholder = "Add a comment...",
  className,
}: {
  onSubmit: (content: string) => void;
  onCancel: () => void;
  placeholder?: string;
  className?: string;
}) {
  const [content, setContent] = useState("");

  const handleSubmit = () => {
    if (!content.trim()) return;
    onSubmit(content.trim());
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      handleSubmit();
    }
    if (e.key === "Escape") {
      onCancel();
    }
  };

  return (
    <div className={cn("border rounded-md bg-background p-2 space-y-2", className)}>
      <textarea
        autoFocus
        value={content}
        onChange={(e) => setContent(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder={placeholder}
        className="w-full min-h-[60px] p-2 text-sm rounded border bg-muted/30 resize-none focus:outline-none focus:ring-1 focus:ring-primary/50"
      />
      <div className="flex justify-end gap-2">
        <Button variant="ghost" size="sm" className="h-7" onClick={onCancel}>
          Cancel
        </Button>
        <Button size="sm" className="h-7" onClick={handleSubmit} disabled={!content.trim()}>
          Comment
        </Button>
      </div>
    </div>
  );
}
