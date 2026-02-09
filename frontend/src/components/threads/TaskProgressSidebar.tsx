import { Link } from "@tanstack/react-router";
import { cn } from "@/lib/utils";
import { ScrollArea } from "@/components/ui/scroll-area";
import { CheckCircle2, Circle, Loader2, Target, ChevronDown, ChevronRight } from "lucide-react";
import { useState } from "react";
import { taskStatusConfig } from "@/constants/taskStatusConfig";
import { ResourceRequirementList } from "@/components/resources/ResourceRequirementList";
import type { Task } from "@/types/task";

interface ProgressStep {
  id: string;
  name: string;
  status: "completed" | "running" | "pending";
  detail: string;
}

interface TaskProgressSidebarProps {
  task: Task;
  progressSteps: ProgressStep[];
  dependencyTasks: Task[];
  dependentTasks: Task[];
  threadId: string;
  isCompleted: boolean;
  className?: string;
}

export function TaskProgressSidebar({
  task,
  progressSteps,
  dependencyTasks,
  dependentTasks,
  threadId,
  isCompleted,
  className,
}: TaskProgressSidebarProps) {
  const [showDependencies, setShowDependencies] = useState(true);

  const completedSteps = progressSteps.filter((s) => s.status === "completed").length;
  const totalSteps = progressSteps.length;
  const hasDependencies = dependencyTasks.length > 0 || dependentTasks.length > 0;
  const hasResources =
    (task.inputRequirements && task.inputRequirements.length > 0) ||
    (task.outputRequirements && task.outputRequirements.length > 0);

  return (
    <ScrollArea className={cn("h-full", className)}>
      <div className="p-4 space-y-4">
        {/* Progress Section */}
        <div className="rounded-lg border">
          <div className="flex items-center justify-between p-3 border-b">
            <div className="flex items-center gap-2">
              <Target className="size-4 text-muted-foreground" />
              <h3 className="font-medium text-sm">Progress</h3>
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
            {progressSteps.map((step) => (
              <div
                key={step.id}
                className={cn(
                  "flex items-start gap-2 py-1.5 px-2 rounded-md transition-colors",
                  step.status === "running" && "bg-blue-500/5",
                  step.status === "completed" && "text-muted-foreground",
                )}
              >
                <div className="mt-0.5">
                  {step.status === "completed" ? (
                    <CheckCircle2 className="size-3.5 text-emerald-500" />
                  ) : step.status === "running" ? (
                    <Loader2 className="size-3.5 text-blue-500 animate-spin" />
                  ) : (
                    <Circle className="size-3.5 text-muted-foreground/50" />
                  )}
                </div>
                <div className="flex-1 min-w-0">
                  <span
                    className={cn(
                      "text-xs font-medium",
                      step.status === "completed" && "line-through text-muted-foreground",
                      step.status === "running" && "text-blue-500",
                    )}
                  >
                    {step.name}
                  </span>
                  {step.status === "running" && (
                    <p className="text-[10px] text-muted-foreground mt-0.5 truncate">
                      {step.detail}
                    </p>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Dependencies Section */}
        {hasDependencies && (
          <div className="rounded-lg border">
            <button
              onClick={() => setShowDependencies(!showDependencies)}
              className="flex items-center justify-between w-full p-3 hover:bg-muted/50 transition-colors"
            >
              <h3 className="text-sm font-medium">Dependencies</h3>
              {showDependencies ? (
                <ChevronDown className="size-4 text-muted-foreground" />
              ) : (
                <ChevronRight className="size-4 text-muted-foreground" />
              )}
            </button>
            {showDependencies && (
              <div className="px-3 pb-3 space-y-3">
                {dependencyTasks.length > 0 && (
                  <div>
                    <div className="text-[10px] text-muted-foreground mb-1.5 uppercase tracking-wider">
                      Depends on
                    </div>
                    <div className="space-y-1">
                      {dependencyTasks.map((dep) => {
                        const depStatus = taskStatusConfig[dep.status];
                        const DepIcon = depStatus.icon;
                        return (
                          <Link
                            key={dep.id}
                            to="/app/tasks/$threadId/$taskId"
                            params={{ threadId, taskId: dep.id }}
                            className="flex items-center gap-1.5 rounded-md border px-2 py-1.5 text-xs hover:bg-muted transition-colors"
                          >
                            <DepIcon className={cn("size-3", depStatus.color)} />
                            <span className="truncate">{dep.name}</span>
                          </Link>
                        );
                      })}
                    </div>
                  </div>
                )}
                {dependentTasks.length > 0 && (
                  <div>
                    <div className="text-[10px] text-muted-foreground mb-1.5 uppercase tracking-wider">
                      Blocks
                    </div>
                    <div className="space-y-1">
                      {dependentTasks.map((dep) => {
                        const depStatus = taskStatusConfig[dep.status];
                        const DepIcon = depStatus.icon;
                        return (
                          <Link
                            key={dep.id}
                            to="/app/tasks/$threadId/$taskId"
                            params={{ threadId, taskId: dep.id }}
                            className="flex items-center gap-1.5 rounded-md border px-2 py-1.5 text-xs hover:bg-muted transition-colors"
                          >
                            <DepIcon className={cn("size-3", depStatus.color)} />
                            <span className="truncate">{dep.name}</span>
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

        {/* Resources Section */}
        {hasResources && (
          <ResourceRequirementList
            inputRequirements={task.inputRequirements}
            outputRequirements={task.outputRequirements}
            availableResources={task.availableResources}
            compact
          />
        )}
      </div>
    </ScrollArea>
  );
}
