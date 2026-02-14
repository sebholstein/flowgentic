import { Badge } from "@/components/ui/badge";

interface FlowgenticPlanToolResultProps {
  toolName: string;
  status: "running" | "success" | "error";
  result: Record<string, unknown> | null;
}

const toolActionLabels: Record<string, string> = {
  set_topic: "Set Topic",
  plan_get_current_dir: "Get Plan Directory",
  plan_request_thread_dir: "Allocate Thread Directory",
  plan_remove_thread: "Remove Thread Directory",
  plan_clear_current: "Clear Current Plan",
  plan_commit: "Commit Plans",
};

function getSummary(toolName: string, result: Record<string, unknown> | null): string | null {
  if (!result) return null;

  if (toolName === "set_topic" && typeof result.topic === "string") {
    return result.topic;
  }
  if (toolName === "plan_get_current_dir" && typeof result.plan_dir === "string") {
    return result.plan_dir;
  }
  if (toolName === "plan_request_thread_dir") {
    const threadId = typeof result.thread_id === "string" ? result.thread_id : null;
    const planDir = typeof result.plan_dir === "string" ? result.plan_dir : null;
    if (threadId && planDir) return `${threadId} -> ${planDir}`;
  }
  if (toolName === "plan_remove_thread" && typeof result.thread_id === "string") {
    return result.thread_id;
  }
  if (toolName === "plan_clear_current" && result.cleared === true) {
    return "Current plan directory cleared";
  }
  if (toolName === "plan_commit" && typeof result.submitted_plans === "number") {
    return `${result.submitted_plans} plan dir(s) submitted`;
  }

  return null;
}

export function FlowgenticPlanToolResult({ toolName, status, result }: FlowgenticPlanToolResultProps) {
  const actionLabel = toolActionLabels[toolName] ?? toolName;
  const summary = getSummary(toolName, result);

  return (
    <div className="flex items-center gap-2 text-xs">
      <Badge variant="outline" className="h-5 px-1.5 text-[10px]">
        Flowgentic
      </Badge>
      <span className="text-muted-foreground">{actionLabel}</span>
      {status === "running" && <span className="text-blue-400">running...</span>}
      {status === "success" && summary && (
        <code className="font-mono text-foreground/85 truncate">{summary}</code>
      )}
    </div>
  );
}
