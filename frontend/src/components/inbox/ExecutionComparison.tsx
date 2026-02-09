import { Badge } from "@/components/ui/badge";
import { GitCompare } from "lucide-react";
import type { TaskExecution } from "@/types/inbox";
import { ExecutionCard } from "./ExecutionCard";

interface ExecutionComparisonProps {
  title: string;
  taskName?: string;
  executions: TaskExecution[];
  selectedExecutionId?: string;
  onSelectExecution?: (executionId: string) => void;
}

export function ExecutionComparison({
  title,
  taskName,
  executions,
  selectedExecutionId,
  onSelectExecution,
}: ExecutionComparisonProps) {
  const completedCount = executions.filter((e) => e.status === "completed").length;
  const runningCount = executions.filter((e) => e.status === "running").length;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="space-y-2">
        <div className="flex items-center gap-3">
          <div className="rounded-lg bg-purple-500/10 p-2">
            <GitCompare className="size-5 text-purple-400" />
          </div>
          <div>
            <h2 className="text-lg font-semibold">{title}</h2>
            {taskName && <p className="text-sm text-muted-foreground">Task: {taskName}</p>}
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Badge variant="outline" className="text-xs">
            {executions.length} execution{executions.length !== 1 ? "s" : ""}
          </Badge>
          {completedCount > 0 && (
            <Badge className="bg-emerald-500/20 text-emerald-400 border-emerald-500/30 text-xs">
              {completedCount} completed
            </Badge>
          )}
          {runningCount > 0 && (
            <Badge className="bg-blue-500/20 text-blue-400 border-blue-500/30 text-xs">
              {runningCount} running
            </Badge>
          )}
        </div>
      </div>

      {/* Execution Grid */}
      <div className="grid gap-4 md:grid-cols-2">
        {executions.map((execution) => (
          <ExecutionCard
            key={execution.id}
            execution={execution}
            isSelected={execution.id === selectedExecutionId}
            onSelect={onSelectExecution}
            showSelectButton={!selectedExecutionId}
          />
        ))}
      </div>

      {/* Empty State */}
      {executions.length === 0 && (
        <div className="flex flex-col items-center justify-center py-12 text-center">
          <GitCompare className="size-12 text-muted-foreground/50 mb-4" />
          <p className="text-muted-foreground">No executions yet</p>
        </div>
      )}
    </div>
  );
}
