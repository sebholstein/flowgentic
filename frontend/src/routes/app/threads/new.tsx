import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useState, useCallback, useRef } from "react";
import { MessageSquarePlus } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { dragStyle } from "@/components/layout/WindowDragRegion";
import { AgentChatPanel } from "@/components/chat/AgentChatPanel";
import { projects } from "@/data/mockThreadsData";
import { useInfrastructureStore, selectActiveControlPlane } from "@/stores/serverStore";

type SearchParams = {
  projectId?: string;
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

  const currentProject = projects.find((p) => p.id === projectId) ?? projects[0];
  const [threadMode, setThreadMode] = useState<"single_agent" | "orchestrated">("orchestrated");
  const [threadModel, setThreadModel] = useState("default");
  const [harness, setHarness] = useState("claude-code");

  const activeCP = useInfrastructureStore(selectActiveControlPlane);
  const allWorkers = useInfrastructureStore((s) => s.workers);
  const workers = allWorkers.filter((w) => w.controlPlaneId === activeCP.id);
  const [workerId, setWorkerId] = useState(workers[0]?.id ?? "");

  const handleProjectChange = (newProjectId: string) => {
    navigate({
      to: "/app/threads/new",
      search: { projectId: newProjectId },
      replace: true,
    });
  };

  const isSingleAgent = threadMode === "single_agent";

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

  const chatPanel = (
    <AgentChatPanel
      target={{
        type: "thread_overseer",
        entityId: "new",
        agentName: "Overseer",
        title: "New Thread",
        agentColor: "bg-violet-500",
      }}
      hideHeader
      enableSimulation
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
      harness={harness}
      onHarnessChange={setHarness}
    />
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

      {/* Body */}
      <div className="flex flex-1 min-h-0" ref={containerRef}>
        {isSingleAgent ? (
          <div className="flex-1 h-full overflow-hidden">{chatPanel}</div>
        ) : (
          <>
            {/* Left: Chat panel */}
            <div
              className="flex-shrink-0 h-full overflow-hidden"
              style={{ width: `${leftPanelPercent}%` }}
            >
              {chatPanel}
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
          </>
        )}
      </div>
    </div>
  );
}
