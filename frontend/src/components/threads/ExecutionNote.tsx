import { memo } from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { cn } from "@/lib/utils";
import { CheckCircle2, Loader2, AlertCircle, Coins, Zap } from "lucide-react";

interface ExecutionData {
  id: string;
  agentId: string;
  agentName: string;
  status: "running" | "completed" | "failed";
  duration?: string;
  tokens?: {
    input: number;
    output: number;
    total: number;
  };
  isSelected?: boolean;
}

export interface ExecutionNoteData {
  execution: ExecutionData;
  taskId: string;
}

const agentColors: Record<string, string> = {
  "claude-opus-4": "bg-orange-500",
  "claude-sonnet": "bg-amber-500",
  "gpt-4": "bg-emerald-500",
  "gpt-4o": "bg-teal-500",
  "gemini-pro": "bg-blue-500",
};

const statusConfig = {
  running: { icon: Loader2, color: "text-blue-400", borderColor: "border-blue-500/50" },
  completed: {
    icon: CheckCircle2,
    color: "text-emerald-400",
    borderColor: "border-emerald-500/50",
  },
  failed: { icon: AlertCircle, color: "text-red-400", borderColor: "border-red-500/50" },
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

function ExecutionNoteComponent({ data }: NodeProps<ExecutionNoteData>) {
  const { execution } = data;
  const status = statusConfig[execution.status];
  const StatusIcon = status.icon;
  const agentColor = agentColors[execution.agentId] ?? "bg-muted-foreground";

  return (
    <>
      <Handle
        type="target"
        position={Position.Left}
        className="!w-2 !h-2 !bg-muted-foreground !border !border-border !-left-1"
      />
      <div
        className={cn(
          "flex items-center gap-2 rounded-md border bg-card/95 backdrop-blur px-2.5 py-1.5 shadow-sm",
          status.borderColor,
          execution.isSelected && "ring-2 ring-emerald-500/50",
        )}
      >
        {/* Execution indicator */}
        <div className="flex items-center justify-center size-4 rounded bg-muted shrink-0">
          <Zap className="size-2.5 text-amber-400 fill-amber-400" />
        </div>

        {/* Agent Icon */}
        <div
          className={cn(
            "size-5 rounded flex items-center justify-center text-[0.5rem] font-bold text-white shrink-0",
            agentColor,
          )}
        >
          {execution.agentId[0]?.toUpperCase()}
        </div>

        {/* Info */}
        <div className="flex flex-col min-w-0">
          <span className="text-[0.6rem] font-medium text-foreground truncate max-w-[80px]">
            {execution.agentName}
          </span>
          <div className="flex items-center gap-1.5 text-[0.5rem] text-muted-foreground">
            <StatusIcon
              className={cn(
                "size-2.5",
                status.color,
                execution.status === "running" && "animate-spin",
              )}
            />
            {execution.duration && <span>{execution.duration}</span>}
            {execution.tokens && (
              <>
                <span className="opacity-50">•</span>
                <Coins className="size-2.5" />
                <span>{formatTokens(execution.tokens.total)}</span>
              </>
            )}
          </div>
        </div>

        {/* Selected indicator */}
        {execution.isSelected && <CheckCircle2 className="size-3 text-emerald-500 shrink-0" />}
      </div>
    </>
  );
}

export const ExecutionNote = memo(ExecutionNoteComponent);
