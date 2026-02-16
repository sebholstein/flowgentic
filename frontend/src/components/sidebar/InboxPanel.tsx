import { useState, useMemo } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { cn } from "@/lib/utils";
import { ChevronDown, Filter, Check } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { FeedbackList } from "./FeedbackList";
import type { InboxItem } from "@/types/inbox";

export function InboxPanel({
  isCollapsed,
  pendingCount,
  onToggle,
  items,
  selectedThreadId,
  selectedTaskId,
}: {
  isCollapsed: boolean;
  pendingCount: number;
  onToggle: () => void;
  items: InboxItem[];
  selectedThreadId: string | null;
  selectedTaskId: string | null;
}) {
  const [projectFilter, setProjectFilter] = useState<string | null>(null);

  const projectNames = useMemo(() => {
    const names = new Set<string>();
    for (const item of items) {
      if (item.threadName) names.add(item.threadName);
    }
    return Array.from(names).sort();
  }, [items]);

  const filteredItems = useMemo(
    () => projectFilter ? items.filter((i) => i.threadName === projectFilter) : items,
    [items, projectFilter],
  );

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="flex items-center border-t border-border shrink-0">
        <button
          type="button"
          onClick={onToggle}
          className="flex flex-1 items-center gap-2 px-3 py-2 text-xs font-medium text-muted-foreground hover:text-foreground hover:bg-muted/50 transition-colors cursor-pointer"
        >
          <motion.span
            animate={{ rotate: isCollapsed ? 180 : 0 }}
            transition={{ duration: 0.2 }}
            className="inline-flex"
          >
            <ChevronDown className="size-3" />
          </motion.span>
          <span>Inbox</span>
          {pendingCount > 0 && (
            <motion.span
              key={pendingCount}
              initial={{ scale: 1.3 }}
              animate={{ scale: 1 }}
              transition={{ duration: 0.2 }}
              className="text-[0.6rem] px-1.5 py-0.5 rounded-full min-w-[1.25rem] text-center bg-rose-400/20 text-rose-600 dark:bg-rose-500/20 dark:text-rose-400 font-medium"
            >
              {pendingCount}
            </motion.span>
          )}
        </button>
        {!isCollapsed && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                type="button"
                className={cn(
                  "mr-2 p-1 rounded transition-colors",
                  projectFilter
                    ? "text-primary hover:text-primary/80"
                    : "text-muted-foreground hover:text-foreground",
                )}
                title="Filter by project"
              >
                <Filter className="size-3" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="min-w-[160px]">
              <DropdownMenuItem
                className="text-xs gap-2"
                onClick={() => setProjectFilter(null)}
              >
                <Check className={cn("size-3", projectFilter === null ? "opacity-100" : "opacity-0")} />
                All projects
              </DropdownMenuItem>
              {projectNames.map((name) => (
                <DropdownMenuItem
                  key={name}
                  className="text-xs gap-2"
                  onClick={() => setProjectFilter(name)}
                >
                  <Check className={cn("size-3", projectFilter === name ? "opacity-100" : "opacity-0")} />
                  <span className="truncate">{name}</span>
                </DropdownMenuItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>
      <AnimatePresence>
        {!isCollapsed && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.15 }}
            className="flex-1 min-h-0"
          >
            <FeedbackList items={filteredItems} selectedThreadId={selectedThreadId} selectedTaskId={selectedTaskId} />
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
