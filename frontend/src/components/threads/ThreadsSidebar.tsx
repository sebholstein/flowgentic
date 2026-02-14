import { useState, useMemo, useRef, useEffect, useCallback } from "react";
import { Link, useSearch, useNavigate } from "@tanstack/react-router";
import { useVirtualizer } from "@tanstack/react-virtual";
import {
  DndContext,
  DragOverlay,
  closestCenter,
  PointerSensor,
  KeyboardSensor,
  useSensor,
  useSensors,
  type DragStartEvent,
  type DragEndEvent,
  type DragOverEvent,
  type DraggableAttributes,
} from "@dnd-kit/core";
import {
  SortableContext,
  verticalListSortingStrategy,
  useSortable,
  arrayMove,
} from "@dnd-kit/sortable";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  PanelLeftClose,
  ChevronRight,
  ChevronDown,
  Folder,
  FolderOpen,
  Pin,
  Archive,
  Plus,
} from "lucide-react";
import { useQuery, useQueries, useMutation, useQueryClient } from "@tanstack/react-query";
import { inboxItems } from "@/data/mockInboxData";
import { threadStatusConfig } from "@/constants/threadStatusConfig";
import { taskStatusConfig } from "@/constants/taskStatusConfig";
import { FeedbackList } from "./FeedbackList";
import type { Thread } from "@/types/thread";
import type { Project } from "@/types/project";
import type { Task } from "@/types/task";
import { useSidebarStore } from "@/stores/sidebarStore";
import { WindowDragHeader } from "@/components/layout/WindowDragHeader";
import { useIsMacOS } from "@/hooks/use-electron";
import { NewProjectDialog } from "@/components/projects/NewProjectDialog";
import { useClient } from "@/lib/connect";
import { useTypewriter } from "@/hooks/use-typewriter";
import { ProjectService } from "@/proto/gen/controlplane/v1/project_service_pb";
import { WorkerService } from "@/proto/gen/controlplane/v1/worker_service_pb";
import { ThreadService } from "@/proto/gen/controlplane/v1/thread_service_pb";
import { projectsQueryOptions } from "@/lib/queries/projects";
import { workersQueryOptions } from "@/lib/queries/workers";
import { threadsQueryOptions } from "@/lib/queries/threads";

// Flattened tree node types for virtualization
type FlatTreeNode =
  | { type: "project"; project: Project; threadCount: number }
  | { type: "thread"; thread: Thread; projectId: string; hasChildren: boolean }
  | { type: "task"; task: Task; threadId: string };

// Flat row components for virtualized tree

function ProjectRow({
  project,
  threadCount,
  isExpanded,
  isDragging,
  onToggle,
  onAddThread,
  sortableRef,
  dragListeners,
  dragAttributes,
}: {
  project: Project;
  threadCount: number;
  isExpanded: boolean;
  isDragging?: boolean;
  onToggle: () => void;
  onAddThread: () => void;
  sortableRef?: (node: HTMLElement | null) => void;
  dragListeners?: Record<string, Function>;
  dragAttributes?: DraggableAttributes;
}) {
  return (
    <div
      ref={sortableRef}
      className={cn(
        "group/project flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-sm hover:bg-muted/50 transition-colors text-left select-none",
        "text-foreground font-medium",
        isDragging && "opacity-30",
      )}
      style={{ paddingLeft: "8px" }}
    >
      <button
        type="button"
        onClick={onToggle}
        className="size-4 flex items-center justify-center shrink-0"
      >
        {isExpanded ? <ChevronDown className="size-3.5" /> : <ChevronRight className="size-3.5" />}
      </button>
      <span
        className="flex flex-1 items-center gap-1.5 min-w-0 cursor-grab active:cursor-grabbing"
        {...dragListeners}
        {...dragAttributes}
      >
        {isExpanded ? (
          <FolderOpen className={cn("size-4 shrink-0", project.color || "text-amber-400")} />
        ) : (
          <Folder className={cn("size-4 shrink-0", project.color || "text-amber-400")} />
        )}
        <span className="truncate flex-1 text-left">{project.name}</span>
      </span>
      <span className="text-xs text-muted-foreground tabular-nums group-hover/project:hidden">
        {threadCount}
      </span>
      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation();
          onAddThread();
        }}
        className="hidden group-hover/project:flex items-center justify-center rounded p-0.5 text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
        aria-label={`Add thread to ${project.name}`}
        title="Add thread"
      >
        <Plus className="size-3.5" />
      </button>
    </div>
  );
}

function ProjectRowOverlay({ project, threadCount }: { project: Project; threadCount: number }) {
  return (
    <div
      className={cn(
        "flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-sm",
        "text-foreground font-medium",
        "bg-sidebar shadow-lg border border-border cursor-grabbing",
      )}
      style={{ paddingLeft: "8px" }}
    >
      <span className="size-4 shrink-0" />
      <Folder className={cn("size-4 shrink-0", project.color || "text-amber-400")} />
      <span className="truncate flex-1 text-left">{project.name}</span>
      <span className="text-xs text-muted-foreground tabular-nums">{threadCount}</span>
    </div>
  );
}

function ThreadRow({
  thread,
  isSelected,
  isExpanded,
  hasChildren,
  isPinned,
  isArchived,
  onToggle,
  onTogglePin,
  onToggleArchive,
}: {
  thread: Thread;
  isSelected: boolean;
  isExpanded: boolean;
  hasChildren: boolean;
  isPinned: boolean;
  isArchived: boolean;
  onToggle: () => void;
  onTogglePin: () => void;
  onToggleArchive: () => void;
}) {
  const StatusIcon = threadStatusConfig[thread.status].icon;
  const animatedTopic = useTypewriter(thread.topic);

  return (
    <div
      className={cn(
        "group flex items-center gap-1 pr-2 rounded-md transition-colors",
        isSelected ? "bg-muted text-foreground" : "text-muted-foreground hover:bg-muted/50",
      )}
      style={{ paddingLeft: "24px" }}
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
        to="/app/threads/$threadId"
        params={{ threadId: thread.id }}
        className={cn(
          "flex flex-1 items-center gap-1.5 px-1.5 py-1.5 text-sm transition-colors text-left min-w-0 select-none",
          isArchived && "opacity-60",
        )}
      >
        <StatusIcon className={cn("size-3.5 shrink-0", threadStatusConfig[thread.status].color)} />
        <span className="truncate flex-1 text-xs font-medium">{animatedTopic}</span>
      </Link>
      <div
        className={cn(
          "flex items-center gap-1 opacity-0 transition-opacity",
          "group-hover:opacity-100",
          (isPinned || isArchived) && "opacity-100",
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
}

function TaskRow({
  task,
  threadId,
  isSelected,
}: {
  task: Task;
  threadId: string;
  isSelected: boolean;
}) {
  const StatusIcon = taskStatusConfig[task.status].icon;

  return (
    <Link
      to="/app/tasks/$threadId/$taskId"
      params={{ threadId, taskId: task.id }}
      className={cn(
        "flex items-center gap-1.5 rounded-md px-1.5 py-1 text-xs hover:bg-muted/50 transition-colors text-left min-w-0 select-none",
        isSelected && "bg-muted text-foreground",
        !isSelected && "text-foreground hover:text-foreground",
      )}
      style={{ paddingLeft: "48px" }}
    >
      <StatusIcon
        className={cn(
          "size-3 shrink-0",
          taskStatusConfig[task.status].color,
          task.status === "running" && "animate-spin",
        )}
      />
      <span className="truncate flex-1">{task.name}</span>
    </Link>
  );
}

// Sortable wrapper for project rows — calls useSortable at top level (rules of hooks)
function SortableProjectRow({
  project,
  threadCount,
  isExpanded,
  onToggle,
  onAddThread,
}: {
  project: Project;
  threadCount: number;
  isExpanded: boolean;
  onToggle: () => void;
  onAddThread: () => void;
}) {
  const { setNodeRef, attributes, listeners, isDragging } = useSortable({
    id: project.id,
  });
  return (
    <ProjectRow
      project={project}
      threadCount={threadCount}
      isExpanded={isExpanded}
      isDragging={isDragging}
      onToggle={onToggle}
      onAddThread={onAddThread}
      sortableRef={setNodeRef}
      dragListeners={listeners}
      dragAttributes={attributes}
    />
  );
}

// Tree node row renderer
function TreeNodeRow({
  node,
  selectedThreadId,
  selectedTaskId,
  expandedProjects,
  expandedThreads,
  pinnedThreads,
  archivedThreads,
  onToggleProject,
  onToggleThread,
  onTogglePin,
  onToggleArchive,
  onAddThread,
}: {
  node: FlatTreeNode;
  selectedThreadId: string | null;
  selectedTaskId: string | null;
  expandedProjects: Set<string>;
  expandedThreads: Set<string>;
  pinnedThreads: Set<string>;
  archivedThreads: Set<string>;
  onToggleProject: (id: string) => void;
  onToggleThread: (id: string) => void;
  onTogglePin: (id: string) => void;
  onToggleArchive: (id: string) => void;
  onAddThread: (projectId: string) => void;
}) {
  switch (node.type) {
    case "project":
      return (
        <SortableProjectRow
          project={node.project}
          threadCount={node.threadCount}
          isExpanded={expandedProjects.has(node.project.id)}
          onToggle={() => onToggleProject(node.project.id)}
          onAddThread={() => onAddThread(node.project.id)}
        />
      );
    case "thread":
      return (
        <ThreadRow
          thread={node.thread}
          isSelected={selectedThreadId === node.thread.id && !selectedTaskId}
          isExpanded={expandedThreads.has(node.thread.id)}
          hasChildren={node.hasChildren}
          isPinned={pinnedThreads.has(node.thread.id)}
          isArchived={archivedThreads.has(node.thread.id)}
          onToggle={() => onToggleThread(node.thread.id)}
          onTogglePin={() => onTogglePin(node.thread.id)}
          onToggleArchive={() => onToggleArchive(node.thread.id)}
        />
      );
    case "task":
      return (
        <TaskRow
          task={node.task}
          threadId={node.threadId}
          isSelected={selectedTaskId === node.task.id}
        />
      );
  }
}

// Tab types
type SidebarTab = "threads" | "feedback" | "archived";

// Thread sidebar component
export function ThreadsSidebar({
  selectedThreadId,
  selectedTaskId,
}: {
  selectedThreadId: string | null;
  selectedTaskId: string | null;
}) {
  const hideSidebar = useSidebarStore((s) => s.hide);
  const isMacOS = useIsMacOS();
  const scrollRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();
  const search = useSearch({ strict: false }) as { feedback?: string };
  const queryClient = useQueryClient();
  const projectClient = useClient(ProjectService);
  const workerClient = useClient(WorkerService);
  const threadClient = useClient(ThreadService);
  const { data: projectsData } = useQuery(projectsQueryOptions(projectClient));
  const { data: workersData } = useQuery(workersQueryOptions(workerClient));
  const fetchedProjects = useMemo<Project[]>(
    () =>
      (projectsData?.projects ?? []).map((p) => ({
        id: p.id,
        name: p.name,
        defaultPlannerAgent: p.defaultPlannerAgent,
        defaultPlannerModel: p.defaultPlannerModel,
        embeddedWorkerPath: p.embeddedWorkerPath,
        workerPaths: p.workerPaths,
        sortIndex: p.sortIndex,
      })),
    [projectsData],
  );
  const threadQueries = useQueries({
    queries: fetchedProjects.map((p) => threadsQueryOptions(threadClient, p.id)),
  });

  const backendThreads = useMemo<Thread[]>(() => {
    const result: Thread[] = [];
    for (const q of threadQueries) {
      for (const t of q.data?.threads ?? []) {
        result.push({
          id: t.id,
          topic: t.topic || "Untitled",
          description: "",
          status: "pending",
          taskCount: 0,
          completedTasks: 0,
          createdAt: t.createdAt ?? "",
          updatedAt: t.updatedAt ?? "",
          overseer: { id: "overseer", name: "Overseer" },
          projectId: t.projectId,
          mode: (t.mode === "plan" || t.mode === "build") ? t.mode : "build",
          model: t.model,
          harness: t.agent,
          archived: t.archived,
        });
      }
    }
    return result;
  }, [threadQueries]);

  const [activeTab, setActiveTab] = useState<SidebarTab>(() =>
    search.feedback ? "feedback" : "threads",
  );
  const [searchQuery, setSearchQuery] = useState("");
  const [activeDragId, setActiveDragId] = useState<string | null>(null);
  const [overProjectId, setOverProjectId] = useState<string | null>(null);
  const [expandedProjects, setExpandedProjects] = useState<Set<string>>(
    () => new Set<string>(),
  );

  // Auto-expand all projects when they first load
  useEffect(() => {
    if (fetchedProjects.length === 0) return;
    setExpandedProjects((prev) => {
      if (prev.size > 0) return prev;
      return new Set(fetchedProjects.map((p) => p.id));
    });
  }, [fetchedProjects]);
  const [expandedThreads, setExpandedThreads] = useState<Set<string>>(() => {
    // Auto-expand threads that contain the selected task
    if (selectedTaskId && selectedThreadId) {
      return new Set([selectedThreadId]);
    }
    return new Set();
  });
  const [pinnedThreads, setPinnedThreads] = useState<Set<string>>(() => new Set());
  const archivedThreads = useMemo(
    () => new Set(backendThreads.filter((t) => t.archived).map((t) => t.id)),
    [backendThreads],
  );
  const [newProjectDialogOpen, setNewProjectDialogOpen] = useState(false);
  // showArchived is now derived from activeTab
  const showArchived = activeTab === "archived";

  // Drag-and-drop sensors
  const projectIds = useMemo(() => fetchedProjects.map((p) => p.id), [fetchedProjects]);
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
    useSensor(KeyboardSensor),
  );

  const handleDragStart = useCallback((event: DragStartEvent) => {
    setActiveDragId(String(event.active.id));
  }, []);

  const handleDragOver = useCallback((event: DragOverEvent) => {
    setOverProjectId(event.over ? String(event.over.id) : null);
  }, []);

  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      const { active, over } = event;
      if (over && active.id !== over.id) {
        const oldIndex = fetchedProjects.findIndex((p) => p.id === active.id);
        const newIndex = fetchedProjects.findIndex((p) => p.id === over.id);
        if (oldIndex !== -1 && newIndex !== -1) {
          const reordered = arrayMove(fetchedProjects, oldIndex, newIndex);
          const entries = reordered.map((p, i) => ({ id: p.id, sortIndex: i }));
          projectClient.reorderProjects({ entries }).then(() => {
            queryClient.invalidateQueries({ queryKey: ["projects"] });
          });
        }
      }
      setActiveDragId(null);
      setOverProjectId(null);
    },
    [fetchedProjects, projectClient, queryClient],
  );

  const handleDragCancel = useCallback(() => {
    setActiveDragId(null);
    setOverProjectId(null);
  }, []);

  const activeProject = activeDragId ? fetchedProjects.find((p) => p.id === activeDragId) : null;

  // Auto-expand the selected thread when navigating to it
  useEffect(() => {
    if (selectedThreadId) {
      setExpandedThreads((prev) => {
        if (prev.has(selectedThreadId)) return prev;
        return new Set(prev).add(selectedThreadId);
      });
    }
  }, [selectedThreadId]);

  // Keep feedback tab active when navigating between feedback items
  useEffect(() => {
    if (search.feedback) {
      setActiveTab("feedback");
    }
  }, [search.feedback]);

  // Count pending feedback items
  const pendingFeedbackCount = useMemo(() => {
    return inboxItems.filter((item) => item.status === "pending").length;
  }, []);

  // Group threads by project
  const threadsByProject = useMemo(() => {
    const filtered = searchQuery
      ? backendThreads.filter((t) => t.topic.toLowerCase().includes(searchQuery.toLowerCase()))
      : backendThreads;

    // When on archived tab, only show archived. Otherwise, only show non-archived.
    const visible = filtered.filter((t) =>
      showArchived ? archivedThreads.has(t.id) : !archivedThreads.has(t.id),
    );

    const grouped = new Map<string, Thread[]>();
    for (const thread of visible) {
      const existing = grouped.get(thread.projectId) ?? [];
      existing.push(thread);
      grouped.set(thread.projectId, existing);
    }

    for (const [, projectThreads] of grouped.entries()) {
      projectThreads.sort((a, b) => {
        const aArchived = archivedThreads.has(a.id);
        const bArchived = archivedThreads.has(b.id);
        if (aArchived !== bArchived) return aArchived ? 1 : -1;

        const aPinned = pinnedThreads.has(a.id);
        const bPinned = pinnedThreads.has(b.id);
        if (aPinned !== bPinned) return aPinned ? -1 : 1;

        return 0;
      });
    }

    return grouped;
  }, [searchQuery, backendThreads, showArchived, archivedThreads, pinnedThreads]);

  // Flatten tree for virtualization
  const flattenedNodes = useMemo(() => {
    const nodes: FlatTreeNode[] = [];

    for (const project of fetchedProjects) {
      const projectThreads = threadsByProject.get(project.id) ?? [];
      if (searchQuery && projectThreads.length === 0) continue;

      // Add project node
      nodes.push({ type: "project", project, threadCount: projectThreads.length });

      if (expandedProjects.has(project.id)) {
        for (const thread of projectThreads) {
          nodes.push({
            type: "thread",
            thread,
            projectId: project.id,
            hasChildren: false,
          });
        }
      }
    }

    return nodes;
  }, [fetchedProjects, threadsByProject, expandedProjects, expandedThreads, searchQuery]);

  // Compute which flattened node index to show the drop indicator at
  const dropIndicator = useMemo(() => {
    if (!activeDragId || !overProjectId || activeDragId === overProjectId) return null;

    const activeIdx = fetchedProjects.findIndex((p) => p.id === activeDragId);
    const overIdx = fetchedProjects.findIndex((p) => p.id === overProjectId);
    if (activeIdx === -1 || overIdx === -1) return null;

    if (activeIdx > overIdx) {
      // Dragging upward → line above the target project header
      const nodeIdx = flattenedNodes.findIndex(
        (n) => n.type === "project" && n.project.id === overProjectId,
      );
      return nodeIdx !== -1 ? { nodeIndex: nodeIdx, position: "above" as const } : null;
    }

    // Dragging downward → line below the last child of the target project group
    const projectNodeIdx = flattenedNodes.findIndex(
      (n) => n.type === "project" && n.project.id === overProjectId,
    );
    if (projectNodeIdx === -1) return null;

    let lastIdx = projectNodeIdx;
    for (let i = projectNodeIdx + 1; i < flattenedNodes.length; i++) {
      if (flattenedNodes[i].type === "project") break;
      lastIdx = i;
    }
    return { nodeIndex: lastIdx, position: "below" as const };
  }, [activeDragId, overProjectId, fetchedProjects, flattenedNodes]);

  // Thread count for the active drag overlay
  const activeDragThreadCount = activeProject
    ? (threadsByProject.get(activeProject.id) ?? []).length
    : 0;

  // Set up virtualizer
  const virtualizer = useVirtualizer({
    count: flattenedNodes.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: (index) => {
      const node = flattenedNodes[index];
      switch (node.type) {
        case "project":
          return 36;
        case "thread":
          return 32;
        case "task":
          return 28;
      }
    },
    overscan: 10,
    getItemKey: (index) => {
      const node = flattenedNodes[index];
      switch (node.type) {
        case "project":
          return `p-${node.project.id}`;
        case "thread":
          return `t-${node.thread.id}`;
        case "task":
          return `tk-${node.task.id}`;
      }
    },
  });

  const toggleProject = (id: string) => {
    setExpandedProjects((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const toggleThread = (id: string) => {
    setExpandedThreads((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const togglePin = (id: string) => {
    if (archivedThreads.has(id)) return;
    setPinnedThreads((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const archiveThreadMutation = useMutation({
    mutationFn: (data: { id: string; archived: boolean }) =>
      threadClient.archiveThread({ id: data.id, archived: data.archived }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["threads"] });
    },
  });

  const toggleArchive = (id: string) => {
    const newArchived = !archivedThreads.has(id);
    archiveThreadMutation.mutate({ id, archived: newArchived });
    if (newArchived) {
      setPinnedThreads((prev) => {
        if (!prev.has(id)) return prev;
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
    }
  };

  const handleAddThread = (projectId: string) => {
    navigate({ to: "/app/threads/new", search: { projectId } });
  };

  const createProjectMutation = useMutation({
    mutationFn: (data: {
      id: string;
      name: string;
      defaultPlannerAgent: string;
      defaultPlannerModel: string;
      embeddedWorkerPath: string;
      workerPaths: Record<string, string>;
    }) =>
      projectClient.createProject({
        id: data.id,
        name: data.name,
        defaultPlannerAgent: data.defaultPlannerAgent,
        defaultPlannerModel: data.defaultPlannerModel,
        embeddedWorkerPath: data.embeddedWorkerPath,
        workerPaths: data.workerPaths,
      }),
    onSuccess: (resp) => {
      queryClient.invalidateQueries({ queryKey: ["projects"] });
      if (resp.project) {
        setExpandedProjects((prev) => new Set(prev).add(resp.project!.id));
      }
      setNewProjectDialogOpen(false);
    },
  });

  const handleCreateProject = (data: {
    id: string;
    name: string;
    defaultPlannerAgent: string;
    defaultPlannerModel: string;
    embeddedWorkerPath: string;
    workerPaths: Record<string, string>;
    defaultWorkerId: string;
  }) => {
    createProjectMutation.mutate(data);
  };

  return (
    <div className="flex h-full flex-col bg-sidebar select-none">
      <div className="relative">
        <WindowDragHeader />
        <Button
          variant="ghost"
          size="sm"
          onClick={hideSidebar}
          className="absolute right-1 top-2 size-6 p-0 text-muted-foreground hover:text-foreground"
        >
          <PanelLeftClose className="size-3.5" />
        </Button>
      </div>
      <div className={cn("flex flex-col gap-2.5 border-b p-4 pt-0 pb-3", isMacOS && "mt-4")}>
        <div className="text-sm font-medium text-foreground">Threads</div>
        {/* Tab switcher - compact with space-between */}
        <div className="flex justify-between -mt-1">
          <div className="flex gap-1">
            <button
              type="button"
              onClick={() => setActiveTab("threads")}
              className={cn(
                "px-2.5 py-1 text-xs font-medium rounded-md transition-colors cursor-pointer",
                activeTab === "threads"
                  ? "bg-muted text-foreground"
                  : "text-muted-foreground hover:text-foreground hover:bg-muted/50",
              )}
            >
              Browse
            </button>
            <button
              type="button"
              onClick={() => setActiveTab("feedback")}
              className={cn(
                "px-2.5 py-1 text-xs font-medium rounded-md transition-colors cursor-pointer flex items-center gap-1.5",
                activeTab === "feedback"
                  ? "bg-muted text-foreground"
                  : "text-muted-foreground hover:text-foreground hover:bg-muted/50",
              )}
            >
              Inbox
              {pendingFeedbackCount > 0 && (
                <span className="text-[0.6rem] px-1.5 py-0.5 rounded-full min-w-[1.25rem] text-center bg-rose-400/20 text-rose-600 dark:bg-rose-500/20 dark:text-rose-400 font-medium">
                  {pendingFeedbackCount}
                </span>
              )}
            </button>
          </div>
          <button
            type="button"
            onClick={() => setActiveTab("archived")}
            className={cn(
              "px-2.5 py-1 text-xs font-medium rounded-md transition-colors cursor-pointer",
              activeTab === "archived"
                ? "bg-muted text-foreground"
                : "text-muted-foreground hover:text-foreground hover:bg-muted/50",
            )}
          >
            Archived
          </button>
        </div>
        {(activeTab === "threads" || activeTab === "archived") && (
          <Input
            placeholder="Search threads..."
            className="h-8"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        )}
      </div>

      {activeTab === "feedback" ? (
        <FeedbackList selectedThreadId={selectedThreadId} selectedTaskId={selectedTaskId} />
      ) : (
        <>
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragStart={handleDragStart}
          onDragOver={handleDragOver}
          onDragEnd={handleDragEnd}
          onDragCancel={handleDragCancel}
        >
          <SortableContext items={projectIds} strategy={verticalListSortingStrategy}>
            <ScrollArea
              className="flex-1 overflow-hidden px-2 pt-2"
              viewportRef={scrollRef}
              viewportClassName="!overflow-y-auto"
            >
              {activeTab === "threads" && (
                <button
                  type="button"
                  onClick={() => setNewProjectDialogOpen(true)}
                  className="flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-sm hover:bg-muted/50 transition-colors text-left select-none text-muted-foreground mb-1 mx-2"
                  style={{ paddingLeft: "8px" }}
                >
                  <span className="size-4 flex items-center justify-center shrink-0">
                    <Plus className="size-3.5" />
                  </span>
                  <span className="truncate flex-1">Add Project</span>
                </button>
              )}
              <div
                className="p-2"
                style={{
                  height: `${virtualizer.getTotalSize()}px`,
                  width: "100%",
                  position: "relative",
                }}
              >
                {virtualizer.getVirtualItems().map((virtualRow) => {
                  const node = flattenedNodes[virtualRow.index];
                  return (
                    <div
                      key={virtualRow.key}
                      data-index={virtualRow.index}
                      ref={virtualizer.measureElement}
                      style={{
                        position: "absolute",
                        top: 0,
                        left: 0,
                        width: "100%",
                        transform: `translateY(${virtualRow.start}px)`,
                      }}
                    >
                      <TreeNodeRow
                        node={node}
                        selectedThreadId={selectedThreadId}
                        selectedTaskId={selectedTaskId}
                        expandedProjects={expandedProjects}
                        expandedThreads={expandedThreads}
                        pinnedThreads={pinnedThreads}
                        archivedThreads={archivedThreads}
                        onToggleProject={toggleProject}
                        onToggleThread={toggleThread}
                        onTogglePin={togglePin}
                        onToggleArchive={toggleArchive}
                        onAddThread={handleAddThread}
                      />
                    </div>
                  );
                })}
                {dropIndicator &&
                  (() => {
                    const items = virtualizer.getVirtualItems();
                    const target = items.find((item) => item.index === dropIndicator.nodeIndex);
                    if (!target) return null;
                    const y =
                      dropIndicator.position === "above"
                        ? target.start
                        : target.start + target.size;
                    return (
                      <div
                        className="absolute left-2 right-2 z-10 pointer-events-none"
                        style={{ top: `${y - 1}px` }}
                      >
                        <div className="h-0.5 bg-primary rounded-full" />
                      </div>
                    );
                  })()}
              </div>
            </ScrollArea>
          </SortableContext>
          <DragOverlay dropAnimation={null}>
            {activeProject ? (
              <ProjectRowOverlay project={activeProject} threadCount={activeDragThreadCount} />
            ) : null}
          </DragOverlay>
        </DndContext>
        <NewProjectDialog
          open={newProjectDialogOpen}
          onOpenChange={setNewProjectDialogOpen}
          onSave={handleCreateProject}
          workers={workersData?.workers ?? []}
        />
        </>
      )}
    </div>
  );
}
