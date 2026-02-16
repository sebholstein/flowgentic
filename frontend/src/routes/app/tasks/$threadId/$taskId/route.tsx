import { createFileRoute, Outlet, Link, useParams, useRouterState } from "@tanstack/react-router";
import { dragStyle, noDragStyle } from "@/components/layout/WindowDragRegion";
import { useCallback, useMemo, useState, useRef } from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ChevronLeft, ListChecks, Brain, FileText, GitCompare } from "lucide-react";
import { taskStatusConfig } from "@/constants/taskStatusConfig";
import { threadTasks } from "@/data/mockTasksData";
import { threads } from "@/data/mockThreadsData";
import { mockProgressSteps, mockRunningProgressSteps } from "@/data/mockTaskData";
import { TabbedChatPanel } from "@/components/chat/TabbedChatPanel";
import { TaskContext, type TaskContextValue } from "./context";

export const Route = createFileRoute("/app/tasks/$threadId/$taskId")({
  component: TaskLayout,
});

export { useTaskContext } from "./context";

function TaskLayout() {
  const { threadId, taskId } = useParams({ from: "/app/tasks/$threadId/$taskId" });
  const routerState = useRouterState();
  const [chatPanelPercent, setChatPanelPercent] = useState(50);
  const containerRef = useRef<HTMLDivElement>(null);

  const thread = threads.find((t) => t.id === threadId);
  const tasks = useMemo(() => threadTasks[threadId] ?? [], [threadId]);
  const task = tasks.find((t) => t.id === taskId);

  const isRunning = task?.status === "running";
  const progressSteps = isRunning ? mockRunningProgressSteps : mockProgressSteps;

  // Resize handle for chat panel
  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      const startX = e.clientX;
      const startPercent = chatPanelPercent;
      const containerWidth = containerRef.current?.offsetWidth ?? 1;

      const handleMouseMove = (moveEvent: MouseEvent) => {
        const deltaX = moveEvent.clientX - startX;
        const deltaPercent = (deltaX / containerWidth) * 100;
        // Clamp between 30% and 70%
        setChatPanelPercent(Math.min(70, Math.max(30, startPercent + deltaPercent)));
      };

      const handleMouseUp = () => {
        document.removeEventListener("mousemove", handleMouseMove);
        document.removeEventListener("mouseup", handleMouseUp);
      };

      document.addEventListener("mousemove", handleMouseMove);
      document.addEventListener("mouseup", handleMouseUp);
    },
    [chatPanelPercent],
  );

  if (!thread || !task) {
    return (
      <div className="flex h-full items-center justify-center text-muted-foreground">
        Task not found
      </div>
    );
  }

  const statusConfig = taskStatusConfig[task.status];
  const StatusIcon = statusConfig.icon;

  const completedSteps = progressSteps.filter((s) => s.status === "completed").length;
  const totalSteps = progressSteps.length;

  // Determine active tab from current route
  const currentPath = routerState.location.pathname;
  const getActiveTab = () => {
    if (currentPath.endsWith("/changes")) return "changes";
    if (currentPath.endsWith("/progress")) return "progress";
    if (currentPath.endsWith("/memory")) return "memory";
    return "overview";
  };
  const activeTab = getActiveTab();

  const tabTriggerClass =
    "h-10 flex items-center rounded-none border-b-2 border-transparent data-[active=true]:border-primary px-0.5 gap-1.5 text-xs text-muted-foreground hover:text-foreground data-[active=true]:text-foreground transition-colors cursor-pointer";

  const contextValue: TaskContextValue = {
    task,
    thread,
    progressSteps,
    allTasks: tasks,
  };

  return (
    <TaskContext.Provider value={contextValue}>
      <div className="flex h-full flex-col">
        {/* Compact Header with integrated tabs - sits on dark base */}
        <div className="flex h-10 shrink-0 select-none" style={dragStyle}>
          {/* Left section - matches chat panel width */}
          <div
            className="flex items-center gap-2 px-4 shrink-0"
            style={{ width: `${chatPanelPercent}%` }}
          >
            <Button
              variant="ghost"
              size="sm"
              className="h-7 w-7 p-0 shrink-0"
              style={noDragStyle}
              asChild
            >
              <Link to="/app/threads/$threadId" params={{ threadId }}>
                <ChevronLeft className="size-4" />
              </Link>
            </Button>
            <Badge
              variant="outline"
              className="text-[10px] px-1.5 py-0 h-5 shrink-0 font-medium text-orange-400 border-orange-500/30"
            >
              Task
            </Badge>
            <StatusIcon
              className={cn("size-4 shrink-0", statusConfig.color, isRunning && "animate-spin")}
            />
            <span className="text-xs text-muted-foreground">#{task.id}</span>
            <span className="font-medium text-sm truncate">{task.name}</span>
          </div>

          {/* Right section - matches content panel */}
          <div className="flex-1 min-w-0 flex items-center px-4">
            {/* Tab Navigation */}
            <nav className="flex items-center gap-4 h-full" style={noDragStyle}>
              <Link
                to="/app/tasks/$threadId/$taskId"
                params={{ threadId, taskId }}
                className={cn(tabTriggerClass)}
                data-active={activeTab === "overview"}
              >
                <FileText className="size-3.5" />
                Plan
              </Link>
              <Link
                to="/app/tasks/$threadId/$taskId/changes"
                params={{ threadId, taskId }}
                className={cn(tabTriggerClass)}
                data-active={activeTab === "changes"}
              >
                <GitCompare className="size-3.5" />
                Changes
                <span className="text-[9px] tabular-nums">
                  <span className="text-emerald-500">+324</span>{" "}
                  <span className="text-red-500">-89</span>
                </span>
              </Link>
              <Link
                to="/app/tasks/$threadId/$taskId/progress"
                params={{ threadId, taskId }}
                className={cn(tabTriggerClass)}
                data-active={activeTab === "progress"}
              >
                <ListChecks className="size-3.5" />
                Steps
                <span className="text-muted-foreground">
                  ({completedSteps}/{totalSteps})
                </span>
              </Link>
              <Link
                to="/app/tasks/$threadId/$taskId/memory"
                params={{ threadId, taskId }}
                className={cn(tabTriggerClass)}
                data-active={activeTab === "memory"}
              >
                <Brain className="size-3.5" />
                Memory
              </Link>
            </nav>

            {/* Status badge - pushed to right */}
            <Badge className={cn("shrink-0 ml-auto", statusConfig.bgColor, statusConfig.color)}>
              {statusConfig.label}
            </Badge>
          </div>
        </div>

        {/* Main content area with chat on left and content on right */}
        <div className="flex flex-1 min-h-0 bg-surface rounded-lg overflow-hidden" ref={containerRef}>
          {/* Chat panel - always visible */}
          <div className="flex-shrink-0 overflow-hidden" style={{ width: `${chatPanelPercent}%` }}>
            <TabbedChatPanel
              entityId={task.id}
              entityType="task"
              entityTitle={task.name}
              currentStep={
                isRunning
                  ? {
                      name: progressSteps.find((s) => s.status === "running")?.label ?? task.name,
                      current: completedSteps + 1,
                      total: totalSteps,
                    }
                  : undefined
              }
            />
          </div>
          {/* Resize handle - wide hit area, thin visual line */}
          <div
            className="w-3 -ml-[6px] -mr-[5px] flex-shrink-0 cursor-col-resize flex justify-center group relative z-10"
            onMouseDown={handleMouseDown}
          >
            <div className="w-px h-full bg-border group-hover:bg-primary/30 transition-colors" />
          </div>

          {/* Right side: Tab content */}
          <div className="min-w-0 flex-1 overflow-hidden">
            <Outlet />
          </div>
        </div>
      </div>
    </TaskContext.Provider>
  );
}
