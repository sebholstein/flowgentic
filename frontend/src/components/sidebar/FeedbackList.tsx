import { Link, useSearch } from "@tanstack/react-router";
import { cn } from "@/lib/utils";
import { ScrollArea } from "@/components/ui/scroll-area";
import { CheckCircle2, AlertCircle, Clock, Check } from "lucide-react";
import { inboxItems } from "@/data/mockInboxData";

const statusIcons = {
  pending: Clock,
  accepted: CheckCircle2,
  rejected: AlertCircle,
  resolved: Check,
};

const statusColors = {
  pending: "text-amber-500",
  accepted: "text-emerald-500",
  rejected: "text-rose-500",
  resolved: "text-emerald-500",
};

export function FeedbackList({
  selectedThreadId,
  selectedTaskId,
}: {
  selectedThreadId: string | null;
  selectedTaskId: string | null;
}) {
  const search = useSearch({ strict: false }) as { feedback?: string };
  const selectedFeedbackId = search.feedback ?? null;

  return (
    <ScrollArea className="flex-1 overflow-hidden px-2 pt-2">
      <div className="space-y-1 p-2">
        {inboxItems.map((item) => {
          const StatusIcon = statusIcons[item.status as keyof typeof statusIcons] ?? Clock;
          const isSelected = selectedFeedbackId === item.id;

          return (
            <Link
              key={item.id}
              to="/app/threads/$threadId"
              params={{ threadId: item.threadId }}
              search={{ feedback: item.id }}
              className={cn(
                "flex items-start gap-2 rounded-md px-2 py-2 text-sm transition-colors",
                isSelected
                  ? "bg-muted text-foreground"
                  : "text-muted-foreground hover:bg-muted/50 hover:text-foreground",
              )}
            >
              <StatusIcon className={cn("size-4 shrink-0 mt-0.5", statusColors[item.status])} />
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-1.5">
                  <span className="font-medium truncate">{item.title}</span>
                </div>
                <div className="text-xs text-muted-foreground truncate">{item.preview}</div>
                <div className="text-xs text-muted-foreground mt-1">
                  {new Date(item.timestamp).toLocaleDateString()}
                </div>
              </div>
            </Link>
          );
        })}
      </div>
    </ScrollArea>
  );
}
