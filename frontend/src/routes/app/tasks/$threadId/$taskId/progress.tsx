import { createFileRoute, Link } from "@tanstack/react-router";
import { cn } from "@/lib/utils";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  CheckCircle2,
  Circle,
  Loader2,
  Target,
  ChevronDown,
  ChevronRight,
  Clock,
  Zap,
} from "lucide-react";
import { useState } from "react";
import { taskStatusConfig } from "@/constants/taskStatusConfig";
import { useTaskContext } from "./context";

export const Route = createFileRoute("/app/tasks/$threadId/$taskId/progress")({
  component: ProgressTab,
});

function ProgressTab() {
  const { task, progressSteps, allTasks, thread } = useTaskContext();
  const [showDependencies, setShowDependencies] = useState(true);

  const completedSteps = progressSteps.filter((s) => s.status === "completed").length;
  const totalSteps = progressSteps.length;
  const isCompleted = task.status === "completed";

  // Find dependency and dependent tasks
  const dependencyTasks = allTasks.filter((t) => task.dependencies.includes(t.id));
  const dependentTasks = allTasks.filter((t) => t.dependencies.includes(task.id));
  const hasDependencies = dependencyTasks.length > 0 || dependentTasks.length > 0;

  return (
    <ScrollArea className="h-full">
      <div className="max-w-3xl mx-auto p-6 space-y-6">
        {/* Steps */}
        <div className="rounded-lg border">
          <div className="flex items-center justify-between px-3 py-2 border-b">
            <div className="flex items-center gap-2">
              <Target className="size-3.5 text-muted-foreground" />
              <h3 className="text-sm font-medium">Steps</h3>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-xs text-muted-foreground tabular-nums">
                {completedSteps}/{totalSteps}
              </span>
              <div className="w-16 h-1.5 bg-muted rounded-full overflow-hidden">
                <div
                  className={cn(
                    "h-full rounded-full transition-all",
                    isCompleted ? "bg-emerald-500" : "bg-blue-500",
                  )}
                  style={{ width: `${(completedSteps / totalSteps) * 100}%` }}
                />
              </div>
            </div>
          </div>
          <div className="p-2 space-y-0.5">
            {progressSteps.map((step, index) => (
              <div
                key={step.id}
                className={cn(
                  "flex items-center gap-2 py-1.5 px-2 rounded transition-colors",
                  step.status === "running" && "bg-blue-500/5",
                  step.status === "completed" && "text-muted-foreground",
                )}
              >
                {step.status === "completed" ? (
                  <CheckCircle2 className="size-3.5 text-emerald-500 shrink-0" />
                ) : step.status === "running" ? (
                  <Loader2 className="size-3.5 text-blue-500 animate-spin shrink-0" />
                ) : (
                  <Circle className="size-3.5 text-muted-foreground/40 shrink-0" />
                )}
                <span className="text-xs text-muted-foreground/60 tabular-nums">{index + 1}.</span>
                <span
                  className={cn(
                    "text-xs",
                    step.status === "completed" && "text-muted-foreground",
                    step.status === "running" && "text-blue-500 font-medium",
                  )}
                >
                  {step.name}
                </span>
              </div>
            ))}
          </div>
        </div>

        {/* Task Metadata */}
        <div className="grid grid-cols-2 gap-4">
          {task.duration && (
            <div className="rounded-lg border p-4">
              <div className="flex items-center gap-2 text-muted-foreground mb-2">
                <Clock className="size-4" />
                <span className="text-xs font-medium uppercase tracking-wider">Duration</span>
              </div>
              <p className="text-lg font-semibold">{task.duration}</p>
            </div>
          )}
          {task.agent && (
            <div className="rounded-lg border p-4">
              <div className="flex items-center gap-2 text-muted-foreground mb-2">
                <Zap className="size-4" />
                <span className="text-xs font-medium uppercase tracking-wider">Agent</span>
              </div>
              <p className="text-lg font-semibold">{task.agent}</p>
            </div>
          )}
        </div>

        {/* Dependencies Section */}
        {hasDependencies && (
          <div className="rounded-lg border">
            <button
              onClick={() => setShowDependencies(!showDependencies)}
              className="flex items-center justify-between w-full p-4 hover:bg-muted/50 transition-colors"
            >
              <h3 className="font-medium">Task Dependencies</h3>
              {showDependencies ? (
                <ChevronDown className="size-4 text-muted-foreground" />
              ) : (
                <ChevronRight className="size-4 text-muted-foreground" />
              )}
            </button>
            {showDependencies && (
              <div className="px-4 pb-4 space-y-4">
                {dependencyTasks.length > 0 && (
                  <div>
                    <div className="text-xs text-muted-foreground mb-2 uppercase tracking-wider">
                      Depends on ({dependencyTasks.length})
                    </div>
                    <div className="space-y-2">
                      {dependencyTasks.map((dep) => {
                        const depStatus = taskStatusConfig[dep.status];
                        const DepIcon = depStatus.icon;
                        return (
                          <Link
                            key={dep.id}
                            to="/app/tasks/$threadId/$taskId"
                            params={{ threadId: thread.id, taskId: dep.id }}
                            className="flex items-center gap-3 rounded-lg border px-3 py-2 hover:bg-muted/50 transition-colors"
                          >
                            <div className={cn("rounded-md p-1", depStatus.bgColor)}>
                              <DepIcon className={cn("size-3.5", depStatus.color)} />
                            </div>
                            <div className="flex-1 min-w-0">
                              <span className="text-sm font-medium truncate">{dep.name}</span>
                            </div>
                            <span className={cn("text-xs", depStatus.color)}>
                              {depStatus.label}
                            </span>
                          </Link>
                        );
                      })}
                    </div>
                  </div>
                )}
                {dependentTasks.length > 0 && (
                  <div>
                    <div className="text-xs text-muted-foreground mb-2 uppercase tracking-wider">
                      Blocks ({dependentTasks.length})
                    </div>
                    <div className="space-y-2">
                      {dependentTasks.map((dep) => {
                        const depStatus = taskStatusConfig[dep.status];
                        const DepIcon = depStatus.icon;
                        return (
                          <Link
                            key={dep.id}
                            to="/app/tasks/$threadId/$taskId"
                            params={{ threadId: thread.id, taskId: dep.id }}
                            className="flex items-center gap-3 rounded-lg border px-3 py-2 hover:bg-muted/50 transition-colors"
                          >
                            <div className={cn("rounded-md p-1", depStatus.bgColor)}>
                              <DepIcon className={cn("size-3.5", depStatus.color)} />
                            </div>
                            <div className="flex-1 min-w-0">
                              <span className="text-sm font-medium truncate">{dep.name}</span>
                            </div>
                            <span className={cn("text-xs", depStatus.color)}>
                              {depStatus.label}
                            </span>
                          </Link>
                        );
                      })}
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        )}
      </div>
    </ScrollArea>
  );
}
