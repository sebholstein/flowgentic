import type { Task } from "@/types/task";

/**
 * Calculate task depth based on dependencies.
 * Tasks with no dependencies are at level 0, tasks depending on level 0 tasks are level 1, etc.
 */
export function calculateTaskLevels(tasks: Task[]): Map<string, number> {
  const taskMap = new Map(tasks.map((t) => [t.id, t]));
  const levels = new Map<string, number>();

  function getLevel(taskId: string, visited: Set<string> = new Set()): number {
    if (visited.has(taskId)) return 0;
    visited.add(taskId);

    if (levels.has(taskId)) return levels.get(taskId)!;

    const task = taskMap.get(taskId);
    if (!task || task.dependencies.length === 0) {
      levels.set(taskId, 0);
      return 0;
    }

    const maxDepLevel = Math.max(
      ...task.dependencies.map((depId) => getLevel(depId, new Set(visited))),
    );
    const level = maxDepLevel + 1;
    levels.set(taskId, level);
    return level;
  }

  tasks.forEach((t) => getLevel(t.id));
  return levels;
}

/**
 * Group tasks by their level for hierarchical display.
 */
export function groupTasksByLevel(
  tasks: Task[],
  levels: Map<string, number>,
): { level: number; tasks: Task[] }[] {
  const groups = new Map<number, Task[]>();

  tasks.forEach((task) => {
    const level = levels.get(task.id) ?? 0;
    if (!groups.has(level)) groups.set(level, []);
    groups.get(level)!.push(task);
  });

  return Array.from(groups.entries())
    .sort(([a], [b]) => a - b)
    .map(([level, levelTasks]) => ({ level, tasks: levelTasks }));
}
