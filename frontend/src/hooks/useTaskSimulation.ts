import { useState, useCallback, useEffect, useRef } from "react";
import type { Task, TaskStatus, TaskPlanStatus } from "@/types/task";

interface TaskExecution {
  id: string;
  agentId: string;
  agentName: string;
  status: "running" | "completed" | "failed";
  duration?: string;
  tokens?: {
    input: number;
    output: number;
    total: number;
  };
}

// Agents available for simulation
const simulationAgents = [
  { id: "claude-opus-4", name: "Claude Opus 4" },
  { id: "gpt-4", name: "GPT-4" },
  { id: "claude-sonnet", name: "Claude Sonnet" },
];

export function useTaskSimulation(initialTasks: Task[]) {
  const [tasks, setTasks] = useState<Task[]>(initialTasks);
  const [isSimulating, setIsSimulating] = useState(false);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const executionCounterRef = useRef(0);

  const resetTasks = useCallback(() => {
    setTasks(
      initialTasks.map((t) => ({
        ...t,
        status: "pending" as TaskStatus,
        duration: undefined,
        startedAt: undefined,
        completedAt: undefined,
        executions: undefined,
        selectedExecutionId: undefined,
        // Reset plan status but keep plannerPrompt/planApproval config
        planStatus: t.plannerPrompt ? ("pending" as TaskPlanStatus) : t.planStatus,
        plan: t.plannerPrompt ? undefined : t.plan,
      })),
    );
    setIsSimulating(false);
    executionCounterRef.current = 0;
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
  }, [initialTasks]);

  const canRun = useCallback((task: Task, currentTasks: Task[]) => {
    if (task.status !== "pending") return false;
    return task.dependencies.every((depId) => {
      const dep = currentTasks.find((t) => t.id === depId);
      return dep?.status === "completed";
    });
  }, []);

  const startSimulation = useCallback(() => {
    if (isSimulating) return;
    setIsSimulating(true);

    const formatTime = () => {
      const now = new Date();
      return now.toLocaleTimeString("en-US", { hour: "numeric", minute: "2-digit", hour12: true });
    };

    intervalRef.current = setInterval(() => {
      setTasks((currentTasks) => {
        const newTasks = [...currentTasks];
        let changed = false;

        // Progress planning phase for tasks with plannerPrompt
        newTasks.forEach((task, i) => {
          if (!task.plannerPrompt) return;

          // Start planning for tasks that have pending planStatus and can run
          if (task.planStatus === "pending" && canRun(task, newTasks)) {
            newTasks[i] = { ...task, planStatus: "in_progress" as TaskPlanStatus };
            changed = true;
            return;
          }

          // Transition from in_progress to approved/awaiting based on planApproval
          if (task.planStatus === "in_progress" && Math.random() > 0.7) {
            const nextStatus: TaskPlanStatus =
              task.planApproval === "auto" ? "approved" : "awaiting_approval";

            newTasks[i] = {
              ...task,
              planStatus: nextStatus,
              plan: {
                summary: `Generated plan for: ${task.name}`,
                steps: [
                  "Step 1: Analyze requirements",
                  "Step 2: Implement solution",
                  "Step 3: Test and verify",
                ],
                estimatedComplexity: "medium",
                agentId: task.planner?.id ?? "claude-opus-4",
                agentName: task.planner?.name ?? "Claude Opus 4",
                createdAt: formatTime(),
              },
            };
            changed = true;
            return;
          }

          // Auto-approve awaiting plans after a short delay (simulating overseer approval)
          if (task.planStatus === "awaiting_approval" && Math.random() > 0.6) {
            newTasks[i] = { ...task, planStatus: "approved" as TaskPlanStatus };
            changed = true;
          }
        });

        // Complete running tasks (random chance)
        newTasks.forEach((task, i) => {
          if (task.status === "running" && Math.random() > 0.6) {
            const startTime = task.startedAt
              ? new Date(`2000/01/01 ${task.startedAt}`)
              : new Date();
            const duration = Math.floor((Date.now() - startTime.getTime()) / 1000);
            const durationStr = `${Math.floor(duration / 60)}m ${duration % 60}s`;

            // Update the execution to completed
            const updatedExecutions = task.executions?.map((exec) =>
              exec.status === "running"
                ? {
                    ...exec,
                    status: "completed" as const,
                    duration: durationStr,
                    tokens: {
                      input: exec.tokens?.input ?? Math.floor(Math.random() * 1000) + 500,
                      output: Math.floor(Math.random() * 800) + 300,
                      total: 0,
                    },
                  }
                : exec,
            );
            // Calculate total tokens
            if (updatedExecutions) {
              updatedExecutions.forEach((exec) => {
                if (exec.tokens) {
                  exec.tokens.total = exec.tokens.input + exec.tokens.output;
                }
              });
            }

            newTasks[i] = {
              ...task,
              status: "completed",
              completedAt: formatTime(),
              duration: durationStr,
              executions: updatedExecutions,
              selectedExecutionId: updatedExecutions?.[0]?.id,
            };
            changed = true;
          }
        });

        // Start pending tasks that can run (and have completed planning if applicable)
        newTasks.forEach((task, i) => {
          // Skip tasks that are still in the planning phase
          if (task.plannerPrompt && task.planStatus !== "approved" && task.planStatus !== "skipped")
            return;

          if (canRun(task, newTasks) && newTasks.filter((t) => t.status === "running").length < 3) {
            // Create a new execution for this task
            executionCounterRef.current += 1;
            const agent = simulationAgents[Math.floor(Math.random() * simulationAgents.length)];
            const newExecution: TaskExecution = {
              id: `sim-exec-${executionCounterRef.current}`,
              agentId: agent.id,
              agentName: agent.name,
              status: "running",
              tokens: { input: Math.floor(Math.random() * 1000) + 500, output: 0, total: 0 },
            };

            newTasks[i] = {
              ...task,
              status: "running",
              startedAt: formatTime(),
              executions: [...(task.executions ?? []), newExecution],
            };
            changed = true;
          }
        });

        // Update blocked status
        newTasks.forEach((task, i) => {
          if (task.status === "pending") {
            const isBlocked = task.dependencies.some((depId) => {
              const dep = newTasks.find((t) => t.id === depId);
              return dep && dep.status !== "completed";
            });
            if (isBlocked && task.dependencies.length > 0) {
              const anyDepRunning = task.dependencies.some((depId) => {
                const dep = newTasks.find((t) => t.id === depId);
                return dep?.status === "running";
              });
              if (!anyDepRunning) {
                newTasks[i] = { ...task, status: "blocked" };
                changed = true;
              }
            }
          }
        });

        // Check if simulation is complete
        const allDone = newTasks.every((t) => t.status === "completed" || t.status === "failed");
        if (allDone && intervalRef.current) {
          clearInterval(intervalRef.current);
          intervalRef.current = null;
          setIsSimulating(false);
        }

        return changed ? newTasks : currentTasks;
      });
    }, 800);
  }, [isSimulating, canRun]);

  const pauseSimulation = useCallback(() => {
    setIsSimulating(false);
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
      intervalRef.current = null;
    }
  }, []);

  useEffect(() => {
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, []);

  useEffect(() => {
    setTasks(initialTasks);
  }, [initialTasks]);

  return { tasks, isSimulating, startSimulation, pauseSimulation, resetTasks };
}
