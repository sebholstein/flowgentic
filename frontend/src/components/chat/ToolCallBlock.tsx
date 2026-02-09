import { useState } from "react";
import { cn } from "@/lib/utils";
import {
  FileText,
  FileCode,
  FolderOpen,
  Terminal,
  Search,
  Edit3,
  Plus,
  CheckCircle2,
  XCircle,
  Loader2,
  ChevronDown,
  ChevronRight,
  Globe,
} from "lucide-react";

export type ToolType =
  | "read_file"
  | "write_file"
  | "edit_file"
  | "list_directory"
  | "bash"
  | "search"
  | "web_search"
  | "create_file";

export type ToolStatus = "running" | "success" | "error";

export interface ToolCall {
  id: string;
  type: ToolType;
  status: ToolStatus;
  title: string;
  subtitle?: string;
  input?: Record<string, unknown>;
  output?: string;
  error?: string;
  duration?: number;
}

interface ToolCallBlockProps {
  tool: ToolCall;
  className?: string;
}

const toolIcons: Record<ToolType, typeof FileText> = {
  read_file: FileText,
  write_file: FileCode,
  edit_file: Edit3,
  create_file: Plus,
  list_directory: FolderOpen,
  bash: Terminal,
  search: Search,
  web_search: Globe,
};

const toolColors: Record<ToolType, string> = {
  read_file: "text-blue-500 bg-blue-500/10",
  write_file: "text-green-500 bg-green-500/10",
  edit_file: "text-amber-500 bg-amber-500/10",
  create_file: "text-emerald-500 bg-emerald-500/10",
  list_directory: "text-purple-500 bg-purple-500/10",
  bash: "text-orange-500 bg-orange-500/10",
  search: "text-cyan-500 bg-cyan-500/10",
  web_search: "text-indigo-500 bg-indigo-500/10",
};

export function ToolCallBlock({ tool, className }: ToolCallBlockProps) {
  const [expanded, setExpanded] = useState(false);
  const Icon = toolIcons[tool.type];
  const colorClass = toolColors[tool.type];

  const hasDetails = tool.output || tool.error || tool.input;

  return (
    <div
      className={cn(
        "rounded-lg border bg-card text-card-foreground",
        tool.status === "error" && "border-red-500/30",
        className,
      )}
    >
      <button
        onClick={() => hasDetails && setExpanded(!expanded)}
        disabled={!hasDetails}
        className={cn(
          "flex items-center gap-2 w-full px-3 py-2 text-left",
          hasDetails && "hover:bg-muted/50 cursor-pointer",
          !hasDetails && "cursor-default",
        )}
      >
        {/* Icon */}
        <div className={cn("rounded-md p-1.5", colorClass)}>
          <Icon className="size-3.5" />
        </div>

        {/* Title and subtitle */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="text-xs font-medium truncate">{tool.title}</span>
            {tool.duration && tool.status !== "running" && (
              <span className="text-[10px] text-muted-foreground tabular-nums">
                {tool.duration}ms
              </span>
            )}
          </div>
          {tool.subtitle && (
            <p className="text-[10px] text-muted-foreground truncate">{tool.subtitle}</p>
          )}
        </div>

        {/* Status indicator */}
        <div className="shrink-0">
          {tool.status === "running" && <Loader2 className="size-3.5 text-blue-500 animate-spin" />}
          {tool.status === "success" && <CheckCircle2 className="size-3.5 text-emerald-500" />}
          {tool.status === "error" && <XCircle className="size-3.5 text-red-500" />}
        </div>

        {/* Expand indicator */}
        {hasDetails && (
          <div className="shrink-0 text-muted-foreground">
            {expanded ? (
              <ChevronDown className="size-3.5" />
            ) : (
              <ChevronRight className="size-3.5" />
            )}
          </div>
        )}
      </button>

      {/* Expanded content */}
      {expanded && hasDetails && (
        <div className="border-t px-3 py-2 space-y-2">
          {tool.input && (
            <div>
              <div className="text-[10px] font-medium text-muted-foreground uppercase tracking-wider mb-1">
                Input
              </div>
              <pre className="text-[10px] bg-muted rounded p-2 overflow-x-auto">
                {JSON.stringify(tool.input, null, 2)}
              </pre>
            </div>
          )}
          {tool.output && (
            <div>
              <div className="text-[10px] font-medium text-muted-foreground uppercase tracking-wider mb-1">
                Output
              </div>
              <pre className="text-[10px] bg-muted rounded p-2 overflow-x-auto max-h-32 overflow-y-auto">
                {tool.output}
              </pre>
            </div>
          )}
          {tool.error && (
            <div>
              <div className="text-[10px] font-medium text-red-500 uppercase tracking-wider mb-1">
                Error
              </div>
              <pre className="text-[10px] bg-red-500/10 text-red-500 rounded p-2 overflow-x-auto">
                {tool.error}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
