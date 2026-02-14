import { createFileRoute } from "@tanstack/react-router";
import { useMemo } from "react";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Markdown } from "@/components/ui/markdown";
import { Clock, Calendar, Layers, Activity, Zap, Users, User } from "lucide-react";
import { useThreadContext } from "./route";

export const Route = createFileRoute("/app/threads/$threadId/")({
  component: ThreadOverviewTab,
});

const threadStatusConfig: Record<string, { color: string; bgColor: string; label: string }> = {
  draft: { color: "text-slate-400", bgColor: "bg-slate-400/10", label: "Draft" },
  pending: { color: "text-slate-400", bgColor: "bg-slate-400/10", label: "Pending" },
  in_progress: { color: "text-blue-400", bgColor: "bg-blue-400/10", label: "In Progress" },
  completed: { color: "text-emerald-400", bgColor: "bg-emerald-400/10", label: "Completed" },
  failed: { color: "text-red-400", bgColor: "bg-red-400/10", label: "Failed" },
};

function ThreadOverviewTab() {
  const { thread, tasks } = useThreadContext();
  const threadStatus = threadStatusConfig["pending"];
  const completedCount = tasks.filter((t) => t.status === "completed").length;
  const runningCount = tasks.filter((t) => t.status === "running").length;
  const needsFeedbackCount = tasks.filter((t) => t.status === "needs_feedback").length;
  const blockedCount = tasks.filter((t) => t.status === "blocked").length;
  const failedCount = tasks.filter((t) => t.status === "failed").length;
  const pendingCount = tasks.filter((t) => t.status === "pending").length;
  const planningCount = tasks.filter(
    (t) => t.planStatus === "in_progress" || t.planStatus === "awaiting_approval",
  ).length;
  const progress = tasks.length > 0 ? Math.round((completedCount / tasks.length) * 100) : 0;

  // Calculate total tokens across all executions
  const totalTokens = useMemo(() => {
    let input = 0;
    let output = 0;
    tasks.forEach((task) => {
      task.executions?.forEach((exec) => {
        if (exec.tokens) {
          input += exec.tokens.input;
          output += exec.tokens.output;
        }
      });
    });
    return { input, output, total: input + output };
  }, [tasks]);

  return (
    <ScrollArea className="h-full">
      <div className="max-w-4xl mx-auto p-8">
        {/* Thread Header */}
        <div className="mb-8">
          <div className="flex items-center gap-2 text-muted-foreground text-sm mb-2">
            <Badge className={cn("text-xs", threadStatus.bgColor, threadStatus.color)}>
              {threadStatus.label}
            </Badge>
            <Badge
              variant="outline"
              className={cn(
                "text-xs gap-1",
                thread.mode === "build"
                  ? "text-slate-400 border-slate-500/30"
                  : "text-violet-400 border-violet-500/30",
              )}
            >
              {thread.mode === "build" ? <User className="size-3" /> : <Users className="size-3" />}
              {thread.mode === "build" ? "Build" : "Plan"}
            </Badge>
          </div>
          <h1 className="text-3xl font-bold tracking-tight mb-3">{thread.topic}</h1>
          <Markdown>{""}</Markdown>
        </div>

        {/* Thread Properties - Compact */}
        <div className="rounded-lg border border-border bg-muted/50 px-4 py-3 mb-6">
          <div className="flex flex-wrap items-center gap-x-6 gap-y-2 text-sm">
            {/* Status */}
            <div className="flex items-center gap-2">
              <Activity className="size-3.5 text-muted-foreground" />
              <Badge className={cn("text-xs", threadStatus.bgColor, threadStatus.color)}>
                {threadStatus.label}
              </Badge>
            </div>

            {/* Progress */}
            <div className="flex items-center gap-2">
              <Layers className="size-3.5 text-muted-foreground" />
              <div className="w-20 h-1.5 bg-muted rounded-full overflow-hidden">
                <div
                  className={cn(
                    "h-full rounded-full transition-all",
                    progress === 100 ? "bg-emerald-500" : "bg-blue-500",
                  )}
                  style={{ width: `${progress}%` }}
                />
              </div>
              <span className="text-xs tabular-nums">{progress}%</span>
              <span className="text-xs text-muted-foreground">
                ({completedCount}/{tasks.length})
              </span>
            </div>

            {/* Tokens */}
            {totalTokens.total > 0 && (
              <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                <Zap className="size-3.5" />
                <span className="tabular-nums">{totalTokens.total.toLocaleString()}</span>
                <span>tokens</span>
                <span className="text-muted-foreground/70">
                  ({totalTokens.input.toLocaleString()}↓ {totalTokens.output.toLocaleString()}↑)
                </span>
              </div>
            )}

            {/* Dates */}
            <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <Calendar className="size-3.5" />
              <span>{thread.createdAt}</span>
            </div>
            <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <Clock className="size-3.5" />
              <span>{thread.updatedAt}</span>
            </div>

          </div>

          {/* Task Status Summary - Inline */}
          <div className="flex flex-wrap items-center gap-3 mt-2 pt-2 border-t border-border/50 text-xs">
            {completedCount > 0 && (
              <span className="flex items-center gap-1">
                <span className="size-2 rounded-full bg-emerald-500" />
                <span className="text-emerald-400 font-medium">{completedCount}</span>
                <span className="text-muted-foreground">done</span>
              </span>
            )}
            {runningCount > 0 && (
              <span className="flex items-center gap-1">
                <span className="size-2 rounded-full bg-blue-500 animate-pulse" />
                <span className="text-blue-400 font-medium">{runningCount}</span>
                <span className="text-muted-foreground">running</span>
              </span>
            )}
            {needsFeedbackCount > 0 && (
              <span className="flex items-center gap-1">
                <span className="size-2 rounded-full bg-purple-500" />
                <span className="text-purple-400 font-medium">{needsFeedbackCount}</span>
                <span className="text-muted-foreground">feedback</span>
              </span>
            )}
            {blockedCount > 0 && (
              <span className="flex items-center gap-1">
                <span className="size-2 rounded-full bg-amber-500" />
                <span className="text-amber-400 font-medium">{blockedCount}</span>
                <span className="text-muted-foreground">blocked</span>
              </span>
            )}
            {failedCount > 0 && (
              <span className="flex items-center gap-1">
                <span className="size-2 rounded-full bg-red-500" />
                <span className="text-red-400 font-medium">{failedCount}</span>
                <span className="text-muted-foreground">failed</span>
              </span>
            )}
            {pendingCount > 0 && (
              <span className="flex items-center gap-1">
                <span className="size-2 rounded-full bg-slate-500" />
                <span className="text-slate-400 font-medium">{pendingCount}</span>
                <span className="text-muted-foreground">pending</span>
              </span>
            )}
            {planningCount > 0 && (
              <span className="flex items-center gap-1">
                <span className="size-2 rounded-full bg-blue-500" />
                <span className="text-blue-400 font-medium">{planningCount}</span>
                <span className="text-muted-foreground">planning</span>
              </span>
            )}
          </div>
        </div>

      </div>
    </ScrollArea>
  );
}
