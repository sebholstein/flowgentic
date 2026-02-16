import { createFileRoute, Outlet, Link, useParams, useRouterState } from "@tanstack/react-router";
import { dragStyle, noDragStyle } from "@/components/layout/WindowDragRegion";
import { useState, useCallback, useMemo, createContext, use, useRef, useEffect } from "react";
import { ReactFlowProvider } from "@xyflow/react";
import "@xyflow/react/dist/style.css";

import { useQuery, useQueryClient, useMutation } from "@tanstack/react-query";
import { cn } from "@/lib/utils";
import { GitBranch, Activity, Workflow, Inbox, Loader2, FileCode2, Brain, Plus, Bot } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { getHarnessIcon } from "@/components/icons/agent-icons";
import { useClient } from "@/lib/connect";
import { ThreadService } from "@/proto/gen/controlplane/v1/thread_service_pb";
import type { ThreadConfig } from "@/proto/gen/controlplane/v1/thread_service_pb";
import { TaskService } from "@/proto/gen/controlplane/v1/task_service_pb";
import { SessionService } from "@/proto/gen/controlplane/v1/session_service_pb";
import { tasksQueryOptions } from "@/lib/queries/tasks";
import { sessionsQueryOptions } from "@/lib/queries/sessions";
import { ThreadGraphView } from "@/components/threads/ThreadGraphView";
import { TaskDetailSidebar } from "@/components/threads/TaskDetailSidebar";
import { AgentChatPanel, type ChatMessage } from "@/components/chat/AgentChatPanel";
import { ThreadSetupForm } from "@/components/ui/ThreadSetupForm";
import { useSessionEvents } from "@/hooks/use-session-events";
import { ProjectService } from "@/proto/gen/controlplane/v1/project_service_pb";
import { WorkerService } from "@/proto/gen/controlplane/v1/worker_service_pb";
import { SystemService } from "@/proto/gen/worker/v1/system_service_pb";
import { Agent, AgentSchema } from "@/proto/gen/worker/v1/agent_pb";
import { projectsQueryOptions } from "@/lib/queries/projects";
import { workersQueryOptions } from "@/lib/queries/workers";
import type { Project } from "@/types/project";
import type { Worker } from "@/types/server";
import { useResizePanel } from "@/hooks/use-resize-panel";
import { deriveThreadViewState, parsePlan, type ThreadViewState, type ParsedPlan } from "@/lib/thread-state";
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
export interface ThreadContextValue {
  thread: ThreadConfig;
  tasks: Task[];
  selectedTaskId?: string;
  onSelectTask: (taskId: string) => void;
  pendingCheckInsCount: number;
  viewState: ThreadViewState;
  parsedPlan: ParsedPlan | null;
}

export const ThreadContext = createContext<ThreadContextValue | null>(null);

export function useThreadContext() {
  const context = use(ThreadContext);
  if (!context) {
    throw new Error("useThreadContext must be used within ThreadLayout");
  }
  return context;
}

/** Normalize agent ID strings like "CLAUDE_CODE" → "claude-code" */
function normalizeAgentId(agent: string): string {
  return agent.toLowerCase().replace(/_/g, "-");
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
  const { percent: leftPanelPercent, handleMouseDown, containerRef } = useResizePanel({
    initial: 60,
    min: 30,
    max: 70,
  });

  const [activeSessionId, setActiveSessionId] = useState<string | undefined>(undefined);
  const [newSessionPopoverOpen, setNewSessionPopoverOpen] = useState(false);

  const threadClient = useClient(ThreadService);
  const { data: threadData, isLoading } = useQuery({
    queryKey: ["thread", threadId],
    queryFn: () => threadClient.getThread({ id: threadId }),
  });

  const [showLoading, setShowLoading] = useState(false);
  useEffect(() => {
    if (!isLoading) {
      setShowLoading(false);
      return;
    }
    const timer = setTimeout(() => setShowLoading(true), 200);
    return () => clearTimeout(timer);
  }, [isLoading]);

  const thread = threadData?.thread;

  // Task and session queries
  const taskClient = useClient(TaskService);
  const sessionClient = useClient(SessionService);
  const { data: tasksData } = useQuery(tasksQueryOptions(taskClient, threadId));
  const { data: sessionsData } = useQuery(sessionsQueryOptions(sessionClient, threadId));

  const tasks = useMemo<Task[]>(() => {
    return (tasksData?.tasks ?? []).map(mapBackendTask);
  }, [tasksData]);

  const hasAnySession = (sessionsData?.sessions?.length ?? 0) > 0;
  const sessions = sessionsData?.sessions ?? [];

  // Default active session to first session
  useEffect(() => {
    if (sessions.length > 0 && !activeSessionId) {
      setActiveSessionId(sessions[0].id);
    }
  }, [sessions, activeSessionId]);

  // Derive view state from thread data
  const viewState = deriveThreadViewState(thread, tasks);
  const threadParsedPlan = thread?.plan ? parsePlan(thread.plan) : null;

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
  const isPrimingExistingSession =
    hasAnySession && !hasReceivedSessionUpdate && !hasPrimingWindowExpired;
  const isPanelStreaming =
    isSessionResponding || isAwaitingBootstrapResponse || isPrimingExistingSession;

  // Hide chat panel while priming an existing session with no content yet to avoid flicker
  const isChatReady =
    !isPrimingExistingSession || sessionMessages.length > 0 || !!pendingAgentText || !!pendingThoughtText;

  // --- Session setup state (shown when no sessions exist) ---
  const projectClient = useClient(ProjectService);
  const workerClient = useClient(WorkerService);
  const systemClient = useClient(SystemService);
  const { data: projectsData } = useQuery(projectsQueryOptions(projectClient));
  const { data: workersData } = useQuery(workersQueryOptions(workerClient));

  const projects = useMemo<Project[]>(
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

  const workers = useMemo<Worker[]>(
    () =>
      (workersData?.workers ?? []).map((w) => ({
        id: w.id,
        name: w.name,
        type: "remote" as const,
        url: w.url,
        status: "connected" as const,
        controlPlaneId: "",
      })),
    [workersData],
  );

  const [agent, setAgent] = useState<Agent>(Agent.CLAUDE_CODE);
  const [workerId, setWorkerId] = useState("");
  const [sessionMode, setSessionMode] = useState("code");
  const [threadModel, setThreadModel] = useState("");

  const {
    data: modelsData,
    isLoading: modelsLoading,
    isError: modelsIsError,
  } = useQuery({
    queryKey: ["agent-models", workerId, agent],
    queryFn: () =>
      systemClient.getAgentModels(
        { agent, disableCache: false },
        { headers: { "X-Worker-Id": workerId } },
      ),
    enabled: !!workerId,
    retry: false,
  });

  useEffect(() => {
    if (!modelsData) return;
    const models = modelsData.models;
    const fallbackModel = modelsData.defaultModel || models[0]?.id || "";
    if (!fallbackModel) return;
    const modelIds = models.map((m) => m.id);
    if (!threadModel || !modelIds.includes(threadModel)) {
      setThreadModel(fallbackModel);
    }
  }, [modelsData, threadModel]);

  // Set initial worker
  const initialWorkerSet = useRef(false);
  if (!initialWorkerSet.current && workers.length > 0 && !workerId) {
    initialWorkerSet.current = true;
    setWorkerId(workers[0].id);
  }

  // Reset mutation state when navigating between threads (component re-renders
  // but doesn't unmount, so stale isSuccess from a previous thread would cause
  // handleSendMessage to route through promptMutation instead of createSession).
  const prevThreadIdRef = useRef(threadId);
  const createSessionMutation = useMutation({
    mutationFn: (prompt: string) =>
      sessionClient.createSession({
        threadId,
        workerId,
        prompt,
        agent: AgentSchema.value[agent].name,
        model: threadModel,
        mode: thread?.mode ?? "plan",
        sessionMode,
      }),
    onSuccess: (_resp, prompt) => {
      const bootstrapData: ThreadBootstrapState = {
        initialPrompt: prompt,
        createdAt: Date.now(),
      };
      queryClient.setQueryData(["thread-bootstrap", threadId], bootstrapData);
      queryClient.invalidateQueries({ queryKey: ["sessions", threadId] });
      queryClient.invalidateQueries({ queryKey: ["thread", threadId] });
    },
  });

  if (prevThreadIdRef.current !== threadId) {
    prevThreadIdRef.current = threadId;
    createSessionMutation.reset();
  }

  // Follow-up message mutation
  const promptMutation = useMutation({
    mutationFn: (text: string) => sessionClient.promptSession({ threadId, text }),
  });

  const handleSendMessage = useCallback(
    (text: string) => {
      if (hasAnySession || createSessionMutation.isPending || createSessionMutation.isSuccess) {
        promptMutation.mutate(text);
      } else {
        createSessionMutation.mutate(text);
      }
    },
    [hasAnySession, createSessionMutation, promptMutation],
  );

  const selectedTask = selectedTaskId ? tasks.find((t) => t.id === selectedTaskId) : undefined;

  const handleSelectTask = useCallback((taskId: string) => {
    setSelectedTaskId(taskId);
  }, []);

  const handleCloseTaskDetail = useCallback(() => {
    setSelectedTaskId(undefined);
  }, []);

  const pendingCheckInsCount = 0;

  if (isLoading && showLoading) {
    return (
      <div className="flex h-full items-center justify-center text-muted-foreground">
        <Loader2 className="size-5 animate-spin" />
      </div>
    );
  }

  if (isLoading) {
    return null;
  }

  if (!thread) {
    return (
      <div className="flex h-full items-center justify-center text-muted-foreground">
        Thread not found
      </div>
    );
  }

  // Determine active tab from current route
  const currentPath = routerState.location.pathname;
  const getActiveTab = () => {
    if (currentPath.includes("/tasks")) return "tasks";
    if (currentPath.includes("/checkins")) return "checkins";
    if (currentPath.includes("/memory")) return "memory";
    if (currentPath.includes("/changes")) return "changes";
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
    viewState,
    parsedPlan: threadParsedPlan,
  };

  const setupFormContent = !hasAnySession ? (
    <ThreadSetupForm
      threadModel={threadModel}
      onModelChange={setThreadModel}
      project={projects.find((p) => p.id === thread.projectId)}
      projects={projects}
      onProjectChange={() => {}}
      workerId={workerId}
      workers={workers}
      onWorkerChange={setWorkerId}
      agent={agent}
      onAgentChange={setAgent}
      availableModels={modelsData?.models ?? []}
      defaultModel={modelsData?.defaultModel ?? ""}
      modelsLoading={modelsLoading}
      modelsError={modelsIsError ? "Could not load models for this agent" : null}
    />
  ) : undefined;

  const chatPanel = (
    <AgentChatPanel
      target={{
        type: "thread_overseer",
        entityId: thread.id,
        agentName: "Agent",
        title: thread.topic || "Untitled",
        agentColor: "bg-violet-500",
      }}
      hideHeader
      emptyStateContent={setupFormContent}
      externalMessages={displayMessages}
      pendingAgentText={pendingAgentText}
      pendingThoughtText={pendingThoughtText}
      isStreaming={isPanelStreaming}
      onSend={handleSendMessage}
      selectedModel={threadModel}
      availableModels={modelsData?.models ?? []}
      modelsLoading={modelsLoading}
      onModelChange={setThreadModel}
      sessionMode={sessionMode}
      onSessionModeChange={setSessionMode}
    />
  );

  // --- Always render split layout: chat (left) + tabbed content (right) ---
  return (
    <ThreadContext value={contextValue}>
      <div className="flex h-full flex-col">
        {/* Header with tabs */}
        <div className="flex border-b h-10 shrink-0 select-none" style={dragStyle}>
          {/* Left section - thread title */}
          {!isTaskDetailPage && (
            <div
              className="flex items-center gap-2 px-4 shrink-0"
              style={{ width: `${leftPanelPercent}%` }}
            >
              <Badge
                variant="outline"
                className="text-[10px] px-1.5 py-0 h-5 shrink-0 font-medium text-violet-400 border-violet-500/30"
              >
                Thread
              </Badge>
              <span className="font-medium text-sm truncate">{thread.topic || "Untitled"}</span>
            </div>
          )}

          {/* Right section - tab navigation */}
          <div className="flex-1 min-w-0 flex items-center px-4">
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
              <Link
                to="/app/threads/$threadId/changes"
                params={{ threadId }}
                className={cn(tabTriggerClass)}
                data-active={activeTab === "changes" && view === "page"}
                onClick={() => setView("page")}
              >
                <FileCode2 className="size-3.5" />
                Changes
              </Link>
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
        </div>

        {/* Session tab strip - below the header, aligned with chat panel */}
        {!isTaskDetailPage && (sessions.length > 0 || hasAnySession) && (
          <div className="flex shrink-0 select-none border-b" style={dragStyle}>
            <div
              className="flex items-center gap-0.5 px-2 h-9 shrink-0"
              style={{ width: `${leftPanelPercent}%` }}
            >
              {sessions.map((session) => {
                const IconComponent = getHarnessIcon(normalizeAgentId(session.agent)) ?? Bot;
                const isActive = session.id === activeSessionId;
                return (
                  <button
                    key={session.id}
                    type="button"
                    className={cn(
                      "h-full flex items-center gap-1.5 px-2 border-b-2 transition-colors cursor-pointer text-xs",
                      isActive
                        ? "border-primary text-foreground"
                        : "border-transparent text-muted-foreground hover:text-foreground",
                    )}
                    style={noDragStyle}
                    onClick={() => setActiveSessionId(session.id)}
                    title={`${session.agent} — ${session.model || "default"}`}
                  >
                    <IconComponent className="size-3" />
                    <span className="truncate max-w-28">{session.prompt?.slice(0, 30) || session.agent}</span>
                  </button>
                );
              })}
              <Popover open={newSessionPopoverOpen} onOpenChange={setNewSessionPopoverOpen}>
                <PopoverTrigger asChild>
                  <button
                    type="button"
                    className="h-full flex items-center px-2 border-b-2 border-transparent text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
                    style={noDragStyle}
                    title="New session"
                  >
                    <Plus className="size-3.5" />
                  </button>
                </PopoverTrigger>
                <PopoverContent className="w-72 p-4" align="start">
                  <ThreadSetupForm
                    compact
                    threadModel={threadModel}
                    onModelChange={setThreadModel}
                    project={projects.find((p) => p.id === thread.projectId)}
                    projects={projects}
                    onProjectChange={() => {}}
                    workerId={workerId}
                    workers={workers}
                    onWorkerChange={setWorkerId}
                    agent={agent}
                    onAgentChange={setAgent}
                    availableModels={modelsData?.models ?? []}
                    defaultModel={modelsData?.defaultModel ?? ""}
                    modelsLoading={modelsLoading}
                    modelsError={modelsIsError ? "Could not load models for this agent" : null}
                    submitLabel="Create Session"
                    onSubmit={() => setNewSessionPopoverOpen(false)}
                  />
                </PopoverContent>
              </Popover>
            </div>
          </div>
        )}

        <div className="flex flex-1 min-h-0" ref={containerRef}>
          {/* Chat panel - visible on the left, hidden on task detail pages */}
          {!isTaskDetailPage && (
            <>
              <div
                className="flex-shrink-0 h-full overflow-hidden"
                style={{ width: `${leftPanelPercent}%` }}
              >
                {isChatReady && chatPanel}
              </div>
              {/* Resize handle */}
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
        </div>
      </div>
    </ThreadContext>
  );
}
