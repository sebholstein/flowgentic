import { useState, useEffect, useRef } from "react";
import { Link, useSearch } from "@tanstack/react-router";
import { cn } from "@/lib/utils";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Clock,
  Code,
  Eye,
  Map,
  HelpCircle,
  Scale,
  Compass,
  type LucideIcon,
} from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { inboxItems as allInboxItems } from "@/data/mockInboxData";
import type { InboxItem } from "@/types/inbox";

const INITIAL_COUNT = 4;
/** Random interval between 3–6 seconds */
function randomInterval() {
  return 3000 + Math.random() * 3000;
}

const typeIcons: Record<string, LucideIcon> = {
  execution_selection: Code,
  thread_review: Eye,
  planning_approval: Map,
  task_plan_approval: Map,
  questionnaire: HelpCircle,
  decision_escalation: Scale,
  direction_clarification: Compass,
};

const typeLabels: Record<string, string> = {
  execution_selection: "Code Review",
  thread_review: "Thread Review",
  planning_approval: "Plan Approval",
  task_plan_approval: "Plan Approval",
  questionnaire: "Question",
  decision_escalation: "Decision",
  direction_clarification: "Clarification",
};

const typeColors: Record<string, string> = {
  execution_selection: "bg-blue-400/15 text-blue-500",
  thread_review: "bg-purple-400/15 text-purple-500",
  planning_approval: "bg-orange-400/15 text-orange-500",
  task_plan_approval: "bg-orange-400/15 text-orange-500",
  questionnaire: "bg-cyan-400/15 text-cyan-500",
  decision_escalation: "bg-rose-400/15 text-rose-500",
  direction_clarification: "bg-emerald-400/15 text-emerald-500",
};


export function useDemoInboxItems() {
  const [items, setItems] = useState<InboxItem[]>(() =>
    allInboxItems.slice(0, INITIAL_COUNT),
  );
  const counterRef = useRef(INITIAL_COUNT);

  useEffect(() => {
    const timer = setTimeout(() => {
      setItems((prev) => {
        const idx = counterRef.current % allInboxItems.length;
        counterRef.current += 1;
        const source = allInboxItems[idx];
        const newItem: InboxItem = { ...source, id: `demo-${counterRef.current}` };
        return [newItem, ...prev];
      });
    }, randomInterval());

    return () => clearTimeout(timer);
  }, [items.length]);

  return items;
}

export function FeedbackList({
  items,
  selectedThreadId,
  selectedTaskId,
}: {
  items: InboxItem[];
  selectedThreadId: string | null;
  selectedTaskId: string | null;
}) {
  const search = useSearch({ strict: false }) as { feedback?: string };
  const selectedFeedbackId = search.feedback ?? null;

  return (
    <ScrollArea className="flex-1 overflow-hidden">
      <div className="space-y-0.5 p-1">
        <AnimatePresence initial={false}>
          {items.map((item) => {
            const isSelected = selectedFeedbackId === item.id;

            return (
              <motion.div
                key={item.id}
                initial={{ opacity: 0, height: 0, y: -8 }}
                animate={{ opacity: 1, height: "auto", y: 0 }}
                exit={{ opacity: 0, height: 0 }}
                transition={{ duration: 0.3, ease: "easeOut" }}
              >
                <Link
                  to="/app/threads/$threadId"
                  params={{ threadId: item.threadId ?? "1" }}
                  search={{ feedback: item.id }}
                  className={cn(
                    "flex items-start gap-2 rounded-md px-2 py-1.5 text-xs transition-colors",
                    isSelected
                      ? "bg-muted text-foreground"
                      : "text-muted-foreground hover:bg-muted/50 hover:text-foreground",
                  )}
                >
                  {(() => {
                    const TypeIcon = typeIcons[item.type] ?? Clock;
                    const colorClass = typeColors[item.type] ?? "bg-muted text-muted-foreground";
                    return (
                      <span
                        className={cn("inline-flex items-center justify-center size-5 rounded shrink-0 mt-0.5", colorClass)}
                        title={typeLabels[item.type] ?? item.type}
                      >
                        <TypeIcon className="size-3" />
                      </span>
                    );
                  })()}
                  <div className="flex-1 min-w-0 overflow-hidden">
                    <div className="flex items-center gap-1.5">
                      <span className="font-semibold text-xs truncate">{item.title}</span>
                    </div>
                    <span className="text-[0.65rem] text-muted-foreground truncate block">
                      {[item.threadName, item.taskName].filter(Boolean).join(" · ")}
                    </span>
                  </div>
                </Link>
              </motion.div>
            );
          })}
        </AnimatePresence>
      </div>
    </ScrollArea>
  );
}
