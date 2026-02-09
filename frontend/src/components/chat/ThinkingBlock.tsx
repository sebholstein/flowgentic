import { useState } from "react";
import { cn } from "@/lib/utils";
import { ChevronDown, ChevronRight } from "lucide-react";

export interface ThinkingPhase {
  id: string;
  status: "thinking" | "complete";
  content?: string;
  streamingContent?: string;
  duration?: number;
}

interface ThinkingBlockProps {
  thinking: ThinkingPhase;
  defaultExpanded?: boolean;
  className?: string;
}

export function ThinkingBlock({ thinking, defaultExpanded = false, className }: ThinkingBlockProps) {
  const [expanded, setExpanded] = useState(defaultExpanded);

  const isThinking = thinking.status === "thinking";
  const displayContent = isThinking ? (thinking.streamingContent ?? "") : thinking.content;
  const hasContent = isThinking || !!displayContent; // Always show content area when thinking (for cursor)

  // Get preview (first ~100 chars)
  const getPreview = () => {
    if (!displayContent) return "";
    const text = displayContent.replace(/\n/g, " ").replace(/\s+/g, " ").trim();
    if (text.length <= 100) return text;
    return text.slice(0, 100);
  };

  const preview = getPreview();
  const hasMoreContent = displayContent && displayContent.length > 100;

  return (
    <div
      className={cn(
        "rounded-md border text-[11px]",
        isThinking ? "border-violet-500/30 bg-violet-500/5 text-violet-500" : "border-border/50 bg-muted/30 text-muted-foreground",
        className,
      )}
    >
      {!expanded ? (
        // Collapsed state - single line
        <button
          onClick={() => setExpanded(true)}
          className="flex items-center gap-1.5 w-full text-left px-2 py-1.5 hover:bg-muted/50 rounded-md cursor-pointer"
        >
          <ChevronRight className="size-3 shrink-0 text-muted-foreground" />
          {isThinking ? (
            <span className="text-violet-400 flex items-center gap-1.5">
              Thinking
              <span className="flex items-center gap-0.5">
                <span className="h-1 w-1 rounded-full bg-violet-500 animate-pulse" />
                <span
                  className="h-1 w-1 rounded-full bg-violet-500 animate-pulse"
                  style={{ animationDelay: "150ms" }}
                />
                <span
                  className="h-1 w-1 rounded-full bg-violet-500 animate-pulse"
                  style={{ animationDelay: "300ms" }}
                />
              </span>
            </span>
          ) : (
            <span className="text-muted-foreground truncate">
              {preview}
              {hasMoreContent && "..."}
            </span>
          )}
        </button>
      ) : (
        // Expanded state
        <div className="px-2 py-1.5">
          <button
            onClick={() => setExpanded(false)}
            className="flex items-center gap-1.5 text-muted-foreground hover:text-foreground mb-1 cursor-pointer"
          >
            <ChevronDown className="size-3" />
            {isThinking ? (
              <span className="text-violet-400 flex items-center gap-1.5">
                Thinking
                <span className="flex items-center gap-0.5">
                  <span className="h-1 w-1 rounded-full bg-violet-500 animate-pulse" />
                  <span
                    className="h-1 w-1 rounded-full bg-violet-500 animate-pulse"
                    style={{ animationDelay: "150ms" }}
                  />
                  <span
                    className="h-1 w-1 rounded-full bg-violet-500 animate-pulse"
                    style={{ animationDelay: "300ms" }}
                  />
                </span>
              </span>
            ) : (
              <span className="text-muted-foreground">Thought</span>
            )}
          </button>
          {hasContent && (
            <p
              className={cn(
                "text-muted-foreground whitespace-pre-wrap leading-relaxed pl-[18px]",
                isThinking && "text-violet-300/80",
              )}
            >
              {displayContent}
              {isThinking && (
                <span className="inline-block w-1.5 h-3 bg-violet-400 ml-0.5 animate-pulse" />
              )}
            </p>
          )}
        </div>
      )}
    </div>
  );
}
