import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState, useCallback, useRef, useMemo, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { MessageSquarePlus } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { dragStyle } from "@/components/layout/WindowDragRegion";
import { AgentChatPanel } from "@/components/chat/AgentChatPanel";
import { ThreadSetupForm } from "@/components/ui/ThreadSetupForm";
import { useClient } from "@/lib/connect";
import { ProjectService } from "@/proto/gen/controlplane/v1/project_service_pb";
import { WorkerService } from "@/proto/gen/controlplane/v1/worker_service_pb";
import { ThreadService } from "@/proto/gen/controlplane/v1/thread_service_pb";
import { SystemService } from "@/proto/gen/worker/v1/system_service_pb";
import { Agent, AgentSchema } from "@/proto/gen/worker/v1/agent_pb";
import { projectsQueryOptions } from "@/lib/queries/projects";
import { workersQueryOptions } from "@/lib/queries/workers";
import type { Project } from "@/types/project";
import type { Worker } from "@/types/server";

type SearchParams = {
  projectId?: string;
};

type ThreadBootstrapState = {
  initialPrompt: string;
  createdAt: number;
};

export const Route = createFileRoute("/app/threads/new")({
  component: NewThreadPage,
  validateSearch: (search: Record<string, unknown>): SearchParams => ({
    projectId: typeof search.projectId === "string" ? search.projectId : undefined,
  }),
});

function NewThreadPage() {
  const { projectId } = Route.useSearch();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const projectClient = useClient(ProjectService);
  const workerClient = useClient(WorkerService);
  const threadClient = useClient(ThreadService);
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

  const currentProject = projects.find((p) => p.id === projectId) ?? projects[0];
  const [threadMode, setThreadMode] = useState<"plan" | "build">("plan");
  const [threadModel, setThreadModel] = useState("");

  const [agent, setAgent] = useState<Agent>(Agent.CLAUDE_CODE);
  const [workerId, setWorkerId] = useState("");
  const [sessionMode, setSessionMode] = useState("code");

  const {
    data: modelsData,
    isLoading: modelsLoading,
    isError: modelsIsError,
  } = useQuery({
    queryKey: ["agent-models", workerId, agent],
    queryFn: () =>
      systemClient.getAgentModels(
        {
          agent,
          disableCache: false,
        },
        {
          headers: {
            "X-Worker-Id": workerId,
          },
        },
      ),
    enabled: !!workerId,
    retry: false,
  });

  useEffect(() => {
    if (!modelsData) {
      return;
    }
    const models = modelsData.models;
    const fallbackModel = modelsData.defaultModel || models[0] || "";
    if (!fallbackModel) {
      return;
    }
    if (!threadModel || !models.includes(threadModel)) {
      setThreadModel(fallbackModel);
    }
  }, [modelsData, threadModel]);

  // Set initial worker when data loads
  const initialWorkerSet = useRef(false);
  if (!initialWorkerSet.current && workers.length > 0 && !workerId) {
    initialWorkerSet.current = true;
    setWorkerId(workers[0].id);
  }

  const createThreadMutation = useMutation({
    mutationFn: ({ prompt }: { prompt: string }) =>
      threadClient.createThread({
        projectId: currentProject?.id ?? "",
        agent: AgentSchema.value[agent].name,
        model: threadModel,
        prompt,
        mode: threadMode,
        workerId,
        sessionMode,
      }),
    onSuccess: (resp, variables) => {
      const id = resp.thread?.id ?? "";
      if (id) {
        const bootstrapState: ThreadBootstrapState = {
          initialPrompt: variables.prompt,
          createdAt: Date.now(),
        };
        queryClient.setQueryData(["thread-bootstrap", id], bootstrapState);
      }
      queryClient.invalidateQueries({ queryKey: ["threads"] });
      navigate({
        to: "/app/threads/$threadId",
        params: { threadId: id },
        replace: true,
      });
    },
  });

  const handleSendMessage = useCallback(
    (message: string) => {
      createThreadMutation.mutate({ prompt: message });
    },
    [createThreadMutation],
  );

  const handleProjectChange = (newProjectId: string) => {
    navigate({
      to: "/app/threads/new",
      search: { projectId: newProjectId },
      replace: true,
    });
  };

  const [leftPanelPercent, setLeftPanelPercent] = useState(50);
  const containerRef = useRef<HTMLDivElement>(null);

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      const startX = e.clientX;
      const startPercent = leftPanelPercent;
      const containerWidth = containerRef.current?.offsetWidth ?? 1;

      const handleMouseMove = (moveEvent: MouseEvent) => {
        const deltaX = moveEvent.clientX - startX;
        const deltaPercent = (deltaX / containerWidth) * 100;
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

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div
        className="flex items-center gap-2 border-b h-10 shrink-0 select-none px-4"
        style={dragStyle}
      >
        <Badge
          variant="outline"
          className="text-[10px] px-1.5 py-0 h-5 shrink-0 font-medium text-violet-400 border-violet-500/30"
        >
          New Thread
        </Badge>
        <Badge
          variant="outline"
          className="text-[10px] px-1.5 py-0 h-5 shrink-0 font-medium text-slate-400 border-slate-500/30"
        >
          Draft
        </Badge>
      </div>

      {/* Body — always two panels */}
      <div className="flex flex-1 min-h-0" ref={containerRef}>
        {/* Left: Chat panel */}
        <div
          className="flex-shrink-0 h-full overflow-hidden"
          style={{ width: `${leftPanelPercent}%` }}
        >
          <AgentChatPanel
            target={{
              type: "thread_overseer",
              entityId: "new",
              agentName: "Overseer",
              title: "New Thread",
              agentColor: "bg-violet-500",
            }}
            hideHeader
            onSend={handleSendMessage}
            emptyStateContent={
              <ThreadSetupForm
                threadMode={threadMode}
                onModeChange={setThreadMode}
                threadModel={threadModel}
                onModelChange={setThreadModel}
                project={currentProject}
                projects={projects}
                onProjectChange={handleProjectChange}
                workerId={workerId}
                workers={workers}
                onWorkerChange={setWorkerId}
                agent={agent}
                onAgentChange={setAgent}
                sessionMode={sessionMode}
                onSessionModeChange={setSessionMode}
                availableModels={modelsData?.models ?? []}
                defaultModel={modelsData?.defaultModel ?? ""}
                modelsLoading={modelsLoading}
                modelsError={modelsIsError ? "Could not load models for this agent" : null}
              />
            }
          />
        </div>
        {/* Resize handle */}
        <div
          className="w-3 -ml-[6px] -mr-[5px] flex-shrink-0 cursor-col-resize flex justify-center group relative z-10"
          onMouseDown={handleMouseDown}
        >
          <div className="w-px h-full bg-border group-hover:bg-primary/30 transition-colors" />
        </div>
        {/* Right: Empty state */}
        <div className="min-w-0 flex-1 overflow-hidden flex items-center justify-center">
          <div className="text-center text-muted-foreground">
            <MessageSquarePlus className="size-10 mx-auto mb-3 opacity-30" />
            <p className="text-sm font-medium">Describe your thread</p>
            <p className="text-xs mt-1 max-w-[240px]">
              The overseer will break it down into tasks once you send your first message.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
