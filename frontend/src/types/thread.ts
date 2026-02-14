import type { ThreadResource } from "@/types/resource";
import type { ThreadVCSContext } from "@/types/vcs";

export type ThreadStatus = "draft" | "pending" | "in_progress" | "completed" | "failed";

export interface ThreadOverseer {
  id: string;
  name: string;
}

export interface Thread {
  id: string;
  topic: string;
  description: string;
  status: ThreadStatus;
  taskCount: number;
  completedTasks: number;
  createdAt: string;
  updatedAt: string;
  overseer: ThreadOverseer;
  memory?: string;
  resources?: ThreadResource[];
  vcs?: ThreadVCSContext;
  projectId: string;
  mode: "plan" | "build";
  model?: string;
  harness?: string;
  controlPlaneId?: string;
  archived?: boolean;
  plan?: string;
}
