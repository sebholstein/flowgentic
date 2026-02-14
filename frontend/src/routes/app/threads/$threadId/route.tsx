import { createFileRoute, Outlet, Link, useParams, useRouterState } from "@tanstack/react-router";
import { dragStyle, noDragStyle } from "@/components/layout/WindowDragRegion";
import { useState, useCallback, useMemo, createContext, use, useRef, useEffect } from "react";
import { ReactFlowProvider } from "@xyflow/react";
import "@xyflow/react/dist/style.css";

import { useQuery, useQueryClient, useMutation } from "@tanstack/react-query";
import { cn } from "@/lib/utils";
import { GitBranch, Activity, Workflow, Inbox, Brain, User, Server, Loader2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { threadStatusConfig } from "@/constants/threadStatusConfig";
import { useClient } from "@/lib/connect";
import { ThreadService } from "@/proto/gen/controlplane/v1/thread_service_pb";
import { TaskService } from "@/proto/gen/controlplane/v1/task_service_pb";
import { SessionService } from "@/proto/gen/controlplane/v1/session_service_pb";
import { tasksQueryOptions } from "@/lib/queries/tasks";
import { sessionsQueryOptions } from "@/lib/queries/sessions";
import { ThreadGraphView } from "@/components/threads/ThreadGraphView";
import { TaskDetailSidebar } from "@/components/threads/TaskDetailSidebar";
import { AgentChatPanel, type ChatMessage } from "@/components/chat/AgentChatPanel";
import { useSessionEvents } from "@/hooks/use-session-events";

const availableModels = [
  { id: "claude-opus-4", name: "Claude Opus 4" },
  { id: "claude-sonnet-4", name: "Claude Sonnet 4" },
  { id: "gpt-4", name: "GPT-4" },
  { id: "gpt-4o", name: "GPT-4o" },
  { id: "gemini-pro", name: "Gemini Pro" },
];
import { useInfrastructureStore, selectControlPlaneById } from "@/stores/serverStore";
import { ServerStatusDot } from "@/components/servers/ServerStatusDot";
import type { Thread } from "@/types/thread";
import type { Task } from "@/types/task";
import type { TaskConfig } from "@/proto/gen/controlplane/v1/task_service_pb";

type SearchParams = {
  taskId?: string;
};

type ThreadBootstrapState = {
  initialPrompt: string;
  createdAt: number;
};

export const Route = createFileRoute("/app/threads/$threadId")({
  component: ThreadLayout,
  validateSearch: (search: Record<string, unknown>): SearchParams => {
    return {
      taskId: typeof search.taskId === "string" ? search.taskId : undefined,
    };
  },
});

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

function mapBackendTask(t: TaskConfig): Task {
  return {
    id: t.id,
    name: t.description,
    description: t.description,
    status: (t.status || "pending") as Task["status"],
    subtasks: t.subtasks.map((s, i) => ({
      id: `${t.id}-sub-${i}`,
      name: s,
      completed: false,
    })),
    dependencies: [],
  };
}

function ThreadLayout() {
  const { threadId } = useParams({ from: "/app/threads/$threadId" });
  const routerState = useRouterState();
  const queryClient = useQueryClient();
  const [view, setView] = useState<"page" | "graph">("page");
  const [selectedTaskId, setSelectedTaskId] = useState<string | undefined>(undefined);
  const [leftPanelPercent, setLeftPanelPercent] = useState(50);
  const containerRef = useRef<HTMLDivElement>(null);

  const threadClient = useClient(ThreadService);
  const { data: threadData, isLoading } = useQuery({
    queryKey: ["thread", threadId],
    queryFn: () => threadClient.getThread({ id: threadId }),
  });

  const thread = useMemo<Thread | undefined>(() => {
    const t = threadData?.thread;
    if (!t) return undefined;
    return {
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
      mode: t.mode === "plan" || t.mode === "build" ? t.mode : "build",
      model: t.model,
      harness: t.agent,
      plan: t.plan || undefined,
    };
  }, [threadData]);

  // Task and session queries
  const taskClient = useClient(TaskService);
  const sessionClient = useClient(SessionService);
  const { data: tasksData } = useQuery(tasksQueryOptions(taskClient, threadId));
  const { data: sessionsData } = useQuery(sessionsQueryOptions(sessionClient, threadId));

  const tasks = useMemo<Task[]>(() => {
    return (tasksData?.tasks ?? []).map(mapBackendTask);
  }, [tasksData]);

  const controlPlane = useInfrastructureStore((s) =>
    thread?.controlPlaneId ? selectControlPlaneById(s, thread.controlPlaneId) : undefined,
  );
  const [threadMode, setThreadMode] = useState<"plan" | "build">("plan");
  const [threadModel, setThreadModel] = useState("claude-opus-4");
  const isBuildMode = threadMode === "build";

  // Stream session events for the thread chat
  const {
    messages: sessionMessages,
    pendingAgentText,
    pendingThoughtText,
    isResponding: isSessionResponding,
    hasReceivedUpdate: hasReceivedSessionUpdate,
  } = useSessionEvents({ threadId });

  const bootstrapState = queryClient.getQueryData<ThreadBootstrapState>([
    "thread-bootstrap",
    threadId,
  ]);

  const hasRealUserMessage = useMemo(
    () => sessionMessages.some((message) => message.type === "user"),
    [sessionMessages],
  );

  const displayMessages = useMemo<ChatMessage[]>(() => {
    if (!bootstrapState || hasRealUserMessage) {
      return sessionMessages;
    }

    const bootstrapMessage: ChatMessage = {
      id: `bootstrap-user-${threadId}`,
      type: "user",
      content: bootstrapState.initialPrompt,
      timestamp: new Date(bootstrapState.createdAt).toISOString(),
    };

    return [bootstrapMessage, ...sessionMessages];
  }, [bootstrapState, hasRealUserMessage, sessionMessages, threadId]);

  useEffect(() => {
    if (bootstrapState && hasRealUserMessage) {
      queryClient.removeQueries({ queryKey: ["thread-bootstrap", threadId], exact: true });
    }
  }, [bootstrapState, hasRealUserMessage, queryClient, threadId]);

  const [hasPrimingWindowExpired, setHasPrimingWindowExpired] = useState(false);
  useEffect(() => {
    setHasPrimingWindowExpired(false);
    const timer = window.setTimeout(() => {
      setHasPrimingWindowExpired(true);
    }, 1200);
    return () => window.clearTimeout(timer);
  }, [threadId]);

  const isAwaitingBootstrapResponse =
    Boolean(bootstrapState) &&
    sessionMessages.length === 0 &&
    !pendingAgentText &&
    !pendingThoughtText;
  const hasAnySession = (sessionsData?.sessions?.length ?? 0) > 0;
  const isPrimingExistingSession =
    hasAnySession && !hasReceivedSessionUpdate && !hasPrimingWindowExpired;
  const isPanelStreaming =
    isSessionResponding || isAwaitingBootstrapResponse || isPrimingExistingSession;

  // Follow-up message mutation
  const promptMutation = useMutation({
    mutationFn: (text: string) => sessionClient.promptSession({ threadId, text }),
  });

  const handleSendFollowUp = useCallback(
    (text: string) => {
      promptMutation.mutate(text);
    },
    [promptMutation],
  );

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

  const pendingCheckInsCount = 0;

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center text-muted-foreground">
        <Loader2 className="size-5 animate-spin" />
      </div>
    );
  }

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
              style={isBuildMode ? undefined : { width: `${leftPanelPercent}%` }}
            >
              <Badge
                variant="outline"
                className="text-[10px] px-1.5 py-0 h-5 shrink-0 font-medium text-violet-400 border-violet-500/30"
              >
                Thread
              </Badge>
              {isBuildMode && (
                <Badge
                  variant="outline"
                  className="text-[10px] px-1.5 py-0 h-5 shrink-0 font-medium text-slate-400 border-slate-500/30 gap-0.5"
                >
                  <User className="size-2.5" />
                  Build
                </Badge>
              )}
              <StatusIcon
                className={cn("size-4 shrink-0", threadStatusConfig[thread.status].color)}
              />
              <span className="font-medium text-sm truncate">{thread.topic}</span>
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

          {/* Right section - tab navigation (hidden for build mode) */}
          {!isBuildMode && (
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
          {isBuildMode && thread.memory && (
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
          {/* Build mode - full-width chat */}
          {isBuildMode && !isTaskDetailPage ? (
            <div className="flex-1 h-full overflow-hidden">
              <AgentChatPanel
                target={{
                  type: "thread_overseer",
                  entityId: thread.id,
                  agentName: threadModel
                    ? (availableModels.find((m) => m.id === threadModel)?.name ?? "Agent")
                    : "Agent",
                  title: thread.topic,
                  agentColor: "bg-violet-500",
                }}
                hideHeader
                showSetupOnEmpty={false}
                externalMessages={displayMessages}
                pendingAgentText={pendingAgentText}
                pendingThoughtText={pendingThoughtText}
                isStreaming={isPanelStreaming}
                onSend={handleSendFollowUp}
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
                        title: thread.topic,
                        agentColor: "bg-violet-500",
                      }}
                      showSetupOnEmpty={false}
                      externalMessages={displayMessages}
                      pendingAgentText={pendingAgentText}
                      pendingThoughtText={pendingThoughtText}
                      isStreaming={isPanelStreaming}
                      onSend={handleSendFollowUp}
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
                        isSimulating={false}
                        onStart={() => {}}
                        onPause={() => {}}
                        onReset={() => {}}
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
