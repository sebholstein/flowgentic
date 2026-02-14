"use client";

import * as React from "react";
import { useMemo } from "react";
import { useAtom } from "jotai";
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
import { useQuery, useQueries } from "@tanstack/react-query";
import { useClient } from "@/lib/connect";
import { ProjectService } from "@/proto/gen/controlplane/v1/project_service_pb";
import { ThreadService } from "@/proto/gen/controlplane/v1/thread_service_pb";
import { projectsQueryOptions } from "@/lib/queries/projects";
import { threadsQueryOptions } from "@/lib/queries/threads";
import type { ThreadConfig } from "@/proto/gen/controlplane/v1/thread_service_pb";

// Import inbox data
import { inboxItems } from "@/data/mockInboxData";
import type { InboxItemType } from "@/types/inbox";
import { commandPaletteOpenAtom } from "@/stores/atoms";

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
  const [open, setOpen] = useAtom(commandPaletteOpenAtom);
  const navigate = useNavigate();

  const projectClient = useClient(ProjectService);
  const threadClient = useClient(ThreadService);

  const { data: projectsData } = useQuery(projectsQueryOptions(projectClient));
  const fetchedProjects = useMemo(
    () => (projectsData?.projects ?? []).map((p) => ({ id: p.id, name: p.name })),
    [projectsData],
  );

  const threadQueries = useQueries({
    queries: fetchedProjects.map((p) => ({
      ...threadsQueryOptions(threadClient, p.id),
      enabled: open,
    })),
  });

  const threadsByProject = useMemo(() => {
    const groups: { project: { id: string; name: string }; threads: ThreadConfig[] }[] = [];
    for (let i = 0; i < fetchedProjects.length; i++) {
      const p = fetchedProjects[i];
      const q = threadQueries[i];
      const threads = q?.data?.threads ?? [];
      if (threads.length > 0) {
        groups.push({ project: p, threads: [...threads] });
      }
    }
    return groups;
  }, [fetchedProjects, threadQueries]);

  // Handle keyboard shortcut
  React.useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === "p" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setOpen((open) => !open);
      }
    };

    document.addEventListener("keydown", down);
    return () => {
      document.removeEventListener("keydown", down);
    };
  }, [setOpen]);

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

          {/* Threads grouped by project */}
          {threadsByProject.map(({ project, threads }) => (
            <React.Fragment key={project.id}>
              <CommandSeparator />
              <CommandGroup heading={project.name}>
                {threads.map((thread) => (
                    <CommandItem
                      key={thread.id}
                      value={`thread-${thread.id}-${thread.topic}`}
                      onSelect={() => handleSelect(`/app/threads/${thread.id}`)}
                    >
                      <GitBranch className="shrink-0 text-muted-foreground" />
                      <span className="flex-1 truncate">{thread.topic || "Untitled"}</span>
                    </CommandItem>
                ))}
              </CommandGroup>
            </React.Fragment>
          ))}

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
