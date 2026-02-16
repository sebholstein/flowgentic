import { useState, useMemo, useEffect, useCallback } from "react";
import {
  PointerSensor,
  KeyboardSensor,
  useSensor,
  useSensors,
  type DragStartEvent,
  type DragEndEvent,
  type DragOverEvent,
} from "@dnd-kit/core";
import { arrayMove } from "@dnd-kit/sortable";
import {
  Separator as PanelResizeHandle,
  usePanelRef,
} from "react-resizable-panels";
import {
  ResizablePanelGroup,
  ResizablePanel,
} from "@/components/ui/resizable";
import type { ThreadConfig } from "@/proto/gen/controlplane/v1/thread_service_pb";
import type { Project } from "@/types/project";
import { demoProject, demoThreads } from "@/data/mockFlowgenticData";
import { useDemoInboxItems } from "./FeedbackList";
import { PlanTemplates } from "./PlanTemplates";
import { NewProjectDialog } from "@/components/projects/NewProjectDialog";
import {
  useSidebarData,
  useArchiveThread,
  useCreateThread,
  useCreateProject,
  useReorderProjects,
} from "./queries";
import { SidebarHeader } from "./SidebarHeader";
import { ThreadTree } from "./ThreadTree";
import { InboxPanel } from "./InboxPanel";
import type { FlatTreeNode } from "./sidebar-types";

export function MainSidebar({
  selectedThreadId,
  selectedTaskId,
  activeView,
}: {
  selectedThreadId: string | null;
  selectedTaskId: string | null;
  activeView: "threads" | "templates";
}) {
  const { fetchedProjects, backendThreads, workersData } = useSidebarData();

  // --- UI state ---
  const [activeTab, setActiveTab] = useState<"threads" | "archived">("threads");
  const [searchQuery, setSearchQuery] = useState("");
  const [activeDragId, setActiveDragId] = useState<string | null>(null);
  const [overProjectId, setOverProjectId] = useState<string | null>(null);
  const [expandedProjects, setExpandedProjects] = useState<Set<string>>(() => new Set<string>());
  const [expandedThreads, setExpandedThreads] = useState<Set<string>>(() => {
    if (selectedTaskId && selectedThreadId) {
      return new Set([selectedThreadId]);
    }
    return new Set();
  });
  const [pinnedThreads, setPinnedThreads] = useState<Set<string>>(() => new Set());
  const [newProjectDialogOpen, setNewProjectDialogOpen] = useState(false);
  const [isInboxCollapsed, setIsInboxCollapsed] = useState(false);
  const inboxPanelRef = usePanelRef();

  const showArchived = activeTab === "archived";

  const archivedThreads = useMemo(
    () => new Set(backendThreads.filter((t) => t.archived).map((t) => t.id)),
    [backendThreads],
  );

  // --- Mutations ---
  const archiveThreadMutation = useArchiveThread();
  const createThreadMutation = useCreateThread();
  const { reorder: reorderProjects } = useReorderProjects();
  const createProjectMutation = useCreateProject({
    onExpandProject: (id) => setExpandedProjects((prev) => new Set(prev).add(id)),
    onCloseDialog: () => setNewProjectDialogOpen(false),
  });

  // --- Demo inbox ---
  const demoInboxItems = useDemoInboxItems();
  const pendingFeedbackCount = demoInboxItems.length;

  // --- Auto-expand ---
  useEffect(() => {
    if (fetchedProjects.length === 0) return;
    setExpandedProjects((prev) => {
      if (prev.size > 0) return prev;
      return new Set([...fetchedProjects.map((p) => p.id), demoProject.id]);
    });
  }, [fetchedProjects]);

  useEffect(() => {
    if (selectedThreadId) {
      setExpandedThreads((prev) => {
        if (prev.has(selectedThreadId)) return prev;
        return new Set(prev).add(selectedThreadId);
      });
    }
  }, [selectedThreadId]);

  // --- DnD ---
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
          reorderProjects(entries);
        }
      }
      setActiveDragId(null);
      setOverProjectId(null);
    },
    [fetchedProjects, reorderProjects],
  );

  const handleDragCancel = useCallback(() => {
    setActiveDragId(null);
    setOverProjectId(null);
  }, []);

  const activeProject = activeDragId ? fetchedProjects.find((p) => p.id === activeDragId) : null;

  // --- Thread grouping ---
  const threadsByProject = useMemo(() => {
    const filtered = searchQuery
      ? backendThreads.filter((t) => t.topic.toLowerCase().includes(searchQuery.toLowerCase()))
      : backendThreads;

    const visible = filtered.filter((t) =>
      showArchived ? archivedThreads.has(t.id) : !archivedThreads.has(t.id),
    );

    const grouped = new Map<string, ThreadConfig[]>();
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

  // --- Flatten tree ---
  const flattenedNodes = useMemo(() => {
    const nodes: FlatTreeNode[] = [];

    for (const project of fetchedProjects) {
      const projectThreads = threadsByProject.get(project.id) ?? [];
      if (searchQuery && projectThreads.length === 0) continue;

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

    // Demo project
    if (!searchQuery || "Flowgentic demo".includes(searchQuery.toLowerCase())) {
      const demoProjectNode: Project = {
        id: demoProject.id,
        name: demoProject.name,
        color: demoProject.color,
        defaultPlannerAgent: "",
        defaultPlannerModel: "",
        embeddedWorkerPath: "",
        workerPaths: {},
        sortIndex: 999,
      };

      const filteredDemoThreads = searchQuery
        ? demoThreads.filter((t) =>
            t.topic.toLowerCase().includes(searchQuery.toLowerCase()),
          )
        : demoThreads;

      if (filteredDemoThreads.length > 0 || !searchQuery) {
        nodes.push({
          type: "project",
          project: demoProjectNode,
          threadCount: filteredDemoThreads.length,
          isDemo: true,
        });

        if (expandedProjects.has(demoProject.id)) {
          for (const dt of filteredDemoThreads) {
            nodes.push({
              type: "thread",
              thread: { id: dt.id, topic: dt.topic, projectId: demoProject.id } as ThreadConfig,
              projectId: demoProject.id,
              hasChildren: false,
              isDemo: true,
            });
          }
        }
      }
    }

    return nodes;
  }, [fetchedProjects, threadsByProject, expandedProjects, expandedThreads, searchQuery]);

  // --- Drop indicator ---
  const dropIndicator = useMemo(() => {
    if (!activeDragId || !overProjectId || activeDragId === overProjectId) return null;

    const activeIdx = fetchedProjects.findIndex((p) => p.id === activeDragId);
    const overIdx = fetchedProjects.findIndex((p) => p.id === overProjectId);
    if (activeIdx === -1 || overIdx === -1) return null;

    if (activeIdx > overIdx) {
      const nodeIdx = flattenedNodes.findIndex(
        (n) => n.type === "project" && n.project.id === overProjectId,
      );
      return nodeIdx !== -1 ? { nodeIndex: nodeIdx, position: "above" as const } : null;
    }

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

  const activeDragThreadCount = activeProject
    ? (threadsByProject.get(activeProject.id) ?? []).length
    : 0;

  // --- Toggle handlers ---
  const toggleProject = useCallback((id: string) => {
    setExpandedProjects((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const toggleThread = useCallback((id: string) => {
    setExpandedThreads((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const togglePin = useCallback(
    (id: string) => {
      if (archivedThreads.has(id)) return;
      setPinnedThreads((prev) => {
        const next = new Set(prev);
        if (next.has(id)) next.delete(id);
        else next.add(id);
        return next;
      });
    },
    [archivedThreads],
  );

  const toggleArchive = useCallback(
    (id: string) => {
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
    },
    [archivedThreads, archiveThreadMutation],
  );

  const handleAddThread = useCallback(
    (projectId: string) => {
      createThreadMutation.mutate({ projectId });
    },
    [createThreadMutation],
  );

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

  // --- Render ---
  return (
    <div className="flex h-full flex-col select-none">
      <SidebarHeader
        activeView={activeView}
        activeTab={activeTab}
        onTabChange={setActiveTab}
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
      />

      {activeView === "templates" ? (
        <PlanTemplates />
      ) : (
        <ResizablePanelGroup orientation="vertical" className="flex-1">
          <ResizablePanel defaultSize={60} minSize={30}>
            <ThreadTree
              flattenedNodes={flattenedNodes}
              projectIds={projectIds}
              sensors={sensors}
              activeProject={activeProject}
              activeDragThreadCount={activeDragThreadCount}
              dropIndicator={dropIndicator}
              selectedThreadId={selectedThreadId}
              selectedTaskId={selectedTaskId}
              expandedProjects={expandedProjects}
              expandedThreads={expandedThreads}
              pinnedThreads={pinnedThreads}
              archivedThreads={archivedThreads}
              showAddProject={activeTab === "threads"}
              onDragStart={handleDragStart}
              onDragOver={handleDragOver}
              onDragEnd={handleDragEnd}
              onDragCancel={handleDragCancel}
              onToggleProject={toggleProject}
              onToggleThread={toggleThread}
              onTogglePin={togglePin}
              onToggleArchive={toggleArchive}
              onAddThread={handleAddThread}
              onAddProject={() => setNewProjectDialogOpen(true)}
            />
          </ResizablePanel>

          <PanelResizeHandle className="h-px w-full shrink-0 bg-border hover:bg-primary/20 data-[resize-handle-state=drag]:bg-primary/30 cursor-row-resize" />

          <ResizablePanel
            panelRef={inboxPanelRef}
            defaultSize={40}
            minSize="32px"
            collapsible
            collapsedSize="32px"
            onResize={({ asPercentage }) => {
              setIsInboxCollapsed(asPercentage <= 5);
            }}
          >
            <InboxPanel
              isCollapsed={isInboxCollapsed}
              pendingCount={pendingFeedbackCount}
              items={demoInboxItems}
              onToggle={() => {
                if (isInboxCollapsed) {
                  inboxPanelRef.current?.expand();
                } else {
                  inboxPanelRef.current?.collapse();
                }
              }}
              selectedThreadId={selectedThreadId}
              selectedTaskId={selectedTaskId}
            />
          </ResizablePanel>
        </ResizablePanelGroup>
      )}

      <NewProjectDialog
        open={newProjectDialogOpen}
        onOpenChange={setNewProjectDialogOpen}
        onSave={handleCreateProject}
        workers={workersData?.workers ?? []}
      />
    </div>
  );
}
