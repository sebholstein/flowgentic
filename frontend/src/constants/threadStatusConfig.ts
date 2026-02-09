import { CheckCircle2, Circle, AlertCircle, PlayCircle, FileEdit } from "lucide-react";
import type { ThreadStatus } from "@/types/thread";

export const threadStatusConfig: Record<
  ThreadStatus,
  { icon: typeof Circle; color: string; label: string }
> = {
  draft: { icon: FileEdit, color: "text-slate-400", label: "Draft" },
  pending: { icon: Circle, color: "text-muted-foreground", label: "Pending" },
  in_progress: { icon: PlayCircle, color: "text-blue-400", label: "In Progress" },
  completed: { icon: CheckCircle2, color: "text-emerald-400", label: "Completed" },
  failed: { icon: AlertCircle, color: "text-red-400", label: "Failed" },
};

// Legacy export for backwards compatibility
export const issueStatusConfig = threadStatusConfig;
