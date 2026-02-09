import { createFileRoute } from "@tanstack/react-router";
import { useState, useCallback, useMemo } from "react";
import { cn } from "@/lib/utils";
import { ScrollArea } from "@/components/ui/scroll-area";
import { GitCommitHorizontal, CornerDownRight } from "lucide-react";
import { TaskDetailCard } from "@/components/threads/TaskDetailCard";
import { calculateTaskLevels, groupTasksByLevel } from "@/lib/task-utils";
import { useThreadContext } from "../route";

export const Route = createFileRoute("/app/threads/$threadId/tasks/")({
  component: TasksTab,
});

function TasksTab() {
  const { thread, tasks, onSelectTask } = useThreadContext();
  const [expandedTaskId, setExpandedTaskId] = useState<string | null>(null);

  const handleToggleTask = useCallback((taskId: string) => {
    setExpandedTaskId((current) => (current === taskId ? null : taskId));
  }, []);

  const handleSelectTask = useCallback(
    (taskId: string) => {
      setExpandedTaskId(taskId);
      onSelectTask(taskId);
    },
    [onSelectTask],
  );

  // Calculate task levels and group them
  const taskLevels = useMemo(() => calculateTaskLevels(tasks), [tasks]);
  const taskGroups = useMemo(() => groupTasksByLevel(tasks, taskLevels), [tasks, taskLevels]);

  return (
    <ScrollArea className="h-full">
      <div className="max-w-4xl mx-auto p-8">
        <div className="space-y-2">
          {taskGroups.map(({ level, tasks: levelTasks }) => (
            <div key={level} className="relative">
              {/* Level indicator - compact inline */}
              <div className="flex items-center gap-2 mb-1.5">
                {level === 0 ? (
                  <GitCommitHorizontal className="size-3 text-muted-foreground" />
                ) : (
                  <CornerDownRight className="size-3 text-muted-foreground" />
                )}
                <span className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
                  {level === 0 ? "Start" : `Step ${level}`}
                </span>
                {levelTasks.length > 1 && (
                  <span className="text-[0.6rem] text-muted-foreground">
                    · {levelTasks.length} parallel
                  </span>
                )}
              </div>

              {/* Tasks at this level with indentation */}
              <div
                className={cn("space-y-1.5", level > 0 && "ml-4 pl-3 border-l border-border/50")}
              >
                {levelTasks.map((task) => (
                  <TaskDetailCard
                    key={task.id}
                    task={task}
                    tasks={tasks}
                    threadId={thread.id}
                    isExpanded={expandedTaskId === task.id}
                    onToggle={() => handleToggleTask(task.id)}
                    onSelectTask={handleSelectTask}
                  />
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    </ScrollArea>
  );
}
