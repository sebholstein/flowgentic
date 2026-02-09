import { createContext, use } from "react";
import type { Task } from "@/types/task";
import type { Thread } from "@/types/thread";
import type { ProgressStep } from "@/data/mockTaskData";

export interface TaskContextValue {
  task: Task;
  thread: Thread;
  progressSteps: ProgressStep[];
  allTasks: Task[];
}

export const TaskContext = createContext<TaskContextValue | null>(null);

export function useTaskContext() {
  const context = use(TaskContext);
  if (!context) {
    throw new Error("useTaskContext must be used within TaskLayout");
  }
  return context;
}
