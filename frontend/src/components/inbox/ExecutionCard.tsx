import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import {
  CheckCircle2,
  Loader2,
  AlertCircle,
  ChevronDown,
  ThumbsUp,
  ThumbsDown,
  Clock,
  Code,
  Coins,
} from "lucide-react";
import type { TaskExecution } from "@/types/inbox";
import { useState } from "react";

// Agent icon mapping
const agentIcons: Record<string, { icon: string; color: string }> = {
  "claude-opus-4": { icon: "C", color: "bg-orange-500" },
  "claude-sonnet": { icon: "C", color: "bg-amber-500" },
  "gpt-4": { icon: "G", color: "bg-emerald-500" },
  "gpt-4o": { icon: "G", color: "bg-teal-500" },
  "gemini-pro": { icon: "G", color: "bg-blue-500" },
  default: { icon: "A", color: "bg-slate-500" },
};

function getAgentIcon(agentName: string) {
  return agentIcons[agentName] ?? agentIcons.default;
}

const statusConfig = {
  running: { icon: Loader2, color: "text-blue-400", bgColor: "bg-blue-400/10", label: "Running" },
  completed: {
    icon: CheckCircle2,
    color: "text-emerald-400",
    bgColor: "bg-emerald-400/10",
    label: "Completed",
  },
  failed: { icon: AlertCircle, color: "text-red-400", bgColor: "bg-red-400/10", label: "Failed" },
};

function formatTokens(count: number): string {
  if (count >= 1000000) {
    return `${(count / 1000000).toFixed(1)}M`;
  }
  if (count >= 1000) {
    return `${(count / 1000).toFixed(1)}k`;
  }
  return count.toString();
}

interface ExecutionCardProps {
  execution: TaskExecution;
  onSelect?: (executionId: string) => void;
  isSelected?: boolean;
  showSelectButton?: boolean;
}

export function ExecutionCard({
  execution,
  onSelect,
  isSelected = false,
  showSelectButton = true,
}: ExecutionCardProps) {
  const [codeOpen, setCodeOpen] = useState(false);
  const status = statusConfig[execution.status];
  const StatusIcon = status.icon;
  const agentIcon = getAgentIcon(execution.agentId);

  return (
    <div
      className={cn(
        "rounded-lg border bg-muted/60 transition-all",
        isSelected && "ring-2 ring-emerald-500 border-emerald-500/50",
        execution.status === "running" && "border-blue-500/50",
        execution.status === "failed" && "border-red-500/30",
      )}
    >
      {/* Header */}
      <div className="flex items-center justify-between gap-3 p-4 border-b border-border/50">
        <div className="flex items-center gap-3">
          <div
            className={cn(
              "size-9 rounded-lg flex items-center justify-center text-sm font-bold text-white",
              agentIcon.color,
            )}
          >
            {agentIcon.icon}
          </div>
          <div>
            <div className="font-medium text-sm">{execution.agentName}</div>
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <div className={cn("flex items-center gap-1", status.color)}>
                <StatusIcon
                  className={cn("size-3", execution.status === "running" && "animate-spin")}
                />
                <span>{status.label}</span>
              </div>
              {execution.duration && (
                <>
                  <span className="text-muted-foreground/70">•</span>
                  <div className="flex items-center gap-1">
                    <Clock className="size-3" />
                    <span>{execution.duration}</span>
                  </div>
                </>
              )}
              {execution.tokens && (
                <>
                  <span className="text-muted-foreground/70">•</span>
                  <div
                    className="flex items-center gap-1"
                    title={`Input: ${execution.tokens.input.toLocaleString()} | Output: ${execution.tokens.output.toLocaleString()}`}
                  >
                    <Coins className="size-3" />
                    <span>{formatTokens(execution.tokens.total)} tokens</span>
                  </div>
                </>
              )}
            </div>
          </div>
        </div>
        {isSelected && (
          <Badge className="bg-emerald-500/20 text-emerald-400 border-emerald-500/30">
            Selected
          </Badge>
        )}
      </div>

      {/* Summary */}
      <div className="p-4 space-y-3">
        <p className="text-sm text-foreground leading-relaxed">{execution.output.summary}</p>

        {/* Code preview */}
        {execution.output.code && (
          <Collapsible open={codeOpen} onOpenChange={setCodeOpen}>
            <CollapsibleTrigger className="flex items-center gap-2 text-xs text-muted-foreground hover:text-foreground transition-colors">
              <Code className="size-3.5" />
              <span>View code</span>
              <ChevronDown
                className={cn("size-3.5 transition-transform", codeOpen && "rotate-180")}
              />
            </CollapsibleTrigger>
            <CollapsibleContent>
              <pre className="mt-3 p-3 bg-muted rounded-md text-xs text-foreground overflow-x-auto max-h-48 overflow-y-auto">
                <code>{execution.output.code}</code>
              </pre>
            </CollapsibleContent>
          </Collapsible>
        )}

        {/* Evaluation */}
        {execution.evaluation && (
          <div className="pt-3 border-t border-border/50 space-y-2">
            {execution.evaluation.pros && execution.evaluation.pros.length > 0 && (
              <div className="space-y-1">
                {execution.evaluation.pros.map((pro, i) => (
                  <div key={i} className="flex items-start gap-2 text-xs">
                    <ThumbsUp className="size-3 text-emerald-400 mt-0.5 shrink-0" />
                    <span className="text-muted-foreground">{pro}</span>
                  </div>
                ))}
              </div>
            )}
            {execution.evaluation.cons && execution.evaluation.cons.length > 0 && (
              <div className="space-y-1">
                {execution.evaluation.cons.map((con, i) => (
                  <div key={i} className="flex items-start gap-2 text-xs">
                    <ThumbsDown className="size-3 text-red-400 mt-0.5 shrink-0" />
                    <span className="text-muted-foreground">{con}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>

      {/* Actions */}
      {showSelectButton && execution.status === "completed" && !isSelected && (
        <div className="px-4 pb-4">
          <Button size="sm" className="w-full" onClick={() => onSelect?.(execution.id)}>
            Select This Execution
          </Button>
        </div>
      )}
    </div>
  );
}
