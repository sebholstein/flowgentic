import { cn } from "@/lib/utils";
import type { ConnectionStatus } from "@/types/server";

const statusColors: Record<ConnectionStatus, string> = {
  connecting: "bg-amber-500 animate-pulse",
  connected: "bg-emerald-500",
  disconnected: "bg-slate-400",
  error: "bg-red-500",
};

export function ServerStatusDot({
  status,
  className,
}: {
  status: ConnectionStatus;
  className?: string;
}) {
  return (
    <span className={cn("size-2 rounded-full inline-block", statusColors[status], className)} />
  );
}
