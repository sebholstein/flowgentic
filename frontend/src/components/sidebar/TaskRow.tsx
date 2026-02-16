import { memo } from "react";
import { Link } from "@tanstack/react-router";
import { cn } from "@/lib/utils";
import { taskStatusConfig } from "@/constants/taskStatusConfig";
import type { Task } from "@/types/task";

export const TaskRow = memo(function TaskRow({
  task,
  threadId,
  isSelected,
}: {
  task: Task;
  threadId: string;
  isSelected: boolean;
}) {
  const StatusIcon = taskStatusConfig[task.status].icon;

  return (
    <Link
      to="/app/tasks/$threadId/$taskId"
      params={{ threadId, taskId: task.id }}
      className={cn(
        "flex items-center gap-1.5 rounded-md px-1.5 py-1 text-xs hover:bg-muted/50 transition-colors text-left min-w-0 select-none",
        isSelected && "bg-muted text-foreground",
        !isSelected && "text-foreground hover:text-foreground",
      )}
      style={{ paddingLeft: "24px" }}
    >
      <StatusIcon
        className={cn(
          "size-3 shrink-0",
          taskStatusConfig[task.status].color,
          task.status === "running" && "animate-spin",
        )}
      />
      <span className="truncate flex-1">{task.name}</span>
    </Link>
  );
});
