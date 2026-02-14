import { useState } from "react";
import { cn } from "@/lib/utils";
import { ChevronDown, Sparkles } from "lucide-react";

const shimmerKeyframes = `
@keyframes shimmer {
  0% { background-position: -200% 0; }
  100% { background-position: 200% 0; }
}
`;

if (typeof document !== "undefined") {
  const styleId = "thinking-shimmer-style";
  if (!document.getElementById(styleId)) {
    const style = document.createElement("style");
    style.id = styleId;
    style.textContent = shimmerKeyframes;
    document.head.appendChild(style);
  }
}

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

export function ThinkingBlock({
  thinking,
  defaultExpanded = false,
  className,
}: ThinkingBlockProps) {
  const [expanded, setExpanded] = useState(defaultExpanded);

  const isThinking = thinking.status === "thinking";
  const displayContent = isThinking ? (thinking.streamingContent ?? "") : thinking.content;
  const hasContent = isThinking || !!displayContent;

  const getPreview = () => {
    if (!displayContent) return "";
    const text = displayContent.replace(/\n/g, " ").replace(/\s+/g, " ").trim();
    if (text.length <= 120) return text;
    return text.slice(0, 120) + "...";
  };

  const preview = getPreview();

  return (
    <button
      type="button"
      onClick={() => setExpanded(!expanded)}
      className={cn(
        "group w-full text-left py-0.5",
        "hover:brightness-125 cursor-pointer",
        className,
      )}
    >
      {!expanded ? (
        <div className="flex items-center gap-2">
          <Sparkles
            className={cn(
              "size-3.5 shrink-0",
              isThinking ? "text-violet-400" : "text-muted-foreground/80",
            )}
          />
          {isThinking ? (
            <span className="flex items-center gap-1.5 text-xs">
              <span
                className="bg-gradient-to-r from-violet-400 via-violet-200 to-violet-400 bg-[length:200%_100%] bg-clip-text text-transparent"
                style={{ animation: "shimmer 2s linear infinite" }}
              >
                Thinking
              </span>
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
            <span className="text-xs text-muted-foreground/80 truncate">{preview}</span>
          )}
          <ChevronDown className="size-3 shrink-0 text-muted-foreground/50 -rotate-90 transition-transform" />
        </div>
      ) : (
        <div>
          <div className="flex items-center gap-2 mb-1">
            <Sparkles
              className={cn(
                "size-3.5 shrink-0",
                isThinking ? "text-violet-400" : "text-muted-foreground/80",
              )}
            />
            {isThinking ? (
              <span className="flex items-center gap-1.5 text-xs">
                <span
                  className="bg-gradient-to-r from-violet-400 via-violet-200 to-violet-400 bg-[length:200%_100%] bg-clip-text text-transparent"
                  style={{ animation: "shimmer 2s linear infinite" }}
                >
                  Thinking
                </span>
              </span>
            ) : (
              <span className="text-xs text-muted-foreground/80">Thought</span>
            )}
            <ChevronDown className="size-3 shrink-0 text-muted-foreground/50 transition-transform" />
          </div>
          {hasContent && (
            <div
              className={cn("ml-[22px] rounded-lg border border-border/30 bg-muted/20 px-3 py-2")}
            >
              <p
                className={cn(
                  "text-[11px] leading-relaxed whitespace-pre-wrap font-mono",
                  isThinking ? "text-violet-300/80" : "text-muted-foreground",
                )}
                style={
                  isThinking
                    ? {
                        background:
                          "linear-gradient(90deg, rgb(192 132 252 / 0.8) 0%, rgb(196 181 253) 50%, rgb(192 132 252 / 0.8) 100%)",
                        backgroundSize: "200% 100%",
                        WebkitBackgroundClip: "text",
                        backgroundClip: "text",
                        WebkitTextFillColor: "transparent",
                        animation: "shimmer 2s linear infinite",
                      }
                    : undefined
                }
              >
                {displayContent}
                {isThinking && (
                  <span
                    className="inline-block w-1.5 h-3 bg-violet-400 ml-0.5 animate-pulse align-middle"
                    style={{ WebkitTextFillColor: "initial" }}
                  />
                )}
              </p>
            </div>
          )}
        </div>
      )}
    </button>
  );
}
