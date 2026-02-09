import { cn } from "@/lib/utils";
import { MessageSquare } from "lucide-react";
import type { CommentActionType } from "@/types/code-review";

interface CommentBadgeProps {
  count: number;
  hasUnresolved?: boolean;
  actionType?: CommentActionType;
  onClick?: () => void;
  className?: string;
}

export function CommentBadge({
  count,
  hasUnresolved = false,
  actionType,
  onClick,
  className,
}: CommentBadgeProps) {
  const getActionColor = () => {
    switch (actionType) {
      case "request_change":
        return "bg-red-500/20 text-red-400 border-red-500/30";
      case "suggestion":
        return "bg-amber-500/20 text-amber-400 border-amber-500/30";
      case "question":
        return "bg-blue-500/20 text-blue-400 border-blue-500/30";
      case "approve":
        return "bg-emerald-500/20 text-emerald-400 border-emerald-500/30";
      default:
        return hasUnresolved
          ? "bg-purple-500/20 text-purple-400 border-purple-500/30"
          : "bg-muted text-muted-foreground border-border";
    }
  };

  if (count === 0) return null;

  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "inline-flex items-center gap-1 px-1.5 py-0.5 rounded border text-xs font-medium transition-colors hover:opacity-80",
        getActionColor(),
        onClick && "cursor-pointer",
        className,
      )}
    >
      <MessageSquare className="size-3" />
      {count > 1 && <span>{count}</span>}
    </button>
  );
}

// Gutter indicator for comment
export function CommentGutterIndicator({
  count,
  hasUnresolved = false,
  onClick,
}: {
  count: number;
  hasUnresolved?: boolean;
  onClick?: () => void;
}) {
  if (count === 0) return null;

  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "flex items-center justify-center size-5 rounded text-xs font-medium transition-colors",
        hasUnresolved
          ? "bg-purple-500/20 text-purple-400 hover:bg-purple-500/30"
          : "bg-muted text-muted-foreground hover:bg-muted/80",
      )}
      title={`${count} comment${count > 1 ? "s" : ""}`}
    >
      {count > 9 ? "9+" : count}
    </button>
  );
}
