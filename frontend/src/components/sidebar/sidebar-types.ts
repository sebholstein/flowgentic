import type { ThreadConfig } from "@/proto/gen/controlplane/v1/thread_service_pb";
import type { Project } from "@/types/project";
import type { Task } from "@/types/task";

export type FlatTreeNode =
  | { type: "project"; project: Project; threadCount: number; isDemo?: boolean }
  | { type: "thread"; thread: ThreadConfig; projectId: string; hasChildren: boolean; isDemo?: boolean }
  | { type: "task"; task: Task; threadId: string };

export type SidebarTab = "threads" | "archived";
export type SidebarView = "threads" | "templates";
