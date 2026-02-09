import { Link, useNavigate } from "@tanstack/react-router";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ChevronLeft, Loader2 } from "lucide-react";
import { taskStatusConfig } from "@/constants/taskStatusConfig";
import type { Task } from "@/types/task";

interface ProgressStep {
  id: string;
  name: string;
  status: "completed" | "running" | "pending";
  detail: string;
}

interface TaskHeaderProps {
  task: Task;
  threadId: string;
  threadTitle: string;
  progressSteps?: ProgressStep[];
}

export function TaskHeader({ task, threadId, threadTitle, progressSteps = [] }: TaskHeaderProps) {
  const navigate = useNavigate();
  const statusConfig = taskStatusConfig[task.status];
  const StatusIcon = statusConfig.icon;

  const isRunning = task.status === "running";
  const completedSteps = progressSteps.filter((s) => s.status === "completed").length;
  const totalSteps = progressSteps.length;
  const currentStep = progressSteps.find((s) => s.status === "running");
  const progressPercent = totalSteps > 0 ? (completedSteps / totalSteps) * 100 : 0;

  return (
    <div className="flex items-center gap-3 border-b px-4 py-3">
      <Button
        variant="ghost"
        size="sm"
        className="h-8 w-8 p-0 shrink-0"
        onClick={() => navigate({ to: "/app/threads/$threadId", params: { threadId } })}
      >
        <ChevronLeft className="size-4" />
      </Button>

      {/* Task info */}
      <div className="flex items-center gap-2 min-w-0">
        <div className={cn("rounded-md p-1.5 shrink-0", statusConfig.bgColor)}>
          <StatusIcon className={cn("size-4", statusConfig.color, isRunning && "animate-spin")} />
        </div>
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-xs text-muted-foreground">#{task.id}</span>
            <h1 className="font-semibold truncate">{task.name}</h1>
          </div>
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <Link
              to="/app/threads/$threadId"
              params={{ threadId }}
              className="hover:text-foreground transition-colors"
            >
              {threadTitle}
            </Link>
          </div>
        </div>
      </div>

      {/* Progress section - show when there are steps */}
      {totalSteps > 0 && (
        <div className="flex items-center gap-3 ml-auto mr-3">
          {/* Current step indicator for running tasks */}
          {isRunning && currentStep && (
            <div className="flex items-center gap-2 text-xs text-muted-foreground max-w-[280px]">
              <Loader2 className="size-3 text-blue-500 animate-spin shrink-0" />
              <span className="truncate">{currentStep.name}</span>
            </div>
          )}

          {/* Progress bar and count */}
          <div className="flex items-center gap-2">
            <div className="w-20 h-1.5 bg-muted rounded-full overflow-hidden">
              <div
                className={cn(
                  "h-full rounded-full transition-all",
                  task.status === "completed" ? "bg-emerald-500" : "bg-blue-500",
                )}
                style={{ width: `${progressPercent}%` }}
              />
            </div>
            <span className="text-xs text-muted-foreground tabular-nums whitespace-nowrap">
              {completedSteps}/{totalSteps}
            </span>
          </div>
        </div>
      )}

      {/* Status badge */}
      <Badge className={cn("shrink-0", statusConfig.bgColor, statusConfig.color)}>
        {statusConfig.label}
      </Badge>
    </div>
  );
}
