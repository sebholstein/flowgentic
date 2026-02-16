import { memo } from "react";
import { Link } from "@tanstack/react-router";
import { cn } from "@/lib/utils";
import {
  ChevronRight,
  ChevronDown,
  Pin,
  Archive,
  MessagesSquare,
} from "lucide-react";
import { useTypewriter } from "@/hooks/use-typewriter";
import type { ThreadConfig } from "@/proto/gen/controlplane/v1/thread_service_pb";

export const ThreadRow = memo(function ThreadRow({
  thread,
  isSelected,
  isExpanded,
  hasChildren,
  isPinned,
  isArchived,
  isDemo,
  onToggle,
  onTogglePin,
  onToggleArchive,
}: {
  thread: ThreadConfig;
  isSelected: boolean;
  isExpanded: boolean;
  hasChildren: boolean;
  isPinned: boolean;
  isArchived: boolean;
  isDemo?: boolean;
  onToggle: () => void;
  onTogglePin: () => void;
  onToggleArchive: () => void;
}) {
  const animatedTopic = useTypewriter(thread.topic || "Untitled");

  const linkProps = isDemo
    ? { to: "/app/demo/$scenarioId" as const, params: { scenarioId: thread.id } }
    : { to: "/app/threads/$threadId" as const, params: { threadId: thread.id } };

  return (
    <div
      className={cn(
        "group flex items-center gap-1 pr-2 rounded-md transition-colors",
        isSelected ? "bg-muted text-foreground" : "text-muted-foreground hover:bg-muted/50",
      )}
      style={{ paddingLeft: "4px" }}
    >
      {hasChildren ? (
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            onToggle();
          }}
          className="size-4 flex items-center justify-center shrink-0 text-muted-foreground hover:text-foreground"
        >
          {isExpanded ? <ChevronDown className="size-3" /> : <ChevronRight className="size-3" />}
        </button>
      ) : (
        <span className="size-4 shrink-0" />
      )}
      <Link
        {...linkProps}
        className={cn(
          "flex flex-1 items-center gap-1.5 px-1.5 py-1.5 text-sm transition-colors text-left min-w-0 select-none",
          isArchived && "opacity-60",
        )}
      >
        <MessagesSquare className="size-3 shrink-0 text-muted-foreground" />
        <span className="truncate flex-1 text-[13px]">{animatedTopic}</span>
      </Link>
      <div
        className={cn(
          "items-center gap-1 hidden",
          "group-hover:flex",
          (isPinned || isArchived) && "flex",
        )}
      >
        <button
          type="button"
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            onTogglePin();
          }}
          className={cn(
            "rounded p-1 transition-colors",
            isArchived && "opacity-40 cursor-not-allowed",
            !isArchived &&
              (isPinned
                ? "text-amber-400 hover:text-amber-300"
                : "text-muted-foreground hover:text-foreground"),
          )}
          aria-label={isPinned ? "Unpin thread" : "Pin thread"}
          title={isPinned ? "Unpin thread" : "Pin thread"}
          disabled={isArchived}
        >
          <Pin className={cn("size-3.5", isPinned && "-rotate-45")} />
        </button>
        <button
          type="button"
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            onToggleArchive();
          }}
          className={cn(
            "rounded p-1 transition-colors",
            isArchived
              ? "text-muted-foreground hover:text-foreground"
              : "text-muted-foreground hover:text-foreground",
          )}
          aria-label={isArchived ? "Unarchive thread" : "Archive thread"}
          title={isArchived ? "Unarchive thread" : "Archive thread"}
        >
          <Archive className="size-3.5" />
        </button>
      </div>
    </div>
  );
});
