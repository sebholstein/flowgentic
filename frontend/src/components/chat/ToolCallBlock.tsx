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
  | "create_file"
  | "mcp";

export type ToolStatus = "running" | "success" | "error";

export interface ToolCall {
  id: string;
  type: ToolType;
  status: ToolStatus;
  title: string;
  subtitle?: string;
  input?: Record<string, unknown>;
  output?: string;
  content?: string;
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
  mcp: Globe,
};

const toolLabels: Record<ToolType, string> = {
  read_file: "Read",
  write_file: "Write",
  edit_file: "Edit",
  create_file: "Create",
  list_directory: "List",
  bash: "Run",
  search: "Search",
  web_search: "Search",
  mcp: "MCP",
};

const runningLabels: Record<ToolType, string> = {
  read_file: "Reading",
  write_file: "Writing",
  edit_file: "Editing",
  create_file: "Creating",
  list_directory: "Listing",
  bash: "Running",
  search: "Searching",
  web_search: "Searching",
  mcp: "Calling",
};

const toolIconColors: Record<ToolType, string> = {
  read_file: "text-blue-400",
  write_file: "text-green-400",
  edit_file: "text-amber-400",
  create_file: "text-emerald-400",
  list_directory: "text-purple-400",
  bash: "text-muted-foreground",
  search: "text-cyan-400",
  web_search: "text-indigo-400",
  mcp: "text-violet-400",
};

/**
 * Extract the most meaningful display text for a tool call.
 * Prefers extracted detail (file path, command) over the raw title.
 */
function getDisplayDetail(tool: ToolCall): string {
  // For file tools: show the file path
  if (
    tool.type === "read_file" ||
    tool.type === "write_file" ||
    tool.type === "edit_file" ||
    tool.type === "create_file"
  ) {
    // Try input.filePath or input.path first
    if (tool.input) {
      const path = tool.input.filePath ?? tool.input.path ?? tool.input.file_path;
      if (typeof path === "string") return path;
    }
    // Try extracting path from title like "Read /path/to/file.ts"
    const match = tool.title.match(/^(?:Read|Write|Edit|Create)\s+(.+)$/i);
    if (match) return match[1];
  }

  // For bash tools: show the command
  if (tool.type === "bash") {
    if (tool.input && typeof tool.input.command === "string") {
      return tool.input.command;
    }
    if (tool.subtitle) return tool.subtitle;
    // Try extracting from title
    const colonIdx = tool.title.indexOf(":");
    if (colonIdx > 0) return tool.title.slice(colonIdx + 1).trim();
  }

  // For search tools: show the pattern/query
  if (tool.type === "search" || tool.type === "web_search") {
    if (tool.input) {
      const pattern = tool.input.pattern ?? tool.input.query ?? tool.input.regex;
      if (typeof pattern === "string") return pattern;
    }
    const match = tool.title.match(/^(?:Search|Find|Grep)\s+(?:for\s+)?(.+)$/i);
    if (match) return match[1];
  }

  // For list_directory
  if (tool.type === "list_directory") {
    if (tool.input) {
      const path = tool.input.path ?? tool.input.directory;
      if (typeof path === "string") return path;
    }
  }

  if (tool.type === "mcp") {
    if (tool.title) return tool.title;
    return "MCP tool";
  }

  return tool.subtitle ?? tool.title;
}

/** Shorten a file path for the compact header: show last 2-3 segments. */
function shortenPath(path: string): string {
  const parts = path.split("/").filter(Boolean);
  if (parts.length <= 3) return path;
  return ".../" + parts.slice(-3).join("/");
}

export function ToolCallBlock({ tool, className }: ToolCallBlockProps) {
  const [expanded, setExpanded] = useState(false);
  const Icon = toolIcons[tool.type];
  const iconColor = toolIconColors[tool.type];
  const label = tool.status === "running" ? runningLabels[tool.type] : toolLabels[tool.type];
  const detail = getDisplayDetail(tool);
  const isFileTool =
    tool.type === "read_file" ||
    tool.type === "write_file" ||
    tool.type === "edit_file" ||
    tool.type === "create_file" ||
    tool.type === "list_directory";

  const hasExpandableContent = tool.output || tool.error || tool.content;
  const isExpandable = !!hasExpandableContent;

  return (
    <div className={cn("group", className)}>
      {/* Header line */}
      <button
        onClick={() => isExpandable && setExpanded(!expanded)}
        disabled={!isExpandable}
        className={cn(
          "flex items-center gap-1.5 w-full text-left py-0.5",
          isExpandable && "hover:brightness-125 cursor-pointer",
          !isExpandable && "cursor-default",
        )}
      >
        {/* Status / type icon */}
        {tool.status === "running" ? (
          <Loader2 className="size-3.5 text-blue-400 animate-spin shrink-0" />
        ) : tool.status === "error" ? (
          <XCircle className="size-3.5 text-red-500 shrink-0" />
        ) : (
          <Icon className={cn("size-3.5 shrink-0", iconColor)} />
        )}

        {/* Label */}
        <span className="text-xs text-muted-foreground shrink-0">{label}</span>

        {/* Detail: file path, command, or search pattern */}
        <code
          className={cn(
            "text-xs font-mono truncate",
            isFileTool
              ? "text-blue-300/80"
              : tool.type === "bash"
                ? "text-foreground/80"
                : "text-cyan-300/80",
          )}
        >
          {isFileTool ? shortenPath(detail) : detail}
        </code>

        {/* Expand chevron */}
        {isExpandable && (
          <ChevronRight
            className={cn(
              "size-3 shrink-0 text-muted-foreground/60 transition-transform ml-auto",
              expanded && "rotate-90",
            )}
          />
        )}
      </button>

      {/* Expanded detail panel */}
      {expanded && hasExpandableContent && (
        <div
          className={cn(
            "mt-1 ml-5 rounded-md overflow-hidden border",
            tool.status === "error"
              ? "border-red-500/20 bg-red-950/20"
              : "border-border/40 bg-muted/20",
          )}
        >
          {/* Bash: show full command if truncated */}
          {tool.type === "bash" && detail.length > 60 && (
            <div className="px-3 py-1.5 font-mono text-[11px] text-foreground/70 bg-muted/40 border-b border-border/30">
              <span className="text-muted-foreground/60 mr-1">$</span>
              {detail}
            </div>
          )}

          {/* Written/edited content preview */}
          {tool.content && (
            <div className="border-b border-border/30">
              <pre className="px-3 py-2 text-[11px] font-mono text-emerald-400/80 overflow-x-auto max-h-40 overflow-y-auto leading-relaxed">
                {tool.content.split("\n").map((line, i) => (
                  <div key={i} className="flex">
                    <span className="text-muted-foreground/40 select-none w-7 shrink-0 text-right pr-2 tabular-nums">
                      {i + 1}
                    </span>
                    <span>
                      <span className="text-emerald-500/50 select-none">+ </span>
                      {line}
                    </span>
                  </div>
                ))}
              </pre>
            </div>
          )}

          {/* Output / error */}
          {(tool.output || tool.error) && (
            <div className="px-3 py-2 flex items-start gap-1.5">
              {tool.status === "success" && (
                <CheckCircle2 className="size-3 text-emerald-500/60 shrink-0 mt-0.5" />
              )}
              {tool.status === "error" && (
                <XCircle className="size-3 text-red-500 shrink-0 mt-0.5" />
              )}
              <pre
                className={cn(
                  "text-[11px] font-mono overflow-x-auto max-h-48 overflow-y-auto flex-1 whitespace-pre-wrap break-all",
                  tool.status === "error" ? "text-red-400" : "text-muted-foreground",
                )}
              >
                {tool.error ?? tool.output}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
