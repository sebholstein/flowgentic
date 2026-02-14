import { useMemo, useState } from "react";
import { ChevronRight, Loader2, Server, XCircle } from "lucide-react";
import { cn } from "@/lib/utils";
import type { ToolCall } from "./ToolCallBlock";
import { FlowgenticPlanToolResult } from "./FlowgenticPlanToolResult";

interface McpToolCallBlockProps {
  tool: ToolCall;
  className?: string;
}

interface ParsedMcpToolName {
  server: string | null;
  tool: string;
}

function parseMcpToolName(rawTitle: string): ParsedMcpToolName {
  const title = rawTitle.trim();
  if (!title) return { server: null, tool: "unknown_tool" };

  const namespaced = title.match(/^mcp__([^_]+)__(.+)$/);
  if (namespaced) {
    return { server: namespaced[1], tool: namespaced[2] };
  }

  const dot = title.match(/^([a-zA-Z0-9_-]+)\.([a-zA-Z0-9_:-]+)$/);
  if (dot) {
    return { server: dot[1], tool: dot[2] };
  }

  return { server: null, tool: title };
}

function tryParseJSON(raw: string | undefined): unknown | null {
  if (!raw) return null;
  try {
    return JSON.parse(raw) as unknown;
  } catch {
    return null;
  }
}

function getObject(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  return value as Record<string, unknown>;
}

function parseMcpOutput(rawOutput: string | undefined): { structured: Record<string, unknown> | null; text: string | null } {
  if (!rawOutput) return { structured: null, text: null };

  const parsed = tryParseJSON(rawOutput);
  const parsedObj = getObject(parsed);
  if (!parsedObj) {
    return { structured: null, text: rawOutput };
  }

  const structured = getObject(parsedObj.structuredContent) ?? parsedObj;
  const content = Array.isArray(parsedObj.content) ? parsedObj.content : null;
  if (content && content.length > 0) {
    const first = getObject(content[0]);
    if (first && first.type === "text" && typeof first.text === "string") {
      return { structured, text: first.text };
    }
  }
  return { structured, text: null };
}

export function McpToolCallBlock({ tool, className }: McpToolCallBlockProps) {
  const [expanded, setExpanded] = useState(false);

  const parsedName = useMemo(() => parseMcpToolName(tool.title), [tool.title]);
  const parsedInput = useMemo(() => tool.input ?? null, [tool.input]);
  const parsedOutput = useMemo(() => parseMcpOutput(tool.output), [tool.output]);

  const hasDetails = !!parsedInput || !!tool.output || !!tool.error;
  const isFlowgentic = parsedName.server === "flowgentic";

  return (
    <div className={cn("group", className)}>
      <button
        onClick={() => hasDetails && setExpanded((v) => !v)}
        disabled={!hasDetails}
        className={cn(
          "flex w-full items-center gap-1.5 py-0.5 text-left",
          hasDetails ? "cursor-pointer hover:brightness-125" : "cursor-default",
        )}
      >
        {tool.status === "running" ? (
          <Loader2 className="size-3.5 shrink-0 animate-spin text-blue-400" />
        ) : tool.status === "error" ? (
          <XCircle className="size-3.5 shrink-0 text-red-500" />
        ) : (
          <Server className="size-3.5 shrink-0 text-violet-400" />
        )}

        <span className="shrink-0 text-xs text-muted-foreground">MCP</span>
        <code className="truncate font-mono text-xs text-foreground/85">
          {parsedName.server ? `${parsedName.server}.${parsedName.tool}` : parsedName.tool}
        </code>

        {hasDetails && (
          <ChevronRight
            className={cn(
              "ml-auto size-3 shrink-0 text-muted-foreground/60 transition-transform",
              expanded && "rotate-90",
            )}
          />
        )}
      </button>

      {isFlowgentic && (
        <div className="ml-5 mt-0.5">
          <FlowgenticPlanToolResult
            toolName={parsedName.tool}
            status={tool.status}
            result={parsedOutput.structured}
          />
        </div>
      )}

      {expanded && hasDetails && (
        <div
          className={cn(
            "ml-5 mt-1 overflow-hidden rounded-md border",
            tool.status === "error" ? "border-red-500/20 bg-red-950/20" : "border-border/40 bg-muted/20",
          )}
        >
          {parsedInput && (
            <div className="border-b border-border/30 px-3 py-2">
              <div className="mb-1 text-[10px] uppercase tracking-wide text-muted-foreground">Arguments</div>
              <pre className="max-h-44 overflow-auto whitespace-pre-wrap break-all font-mono text-[11px] text-muted-foreground">
                {JSON.stringify(parsedInput, null, 2)}
              </pre>
            </div>
          )}

          {(parsedOutput.text || parsedOutput.structured || tool.error) && (
            <div className="px-3 py-2">
              <div className="mb-1 text-[10px] uppercase tracking-wide text-muted-foreground">
                {tool.error ? "Error" : "Result"}
              </div>
              <pre className={cn(
                "max-h-48 overflow-auto whitespace-pre-wrap break-all font-mono text-[11px]",
                tool.error ? "text-red-400" : "text-muted-foreground",
              )}>
                {tool.error ?? parsedOutput.text ?? JSON.stringify(parsedOutput.structured, null, 2)}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
