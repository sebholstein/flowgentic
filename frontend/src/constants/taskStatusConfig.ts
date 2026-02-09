import {
  CheckCircle2,
  Circle,
  Clock,
  AlertCircle,
  Loader2,
  MessageCircle,
  ClipboardList,
  ClipboardCheck,
  ClipboardX,
} from "lucide-react";
import type { TaskStatus, TaskPlanStatus } from "@/types/task";

export const taskStatusConfig: Record<
  TaskStatus,
  { icon: typeof Circle; color: string; bgColor: string; label: string }
> = {
  pending: { icon: Circle, color: "text-slate-400", bgColor: "bg-slate-400/10", label: "Pending" },
  running: { icon: Loader2, color: "text-blue-400", bgColor: "bg-blue-400/10", label: "Running" },
  completed: {
    icon: CheckCircle2,
    color: "text-emerald-400",
    bgColor: "bg-emerald-400/10",
    label: "Completed",
  },
  failed: { icon: AlertCircle, color: "text-red-400", bgColor: "bg-red-400/10", label: "Failed" },
  blocked: { icon: Clock, color: "text-amber-400", bgColor: "bg-amber-400/10", label: "Blocked" },
  needs_feedback: {
    icon: MessageCircle,
    color: "text-purple-400",
    bgColor: "bg-purple-400/10",
    label: "Needs Feedback",
  },
};

export const planStatusConfig: Record<
  TaskPlanStatus,
  { icon: typeof Circle | null; color: string; bgColor: string; label: string }
> = {
  pending: {
    icon: ClipboardList,
    color: "text-slate-400",
    bgColor: "bg-slate-400/10",
    label: "Plan pending",
  },
  in_progress: {
    icon: ClipboardList,
    color: "text-blue-400",
    bgColor: "bg-blue-400/10",
    label: "Planning...",
  },
  awaiting_approval: {
    icon: ClipboardList,
    color: "text-orange-400",
    bgColor: "bg-orange-400/10",
    label: "Plan ready",
  },
  approved: {
    icon: ClipboardCheck,
    color: "text-emerald-400",
    bgColor: "bg-emerald-400/10",
    label: "Plan approved",
  },
  rejected: {
    icon: ClipboardX,
    color: "text-red-400",
    bgColor: "bg-red-400/10",
    label: "Plan rejected",
  },
  skipped: { icon: null, color: "", bgColor: "", label: "" },
};
