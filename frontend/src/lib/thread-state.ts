import type { ThreadConfig } from "@/proto/gen/controlplane/v1/thread_service_pb";
import type { Task } from "@/types/task";

export type ThreadViewState = "chat" | "planning" | "executing";

export interface ParsedPlanTask {
  id: string;
  name: string;
  description: string;
  dependencies: string[];
  estimatedDuration?: string;
  agent?: string;
  subtasks?: string[];
}

export interface ParsedPlan {
  agentId?: string;
  agentName?: string;
  agentModel?: string;
  summary: string;
  approach: string;
  tasks: ParsedPlanTask[];
  considerations: string[];
  estimatedComplexity: "low" | "medium" | "high";
  estimatedDuration?: string;
  affectedFiles?: string[];
}

export function deriveThreadViewState(
  thread: ThreadConfig | undefined,
  tasks: Task[],
): ThreadViewState {
  if (tasks.length > 0) return "executing";
  if (thread?.plan) return "planning";
  return "chat";
}

export function parsePlan(json: string): ParsedPlan | null {
  try {
    const parsed = JSON.parse(json);
    if (!parsed || typeof parsed !== "object") return null;
    return parsed as ParsedPlan;
  } catch {
    return null;
  }
}
