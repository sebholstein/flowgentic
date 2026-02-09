import { createFileRoute, Link } from "@tanstack/react-router";
import { useMemo } from "react";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { ExternalLink, AlertTriangle, Compass, MessageCircle } from "lucide-react";
import { useThreadContext } from "./route";

export const Route = createFileRoute("/app/threads/$threadId/checkins")({
  component: CheckinsTab,
});

function CheckinsTab() {
  const { tasks, onSelectTask } = useThreadContext();

  // Get tasks with pending check-ins
  const pendingCheckIns = useMemo(() => {
    return tasks.filter((task) => task.checkIn);
  }, [tasks]);

  if (pendingCheckIns.length === 0) {
    return (
      <div className="h-full flex items-center justify-center text-muted-foreground">
        No pending check-ins.
      </div>
    );
  }

  return (
    <ScrollArea className="h-full">
      <div className="max-w-4xl mx-auto p-8">
        <div className="space-y-3">
          <p className="text-sm text-muted-foreground">
            These tasks have pending check-ins that need your attention.
          </p>
          {pendingCheckIns.map((task) => (
            <div
              key={task.id}
              className={cn(
                "rounded-lg border p-4",
                task.checkIn?.type === "decision_escalation" &&
                  "border-amber-500/30 bg-amber-500/5",
                task.checkIn?.type === "direction_clarification" &&
                  "border-blue-500/30 bg-blue-500/5",
                !["decision_escalation", "direction_clarification"].includes(
                  task.checkIn?.type ?? "",
                ) && "border-purple-500/30 bg-purple-500/5",
              )}
            >
              <div className="flex items-start justify-between gap-3">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    {task.checkIn?.type === "decision_escalation" && (
                      <AlertTriangle className="size-4 text-amber-400" />
                    )}
                    {task.checkIn?.type === "direction_clarification" && (
                      <Compass className="size-4 text-blue-400" />
                    )}
                    {!["decision_escalation", "direction_clarification"].includes(
                      task.checkIn?.type ?? "",
                    ) && <MessageCircle className="size-4 text-purple-400" />}
                    <span className="font-medium text-sm">{task.checkIn?.title}</span>
                    <Badge
                      variant="outline"
                      className={cn(
                        "text-[0.6rem]",
                        task.checkIn?.priority === "high" && "text-red-400 border-red-500/30",
                        task.checkIn?.priority === "medium" && "text-amber-400 border-amber-500/30",
                        task.checkIn?.priority === "low" && "text-slate-400 border-slate-500/30",
                      )}
                    >
                      {task.checkIn?.priority}
                    </Badge>
                  </div>
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <span>Task:</span>
                    <button
                      onClick={() => onSelectTask(task.id)}
                      className="text-foreground hover:underline"
                    >
                      {task.name}
                    </button>
                  </div>
                </div>
                <Link
                  to="/app/inbox/$itemId"
                  params={{ itemId: task.checkIn?.id ?? "" }}
                  className={cn(
                    "flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-md border transition-colors",
                    task.checkIn?.type === "decision_escalation" &&
                      "border-amber-500/30 text-amber-400 hover:bg-amber-500/10",
                    task.checkIn?.type === "direction_clarification" &&
                      "border-blue-500/30 text-blue-400 hover:bg-blue-500/10",
                    !["decision_escalation", "direction_clarification"].includes(
                      task.checkIn?.type ?? "",
                    ) && "border-purple-500/30 text-purple-400 hover:bg-purple-500/10",
                  )}
                >
                  {task.checkIn?.type === "decision_escalation"
                    ? "Make Decision"
                    : task.checkIn?.type === "direction_clarification"
                      ? "Provide Guidance"
                      : "Respond"}
                  <ExternalLink className="size-3" />
                </Link>
              </div>
            </div>
          ))}
        </div>
      </div>
    </ScrollArea>
  );
}
