import type { Node, Edge } from "@xyflow/react";
import { MarkerType } from "@xyflow/react";
import type { Task } from "@/types/task";
import type { TaskNodeData } from "@/components/threads/TaskNode";
import type { ExecutionNoteData } from "@/components/threads/ExecutionNote";
import { taskStatusConfig } from "@/constants/taskStatusConfig";

/**
 * Generates a DAG (Directed Acyclic Graph) layout for tasks and their executions.
 * Positions nodes in levels based on dependencies, with execution notes to the right of tasks.
 */
export function generateDAGLayout(
  tasks: Task[],
  selectedTaskId?: string,
): { nodes: Node<TaskNodeData | ExecutionNoteData>[]; edges: Edge[] } {
  const taskMap = new Map(tasks.map((t) => [t.id, t]));
  const levels: Map<string, number> = new Map();

  function getLevel(taskId: string, visited: Set<string> = new Set()): number {
    if (visited.has(taskId)) return 0;
    visited.add(taskId);

    if (levels.has(taskId)) return levels.get(taskId)!;

    const task = taskMap.get(taskId);
    if (!task || task.dependencies.length === 0) {
      levels.set(taskId, 0);
      return 0;
    }

    const maxDepLevel = Math.max(...task.dependencies.map((depId) => getLevel(depId, visited)));
    const level = maxDepLevel + 1;
    levels.set(taskId, level);
    return level;
  }

  tasks.forEach((t) => getLevel(t.id));

  const levelGroups: Map<number, Task[]> = new Map();
  tasks.forEach((task) => {
    const level = levels.get(task.id) ?? 0;
    if (!levelGroups.has(level)) levelGroups.set(level, []);
    levelGroups.get(level)!.push(task);
  });

  const nodeWidth = 280;
  const nodeHeight = 100;
  const horizontalSpacing = 200; // Wide spacing for execution notes on the right
  const verticalSpacing = 140; // Generous vertical spacing for edges

  const executionNoteHeight = 40;
  const executionNoteGap = 12; // Gap between stacked execution notes

  const nodes: Node<TaskNodeData | ExecutionNoteData>[] = [];

  levelGroups.forEach((levelTasks, level) => {
    const totalWidth = levelTasks.length * nodeWidth + (levelTasks.length - 1) * horizontalSpacing;
    const startX = -totalWidth / 2;

    levelTasks.forEach((task, index) => {
      const taskX = startX + index * (nodeWidth + horizontalSpacing);
      const taskY = level * (nodeHeight + verticalSpacing);

      // Add task node
      nodes.push({
        id: task.id,
        type: "taskNode",
        position: { x: taskX, y: taskY },
        selected: task.id === selectedTaskId,
        data: {
          task: {
            ...task,
            executionCount: task.executions?.length,
          },
          statusConfig: taskStatusConfig[task.status],
        },
      });

      // Add execution note nodes to the right of the task
      if (task.executions && task.executions.length > 0) {
        task.executions.forEach((execution, execIndex) => {
          nodes.push({
            id: `exec-note-${task.id}-${execution.id}`,
            type: "executionNote",
            position: {
              x: taskX + nodeWidth + 20, // Position to the right of task
              y: taskY + execIndex * (executionNoteHeight + executionNoteGap),
            },
            data: {
              execution: {
                ...execution,
                isSelected: execution.id === task.selectedExecutionId,
              },
              taskId: task.id,
            },
            selectable: false,
            draggable: false,
          });
        });
      }
    });
  });

  // Dependency edges between tasks
  const dependencyEdges: Edge[] = tasks.flatMap((task) =>
    task.dependencies.map((depId) => ({
      id: `${depId}-${task.id}`,
      source: depId,
      target: task.id,
      type: "smoothstep",
      animated: taskMap.get(depId)?.status === "running",
      style: {
        stroke: taskMap.get(depId)?.status === "completed" ? "#34d399" : "#64748b",
        strokeWidth: 2,
      },
      markerEnd: {
        type: MarkerType.ArrowClosed,
        color: taskMap.get(depId)?.status === "completed" ? "#34d399" : "#64748b",
      },
    })),
  );

  // Edges connecting tasks to their execution notes
  const executionEdges: Edge[] = tasks.flatMap((task) =>
    (task.executions ?? []).map((execution) => ({
      id: `${task.id}-exec-${execution.id}`,
      source: task.id,
      target: `exec-note-${task.id}-${execution.id}`,
      type: "straight",
      style: {
        stroke: execution.id === task.selectedExecutionId ? "#34d399" : "#475569",
        strokeWidth: 1,
        strokeDasharray: execution.id === task.selectedExecutionId ? undefined : "4 2",
      },
      sourceHandle: null,
      targetHandle: null,
    })),
  );

  return { nodes, edges: [...dependencyEdges, ...executionEdges] };
}
