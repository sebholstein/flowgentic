import {
  createFileRoute,
  Outlet,
  Link,
  useParams,
  useRouterState,
  useSearch,
} from "@tanstack/react-router";
import { dragStyle, noDragStyle } from "@/components/layout/WindowDragRegion";
import { useState, useCallback, useMemo, createContext, use, useRef } from "react";
import { ReactFlowProvider } from "@xyflow/react";
import "@xyflow/react/dist/style.css";

import { cn } from "@/lib/utils";
import { GitBranch, Activity, Workflow, Inbox, Brain, User, Server } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { threads } from "@/data/mockThreadsData";
import { threadTasks } from "@/data/mockTasksData";
import { inboxItems } from "@/data/mockInboxData";
import { threadStatusConfig } from "@/constants/threadStatusConfig";
import { ThreadGraphView } from "@/components/threads/ThreadGraphView";
import { TaskDetailSidebar } from "@/components/threads/TaskDetailSidebar";
import { AgentChatPanel } from "@/components/chat/AgentChatPanel";

const availableModels = [
  { id: "claude-opus-4", name: "Claude Opus 4" },
  { id: "claude-sonnet-4", name: "Claude Sonnet 4" },
  { id: "gpt-4", name: "GPT-4" },
  { id: "gpt-4o", name: "GPT-4o" },
  { id: "gemini-pro", name: "Gemini Pro" },
];
import { useTaskSimulation } from "@/hooks/useTaskSimulation";
import { useInfrastructureStore, selectControlPlaneById } from "@/stores/serverStore";
import { ServerStatusDot } from "@/components/servers/ServerStatusDot";
import type { Thread } from "@/types/thread";
import type { Task } from "@/types/task";
import type { InboxItem, InboxItemPriority } from "@/types/inbox";

type SearchParams = {
  taskId?: string;
  feedback?: string;
};

export const Route = createFileRoute("/app/threads/$threadId")({
  component: ThreadLayout,
  validateSearch: (search: Record<string, unknown>): SearchParams => {
    return {
      taskId: typeof search.taskId === "string" ? search.taskId : undefined,
      feedback: typeof search.feedback === "string" ? search.feedback : undefined,
    };
  },
});

// Priority order for sorting feedback items
const priorityOrder: Record<InboxItemPriority, number> = {
  high: 3,
  medium: 2,
  low: 1,
};

// Context for sharing thread data with child routes
interface ThreadContextValue {
  thread: Thread;
  tasks: Task[];
  selectedTaskId?: string;
  onSelectTask: (taskId: string) => void;
  pendingCheckInsCount: number;
}

export const ThreadContext = createContext<ThreadContextValue | null>(null);

export function useThreadContext() {
  const context = use(ThreadContext);
  if (!context) {
    throw new Error("useThreadContext must be used within ThreadLayout");
  }
  return context;
}

function ThreadLayout() {
  const { threadId } = useParams({ from: "/app/threads/$threadId" });
  const search = useSearch({ from: "/app/threads/$threadId" });
  const routerState = useRouterState();
  const [view, setView] = useState<"page" | "graph">("page");
  const [selectedTaskId, setSelectedTaskId] = useState<string | undefined>(undefined);
  const [leftPanelPercent, setLeftPanelPercent] = useState(50);
  const containerRef = useRef<HTMLDivElement>(null);

  const thread = threads.find((s) => s.id === threadId);
  const controlPlane = useInfrastructureStore((s) =>
    thread?.controlPlaneId ? selectControlPlaneById(s, thread.controlPlaneId) : undefined,
  );
  const [threadMode, setThreadMode] = useState<"single_agent" | "orchestrated">(
    thread?.mode ?? "orchestrated",
  );
  const [threadModel, setThreadModel] = useState(thread?.model ?? "claude-opus-4");
  const initialTasks = useMemo(() => threadTasks[threadId] ?? [], [threadId]);

  const isSingleAgent = threadMode === "single_agent";

  // Find pending feedback for this thread
  const pendingFeedback = useMemo<InboxItem | null>(() => {
    // Get all pending items for this thread
    const threadItems = inboxItems.filter(
      (item) => item.threadId === threadId && item.status === "pending",
    );

    if (threadItems.length === 0) return null;

    // If specific feedback requested via search param, prioritize that
    if (search.feedback) {
      const requested = threadItems.find((item) => item.id === search.feedback);
      if (requested) return requested;
    }

    // Otherwise return highest priority pending item
    return (
      threadItems.sort((a, b) => priorityOrder[b.priority] - priorityOrder[a.priority])[0] ?? null
    );
  }, [threadId, search.feedback]);

  // Handle feedback submission
  const handleFeedbackSubmit = useCallback((itemId: string, data: unknown) => {
    // In a real app, this would update the inbox item status via API
    console.log("Feedback submitted:", { itemId, data });
    // For now, we just log it - the actual state update would happen via a mutation
  }, []);

  const { tasks, isSimulating, startSimulation, pauseSimulation, resetTasks } =
    useTaskSimulation(initialTasks);

  const selectedTask = selectedTaskId ? tasks.find((t) => t.id === selectedTaskId) : undefined;

  const handleSelectTask = useCallback((taskId: string) => {
    setSelectedTaskId(taskId);
  }, []);

  const handleCloseTaskDetail = useCallback(() => {
    setSelectedTaskId(undefined);
  }, []);

  // Resize handle for left panel (chat)
  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      const startX = e.clientX;
      const startPercent = leftPanelPercent;
      const containerWidth = containerRef.current?.offsetWidth ?? 1;

      const handleMouseMove = (moveEvent: MouseEvent) => {
        const deltaX = moveEvent.clientX - startX;
        const deltaPercent = (deltaX / containerWidth) * 100;
        // Clamp between 30% and 70%
        setLeftPanelPercent(Math.min(70, Math.max(30, startPercent + deltaPercent)));
      };

      const handleMouseUp = () => {
        document.removeEventListener("mousemove", handleMouseMove);
        document.removeEventListener("mouseup", handleMouseUp);
      };

      document.addEventListener("mousemove", handleMouseMove);
      document.addEventListener("mouseup", handleMouseUp);
    },
    [leftPanelPercent],
  );

  // Get tasks with pending check-ins
  const pendingCheckInsCount = useMemo(() => {
    return tasks.filter((task) => task.checkIn).length;
  }, [tasks]);

  if (!thread) {
    return (
      <div className="flex h-full items-center justify-center text-muted-foreground">
        Thread not found
      </div>
    );
  }

  const StatusIcon = threadStatusConfig[thread.status].icon;

  // Determine active tab from current route
  const currentPath = routerState.location.pathname;
  const getActiveTab = () => {
    if (currentPath.includes("/tasks")) return "tasks";
    if (currentPath.includes("/checkins")) return "checkins";
    if (currentPath.includes("/memory")) return "memory";
    return "overview";
  };
  const activeTab = getActiveTab();

  // Check if we're on a task detail page (hide chat in that case)
  const isTaskDetailPage = /\/tasks\/[^/]+/.test(currentPath);

  const tabTriggerClass =
    "h-10 flex items-center rounded-none border-b-2 border-transparent data-[active=true]:border-primary px-0 gap-1.5 text-xs text-muted-foreground hover:text-foreground data-[active=true]:text-foreground transition-colors cursor-pointer";

  const contextValue: ThreadContextValue = {
    thread,
    tasks,
    selectedTaskId,
    onSelectTask: handleSelectTask,
    pendingCheckInsCount,
  };

  return (
    <ThreadContext value={contextValue}>
      <div className="flex h-full flex-col">
        {/* Compact Header with integrated tabs - aligned with panels below */}
        <div className="flex border-b h-10 shrink-0 select-none" style={dragStyle}>
          {/* Left section - matches chat panel width */}
          {!isTaskDetailPage && (
            <div
              className="flex items-center gap-2 px-4 shrink-0"
              style={isSingleAgent ? undefined : { width: `${leftPanelPercent}%` }}
            >
              <Badge
                variant="outline"
                className="text-[10px] px-1.5 py-0 h-5 shrink-0 font-medium text-violet-400 border-violet-500/30"
              >
                Thread
              </Badge>
              {isSingleAgent && (
                <Badge
                  variant="outline"
                  className="text-[10px] px-1.5 py-0 h-5 shrink-0 font-medium text-slate-400 border-slate-500/30 gap-0.5"
                >
                  <User className="size-2.5" />
                  Single Agent
                </Badge>
              )}
              <StatusIcon
                className={cn("size-4 shrink-0", threadStatusConfig[thread.status].color)}
              />
              <span className="text-xs text-muted-foreground">#{thread.id}</span>
              <span className="font-medium text-sm truncate">{thread.title}</span>
              {controlPlane && (
                <Badge
                  variant="outline"
                  className="text-[10px] px-1.5 py-0 h-5 shrink-0 font-medium text-muted-foreground border-border gap-1"
                >
                  <Server className="size-2.5" />
                  {controlPlane.name}
                  <ServerStatusDot status={controlPlane.status} />
                </Badge>
              )}
            </div>
          )}

          {/* Right section - tab navigation (hidden for single_agent mode) */}
          {!isSingleAgent && (
            <div className="flex-1 min-w-0 flex items-center px-4">
              {/* Tab Navigation */}
              <nav className="flex items-center gap-4 h-full" style={noDragStyle}>
                <Link
                  to="/app/threads/$threadId"
                  params={{ threadId }}
                  className={cn(tabTriggerClass)}
                  data-active={activeTab === "overview" && view === "page"}
                  onClick={() => setView("page")}
                >
                  <Activity className="size-3.5" />
                  Overview
                </Link>
                <Link
                  to="/app/threads/$threadId/tasks"
                  params={{ threadId }}
                  className={cn(tabTriggerClass)}
                  data-active={activeTab === "tasks" && view === "page"}
                  onClick={() => setView("page")}
                >
                  <Workflow className="size-3.5" />
                  Tasks
                  <span className="text-muted-foreground">({tasks.length})</span>
                </Link>
                {pendingCheckInsCount > 0 && (
                  <Link
                    to="/app/threads/$threadId/checkins"
                    params={{ threadId }}
                    className={cn(tabTriggerClass)}
                    data-active={activeTab === "checkins" && view === "page"}
                    onClick={() => setView("page")}
                  >
                    <Inbox className="size-3.5" />
                    Check-ins
                    <Badge
                      variant="outline"
                      className="ml-1 text-xs px-1.5 py-0 h-5 text-amber-400 border-amber-500/30"
                    >
                      {pendingCheckInsCount}
                    </Badge>
                  </Link>
                )}
                {thread.memory && (
                  <Link
                    to="/app/threads/$threadId/memory"
                    params={{ threadId }}
                    className={cn(tabTriggerClass)}
                    data-active={activeTab === "memory" && view === "page"}
                    onClick={() => setView("page")}
                  >
                    <Brain className="size-3.5" />
                    Memory
                  </Link>
                )}
                <button
                  type="button"
                  className={cn(tabTriggerClass)}
                  data-active={view === "graph"}
                  onClick={() => setView("graph")}
                >
                  <GitBranch className="size-3.5" />
                  Graph
                </button>
              </nav>
            </div>
          )}
          {/* Memory icon for single agent mode */}
          {isSingleAgent && thread.memory && (
            <div className="flex-1 min-w-0 flex items-center justify-end px-4" style={noDragStyle}>
              <Link
                to="/app/threads/$threadId/memory"
                params={{ threadId }}
                className={cn(tabTriggerClass)}
                data-active={activeTab === "memory"}
                onClick={() => setView("page")}
              >
                <Brain className="size-3.5" />
                Memory
              </Link>
            </div>
          )}
        </div>

        <div className="flex flex-1 min-h-0" ref={containerRef}>
          {/* Single Agent mode - full-width chat */}
          {isSingleAgent && !isTaskDetailPage ? (
            <div className="flex-1 h-full overflow-hidden">
              <AgentChatPanel
                target={{
                  type: "thread_overseer",
                  entityId: thread.id,
                  agentName: threadModel
                    ? (availableModels.find((m) => m.id === threadModel)?.name ?? "Agent")
                    : "Agent",
                  title: thread.title,
                  agentColor: "bg-violet-500",
                }}
                hideHeader
                enableSimulation
                pendingFeedback={pendingFeedback}
                onFeedbackSubmit={handleFeedbackSubmit}
                threadMode={threadMode}
                onModeChange={setThreadMode}
                threadModel={threadModel}
                onModelChange={setThreadModel}
              />
            </div>
          ) : (
            <>
              {/* Chat panel - visible on the left, hidden on task detail pages */}
              {!isTaskDetailPage && (
                <>
                  <div
                    className="flex-shrink-0 h-full overflow-hidden"
                    style={{ width: `${leftPanelPercent}%` }}
                  >
                    <AgentChatPanel
                      target={{
                        type: "thread_overseer",
                        entityId: thread.id,
                        agentName: "Overseer",
                        title: thread.title,
                        agentColor: "bg-violet-500",
                      }}
                      pendingFeedback={pendingFeedback}
                      onFeedbackSubmit={handleFeedbackSubmit}
                      threadMode={threadMode}
                      onModeChange={setThreadMode}
                      threadModel={threadModel}
                      onModelChange={setThreadModel}
                    />
                  </div>
                  {/* Resize handle - wide hit area, thin visual line */}
                  <div
                    className="w-3 -ml-[6px] -mr-[5px] flex-shrink-0 cursor-col-resize flex justify-center group relative z-10"
                    onMouseDown={handleMouseDown}
                  >
                    <div className="w-px h-full bg-border group-hover:bg-primary/30 transition-colors" />
                  </div>
                </>
              )}

              {/* Main content area */}
              <div className="min-w-0 flex-1 overflow-hidden">
                {view === "page" ? (
                  <div className="h-full flex flex-col overflow-auto">
                    {/* Tab Content via Outlet */}
                    <Outlet />
                  </div>
                ) : (
                  <div key={threadId} className="h-full bg-slate-100 relative">
                    <ReactFlowProvider>
                      <ThreadGraphView
                        tasks={tasks}
                        selectedTaskId={selectedTaskId}
                        isSimulating={isSimulating}
                        onStart={startSimulation}
                        onPause={pauseSimulation}
                        onReset={resetTasks}
                        onNodeClick={handleSelectTask}
                      />
                    </ReactFlowProvider>
                    {/* Task detail sidebar for graph view */}
                    {selectedTask && (
                      <div className="absolute right-0 top-0 bottom-0 w-72 border-l bg-sidebar">
                        <TaskDetailSidebar
                          task={selectedTask}
                          tasks={tasks}
                          threadId={threadId}
                          onClose={handleCloseTaskDetail}
                          onSelectTask={handleSelectTask}
                        />
                      </div>
                    )}
                  </div>
                )}
              </div>
            </>
          )}
        </div>
      </div>
    </ThreadContext>
  );
}
