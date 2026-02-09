"use client";

import * as React from "react";
import { useNavigate } from "@tanstack/react-router";
import {
  Command,
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from "@/components/ui/command";
import {
  Inbox,
  GitBranch,
  Bot,
  Settings,
  GitCompare,
  FileCheck,
  ClipboardCheck,
  HelpCircle,
} from "lucide-react";
import { cn } from "@/lib/utils";

// Import threads data
import { threads } from "@/data/mockThreadsData";
import { threadStatusConfig } from "@/constants/threadStatusConfig";

// Import inbox data
import { inboxItems } from "@/data/mockInboxData";
import type { InboxItemType } from "@/types/inbox";

// Type icons mapping for inbox items
const inboxTypeIcons: Record<InboxItemType, typeof GitCompare> = {
  execution_selection: GitCompare,
  thread_review: FileCheck,
  planning_approval: ClipboardCheck,
  task_plan_approval: ClipboardCheck,
  questionnaire: HelpCircle,
  decision_escalation: GitCompare,
  direction_clarification: HelpCircle,
};

export function CommandPalette() {
  const [open, setOpen] = React.useState(false);
  const navigate = useNavigate();

  // Handle keyboard shortcut
  React.useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "p" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setOpen((open) => !open);
      }
    };

    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, []);

  const handleSelect = (path: string) => {
    setOpen(false);
    navigate({ to: path });
  };

  // Get pending inbox items
  const pendingItems = inboxItems.filter((item) => item.status === "pending");

  return (
    <CommandDialog
      open={open}
      onOpenChange={setOpen}
      title="Command Palette"
      description="Search for threads, tasks, and navigation"
      overlayClassName="bg-black/20 backdrop-blur-[1px]"
    >
      <Command className="rounded-lg">
        <CommandInput placeholder="Search threads, tasks, or navigate..." />
        <CommandList className="max-h-[400px]">
          <CommandEmpty>No results found.</CommandEmpty>

          {/* Navigation */}
          <CommandGroup heading="Navigation">
            <CommandItem onSelect={() => handleSelect("/app/inbox")}>
              <Inbox className="text-muted-foreground" />
              <span>Inbox</span>
              {pendingItems.length > 0 && (
                <span className="ml-auto text-xs text-amber-400">
                  {pendingItems.length} pending
                </span>
              )}
            </CommandItem>
            <CommandItem onSelect={() => handleSelect("/app/threads")}>
              <GitBranch className="text-muted-foreground" />
              <span>Threads</span>
            </CommandItem>
            <CommandItem onSelect={() => handleSelect("/app/overseer")}>
              <Bot className="text-muted-foreground" />
              <span>Project Overseer</span>
            </CommandItem>
            <CommandItem onSelect={() => handleSelect("/app/settings")}>
              <Settings className="text-muted-foreground" />
              <span>Settings</span>
            </CommandItem>
          </CommandGroup>

          <CommandSeparator />

          {/* Threads */}
          <CommandGroup heading="Threads">
            {threads.map((thread) => {
              const StatusIcon = threadStatusConfig[thread.status].icon;
              const progress =
                thread.taskCount > 0
                  ? Math.round((thread.completedTasks / thread.taskCount) * 100)
                  : 0;

              return (
                <CommandItem
                  key={thread.id}
                  value={`thread-${thread.id}-${thread.title}`}
                  onSelect={() => handleSelect(`/app/threads/${thread.id}`)}
                >
                  <StatusIcon className={cn("shrink-0", threadStatusConfig[thread.status].color)} />
                  <span className="text-muted-foreground tabular-nums">#{thread.id}</span>
                  <span className="flex-1 truncate">{thread.title}</span>
                  <span className="text-xs text-muted-foreground tabular-nums">
                    {thread.completedTasks}/{thread.taskCount}
                  </span>
                  {thread.status === "in_progress" && (
                    <span className="text-xs text-muted-foreground tabular-nums w-8 text-right">
                      {progress}%
                    </span>
                  )}
                </CommandItem>
              );
            })}
          </CommandGroup>

          <CommandSeparator />

          {/* Pending Inbox Items */}
          {pendingItems.length > 0 && (
            <CommandGroup heading="Pending Items">
              {pendingItems.map((item) => {
                const TypeIcon = inboxTypeIcons[item.type];

                return (
                  <CommandItem
                    key={item.id}
                    value={`inbox-${item.id}-${item.title}-${item.description}`}
                    onSelect={() => handleSelect(`/app/inbox/${item.id}`)}
                  >
                    <TypeIcon className="text-muted-foreground shrink-0" />
                    <span className="flex-1 truncate">
                      {item.title}: {item.description}
                    </span>
                    <span
                      className={cn(
                        "text-xs capitalize",
                        item.priority === "high" && "text-red-400",
                        item.priority === "medium" && "text-amber-400",
                        item.priority === "low" && "text-slate-400",
                      )}
                    >
                      {item.priority}
                    </span>
                  </CommandItem>
                );
              })}
            </CommandGroup>
          )}
        </CommandList>
      </Command>
    </CommandDialog>
  );
}
