import { useMemo, useRef } from "react";
import { Link } from "@tanstack/react-router";
import { useVirtualizer } from "@tanstack/react-virtual";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Inbox,
  GitCompare,
  FileCheck,
  ClipboardList,
  MessageCircleQuestion,
  AlertTriangle,
  Compass,
} from "lucide-react";
import { inboxItems } from "@/data/mockInboxData";
import type { InboxItem, InboxItemType } from "@/types/inbox";

const typeConfig: Record<InboxItemType, { label: string; icon: React.ElementType; color: string }> =
  {
    execution_selection: {
      label: "Code Review",
      icon: GitCompare,
      color: "bg-violet-400/15 text-violet-600 dark:bg-violet-500/15 dark:text-violet-400",
    },
    thread_review: {
      label: "Thread Review",
      icon: FileCheck,
      color: "bg-sky-400/15 text-sky-600 dark:bg-sky-500/15 dark:text-sky-400",
    },
    planning_approval: {
      label: "Planning",
      icon: ClipboardList,
      color: "bg-amber-400/15 text-amber-600 dark:bg-amber-500/15 dark:text-amber-400",
    },
    task_plan_approval: {
      label: "Task Plan",
      icon: ClipboardList,
      color: "bg-orange-400/15 text-orange-600 dark:bg-orange-500/15 dark:text-orange-400",
    },
    questionnaire: {
      label: "Question",
      icon: MessageCircleQuestion,
      color: "bg-emerald-400/15 text-emerald-600 dark:bg-emerald-500/15 dark:text-emerald-400",
    },
    decision_escalation: {
      label: "Decision",
      icon: AlertTriangle,
      color: "bg-rose-400/15 text-rose-600 dark:bg-rose-500/15 dark:text-rose-400",
    },
    direction_clarification: {
      label: "Clarification",
      icon: Compass,
      color: "bg-cyan-400/15 text-cyan-600 dark:bg-cyan-500/15 dark:text-cyan-400",
    },
  };

interface FeedbackListProps {
  selectedThreadId: string | null;
  selectedTaskId: string | null;
}

export function FeedbackList({ selectedThreadId, selectedTaskId }: FeedbackListProps) {
  const scrollRef = useRef<HTMLDivElement>(null);

  // Filter to pending items only
  const pendingItems = useMemo(() => {
    return inboxItems.filter((item) => item.status === "pending");
  }, []);

  const virtualizer = useVirtualizer({
    count: pendingItems.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => 56,
    overscan: 5,
    getItemKey: (index) => pendingItems[index].id,
  });

  if (pendingItems.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 px-4 text-center">
        <Inbox className="mb-2 size-5 text-muted-foreground" aria-hidden="true" />
        <p className="text-xs text-muted-foreground">No pending feedback</p>
      </div>
    );
  }

  return (
    <ScrollArea
      className="flex-1 overflow-hidden"
      viewportRef={scrollRef}
      viewportClassName="!overflow-y-auto"
    >
      <div
        className="py-1"
        style={{
          height: `${virtualizer.getTotalSize()}px`,
          width: "100%",
          position: "relative",
        }}
      >
        {virtualizer.getVirtualItems().map((virtualRow) => {
          const item = pendingItems[virtualRow.index];
          return (
            <div
              key={virtualRow.key}
              data-index={virtualRow.index}
              ref={virtualizer.measureElement}
              className="py-0.5"
              style={{
                position: "absolute",
                top: 0,
                left: 0,
                width: "100%",
                transform: `translateY(${virtualRow.start}px)`,
              }}
            >
              <FeedbackListItem
                item={item}
                index={virtualRow.index}
                isSelected={
                  (item.threadId === selectedThreadId && !item.taskId) ||
                  item.taskId === selectedTaskId
                }
              />
            </div>
          );
        })}
      </div>
    </ScrollArea>
  );
}

function FeedbackListItem({
  item,
  index,
  isSelected,
}: {
  item: InboxItem;
  index: number;
  isSelected: boolean;
}) {
  const linkTo = item.taskId
    ? `/app/tasks/${item.threadId}/${item.taskId}`
    : `/app/threads/${item.threadId}`;

  const primaryName = item.taskName || item.threadName || item.title;
  const secondaryName = item.taskName ? item.threadName : null;
  const isOdd = index % 2 === 1;
  const config = typeConfig[item.type];
  const Icon = config.icon;

  return (
    <Link
      to={linkTo}
      search={{ feedback: item.id }}
      className={cn(
        "mx-2 block rounded-lg px-3 py-2 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
        isOdd && "bg-muted/40",
        isSelected && "bg-muted",
        !isSelected && "hover:bg-muted/50",
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <Badge
          variant="secondary"
          className={cn("h-4 gap-1 px-1.5 text-[10px] font-medium border-0", config.color)}
        >
          <Icon className="size-2.5" aria-hidden="true" />
          {config.label}
        </Badge>
        <span className="text-[11px] tabular-nums text-muted-foreground">{item.createdAt}</span>
      </div>
      <div className="mt-1 truncate text-[13px] font-medium">{primaryName}</div>
      {secondaryName && (
        <div className="truncate text-[11px] text-muted-foreground">{secondaryName}</div>
      )}
    </Link>
  );
}
