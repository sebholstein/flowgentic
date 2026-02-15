import { cn } from "@/lib/utils";
import { ScrollArea } from "@/components/ui/scroll-area";
import { taskStatusConfig } from "@/constants/taskStatusConfig";
import type { Task } from "@/types/task";

export function ExecutionProgressPanel({ tasks }: { tasks: Task[] }) {
  const completed = tasks.filter((t) => t.status === "completed").length;
  const total = tasks.length;
  const progressPercent = total > 0 ? Math.round((completed / total) * 100) : 0;

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="border-b px-4 py-2.5 space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium">Execution Progress</span>
          <span className="text-xs text-muted-foreground">
            {completed}/{total} tasks
          </span>
        </div>
        {/* Progress bar */}
        <div className="h-1.5 rounded-full bg-muted overflow-hidden">
          <div
            className="h-full rounded-full bg-emerald-500 transition-all duration-500"
            style={{ width: `${progressPercent}%` }}
          />
        </div>
      </div>

      {/* Task list */}
      <ScrollArea className="flex-1 min-h-0">
        <div className="p-3 space-y-1">
          {tasks.map((task) => {
            const config = taskStatusConfig[task.status];
            const StatusIcon = config.icon;
            return (
              <div
                key={task.id}
                className={cn(
                  "rounded-lg border p-3 space-y-2 transition-colors",
                  task.status === "running" && "border-blue-500/30 bg-blue-500/5",
                )}
              >
                <div className="flex items-center gap-2">
                  <StatusIcon
                    className={cn(
                      "size-3.5 shrink-0",
                      config.color,
                      task.status === "running" && "animate-spin",
                    )}
                  />
                  <span className="text-sm font-medium flex-1 truncate">{task.name}</span>
                  {task.duration && (
                    <span className="text-[10px] text-muted-foreground">{task.duration}</span>
                  )}
                </div>
                <p className="text-xs text-muted-foreground pl-5.5">{task.description}</p>

                {/* Subtasks */}
                {task.subtasks && task.subtasks.length > 0 && (
                  <div className="pl-5.5 space-y-0.5">
                    {task.subtasks.map((st) => (
                      <div key={st.id} className="flex items-center gap-1.5 text-xs">
                        <span
                          className={cn(
                            "h-1 w-1 rounded-full shrink-0",
                            st.completed ? "bg-emerald-400" : "bg-muted-foreground/30",
                          )}
                        />
                        <span
                          className={cn(
                            st.completed
                              ? "text-muted-foreground line-through"
                              : "text-foreground",
                          )}
                        >
                          {st.name}
                        </span>
                      </div>
                    ))}
                  </div>
                )}

                {/* Dependencies */}
                {task.dependencies.length > 0 && (
                  <div className="pl-5.5 text-[10px] text-muted-foreground">
                    blocked by:{" "}
                    {task.dependencies
                      .map((dep) => tasks.find((t) => t.id === dep)?.name ?? dep)
                      .join(", ")}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </ScrollArea>
    </div>
  );
}
