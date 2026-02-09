import { Link } from "@tanstack/react-router";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { X, ExternalLink, GitCompare } from "lucide-react";
import type { Task } from "@/types/task";
import { taskStatusConfig } from "@/constants/taskStatusConfig";

interface TaskDetailSidebarProps {
  task: Task;
  tasks: Task[];
  threadId: string;
  onClose: () => void;
  onSelectTask: (taskId: string) => void;
}

/**
 * Sidebar panel showing task overview in the graph view.
 * Displays task status, dependencies, subtasks progress, and links to full details.
 */
export function TaskDetailSidebar({
  task,
  tasks,
  threadId,
  onClose,
  onSelectTask,
}: TaskDetailSidebarProps) {
  const statusCfg = taskStatusConfig[task.status];
  const StatusIcon = statusCfg.icon;
  const dependencyTasks = task.dependencies
    .map((id) => tasks.find((t) => t.id === id))
    .filter(Boolean) as Task[];
  const dependentTasks = tasks.filter((t) => t.dependencies.includes(task.id));

  const subtasksDone = task.subtasks?.filter((s) => s.completed).length ?? 0;
  const subtasksTotal = task.subtasks?.length ?? 0;
  const hasSubtasks = subtasksTotal > 0;
  const hasExecutions = (task.executions?.length ?? 0) > 0;

  return (
    <div className="flex h-full flex-col bg-sidebar">
      <div className="flex items-center justify-between border-b px-4 py-3">
        <span className="text-sm font-medium">Task Overview</span>
        <Button variant="ghost" size="sm" onClick={onClose} className="size-7 p-0">
          <X className="size-4" />
        </Button>
      </div>
      <ScrollArea className="flex-1">
        <div className="space-y-4 p-4">
          {/* Header */}
          <div className="space-y-2">
            <div className="flex items-center gap-2">
              <div className={cn("rounded-md p-1.5", statusCfg.bgColor)}>
                <StatusIcon
                  className={cn(
                    "size-4",
                    statusCfg.color,
                    task.status === "running" && "animate-spin",
                  )}
                />
              </div>
              <Badge variant="outline" className="text-[0.65rem]">
                {statusCfg.label}
              </Badge>
              {task.agent && (
                <Badge variant="outline" className="text-[0.65rem]">
                  {task.agent}
                </Badge>
              )}
            </div>
            <h3 className="text-base font-semibold">{task.name}</h3>
            <p className="text-sm text-muted-foreground line-clamp-3">{task.description}</p>
          </div>

          {/* Quick Stats */}
          <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
            {task.duration && <span>{task.duration}</span>}
            {hasExecutions && (
              <Badge className="bg-purple-500/20 text-purple-400 border-purple-500/30 text-[0.6rem]">
                <GitCompare className="size-2.5 mr-0.5" />
                {task.executions?.length} execution{task.executions?.length !== 1 ? "s" : ""}
              </Badge>
            )}
          </div>

          {/* Subtasks Progress */}
          {hasSubtasks && (
            <div className="space-y-1.5">
              <div className="flex items-center justify-between text-xs">
                <span className="text-muted-foreground">Subtasks</span>
                <span className="text-muted-foreground tabular-nums">
                  {subtasksDone}/{subtasksTotal}
                </span>
              </div>
              <div className="h-1.5 bg-muted rounded-full overflow-hidden">
                <div
                  className={cn(
                    "h-full rounded-full transition-all",
                    subtasksDone === subtasksTotal ? "bg-emerald-500" : "bg-blue-500",
                  )}
                  style={{
                    width: `${subtasksTotal > 0 ? (subtasksDone / subtasksTotal) * 100 : 0}%`,
                  }}
                />
              </div>
            </div>
          )}

          {/* Dependencies - Compact */}
          {(dependencyTasks.length > 0 || dependentTasks.length > 0) && (
            <div className="space-y-2 text-xs">
              {dependencyTasks.length > 0 && (
                <div className="flex items-center gap-1.5 flex-wrap">
                  <span className="text-muted-foreground">Depends on:</span>
                  {dependencyTasks.slice(0, 2).map((dep) => {
                    const depStatus = taskStatusConfig[dep.status];
                    const DepIcon = depStatus.icon;
                    return (
                      <button
                        key={dep.id}
                        onClick={() => onSelectTask(dep.id)}
                        className="flex items-center gap-1 rounded border border-border px-1.5 py-0.5 hover:bg-muted transition-colors"
                      >
                        <DepIcon className={cn("size-2.5", depStatus.color)} />
                        <span className="max-w-[60px] truncate">{dep.name}</span>
                      </button>
                    );
                  })}
                  {dependencyTasks.length > 2 && (
                    <span className="text-muted-foreground">+{dependencyTasks.length - 2}</span>
                  )}
                </div>
              )}
              {dependentTasks.length > 0 && (
                <div className="text-muted-foreground">
                  Blocks {dependentTasks.length} task{dependentTasks.length > 1 ? "s" : ""}
                </div>
              )}
            </div>
          )}

          {/* View Full Details Link */}
          <Link
            to="/app/tasks/$threadId/$taskId"
            params={{ threadId, taskId: task.id }}
            className="flex items-center justify-center gap-1.5 w-full rounded-md border border-border bg-muted/50 px-3 py-2 text-xs text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
          >
            <span>View full details</span>
            <ExternalLink className="size-3" />
          </Link>
        </div>
      </ScrollArea>
    </div>
  );
}
