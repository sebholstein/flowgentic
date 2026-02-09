import { useEffect, useRef, useState, useCallback } from "react";
import { MergeView } from "@codemirror/merge";
import {
  EditorView,
  lineNumbers,
  highlightActiveLineGutter,
  highlightSpecialChars,
  drawSelection,
  highlightActiveLine,
  keymap,
} from "@codemirror/view";
import { EditorState } from "@codemirror/state";
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import {
  syntaxHighlighting,
  defaultHighlightStyle,
  bracketMatching,
  foldGutter,
  indentOnInput,
  foldKeymap,
} from "@codemirror/language";
import { javascript } from "@codemirror/lang-javascript";
import { css } from "@codemirror/lang-css";
import { html } from "@codemirror/lang-html";
import { json } from "@codemirror/lang-json";
import { python } from "@codemirror/lang-python";
import { cn } from "@/lib/utils";
import type { FileDiff, DiffViewMode, LineComment } from "@/types/code-review";

interface DiffViewerProps {
  file: FileDiff;
  viewMode: DiffViewMode;
  comments?: LineComment[];
  onLineClick?: (lineNumber: number) => void;
  className?: string;
}

function getLanguageExtension(language: string) {
  switch (language) {
    case "javascript":
      return javascript({ jsx: true });
    case "typescript":
      return javascript({ jsx: true, typescript: true });
    case "css":
      return css();
    case "html":
      return html();
    case "json":
      return json();
    case "python":
      return python();
    default:
      return [];
  }
}

// Custom theme for dark mode diff viewer
const diffTheme = EditorView.theme(
  {
    "&": {
      backgroundColor: "transparent",
      fontSize: "13px",
    },
    ".cm-content": {
      fontFamily:
        "ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Consolas, 'Liberation Mono', monospace",
      padding: "8px 0",
    },
    ".cm-gutters": {
      backgroundColor: "hsl(var(--muted) / 0.3)",
      borderRight: "1px solid hsl(var(--border))",
      color: "hsl(var(--muted-foreground))",
    },
    ".cm-lineNumbers .cm-gutterElement": {
      padding: "0 8px 0 12px",
      minWidth: "40px",
    },
    ".cm-activeLine": {
      backgroundColor: "hsl(var(--muted) / 0.5)",
    },
    ".cm-activeLineGutter": {
      backgroundColor: "hsl(var(--muted) / 0.5)",
    },
    // Merge view specific styles
    ".cm-mergeView": {
      height: "100%",
    },
    ".cm-mergeViewEditors": {
      height: "100%",
    },
    ".cm-mergeViewEditor": {
      height: "100%",
      overflow: "auto",
    },
    // Deletion styling (red background)
    ".cm-deletedChunk": {
      backgroundColor: "hsl(0 62% 30% / 0.3) !important",
    },
    ".cm-deletedLine": {
      backgroundColor: "hsl(0 62% 30% / 0.2)",
    },
    // Addition styling (green background)
    ".cm-insertedChunk": {
      backgroundColor: "hsl(142 71% 29% / 0.3) !important",
    },
    ".cm-insertedLine": {
      backgroundColor: "hsl(142 71% 29% / 0.2)",
    },
    // Changed text highlight
    ".cm-changedText": {
      backgroundColor: "hsl(48 96% 53% / 0.25) !important",
    },
    // Collapsed unchanged regions
    ".cm-collapsedLines": {
      backgroundColor: "hsl(var(--muted) / 0.5)",
      color: "hsl(var(--muted-foreground))",
      padding: "4px 12px",
      cursor: "pointer",
      borderTop: "1px solid hsl(var(--border))",
      borderBottom: "1px solid hsl(var(--border))",
    },
    ".cm-collapsedLines:hover": {
      backgroundColor: "hsl(var(--muted))",
    },
  },
  { dark: true },
);

const baseExtensions = [
  lineNumbers(),
  highlightActiveLineGutter(),
  highlightSpecialChars(),
  history(),
  foldGutter(),
  drawSelection(),
  indentOnInput(),
  syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
  bracketMatching(),
  highlightActiveLine(),
  keymap.of([...defaultKeymap, ...historyKeymap, ...foldKeymap]),
  EditorView.editable.of(false),
  EditorState.readOnly.of(true),
  diffTheme,
];

export function DiffViewer({
  file,
  viewMode,
  comments = [],
  onLineClick: _onLineClick,
  className,
}: DiffViewerProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const mergeViewRef = useRef<MergeView | null>(null);
  const [isReady, setIsReady] = useState(false);

  // Get comments for this file grouped by line (for future use)
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const _commentsByLine = comments.reduce(
    (acc, comment) => {
      if (!comment.parentId) {
        const line = comment.lineNumber;
        if (!acc[line]) acc[line] = [];
        acc[line].push(comment);
      }
      return acc;
    },
    {} as Record<number, LineComment[]>,
  );

  // Cleanup function
  const cleanup = useCallback(() => {
    if (mergeViewRef.current) {
      mergeViewRef.current.destroy();
      mergeViewRef.current = null;
    }
  }, []);

  useEffect(() => {
    if (!containerRef.current) return;

    cleanup();

    const langExtension = getLanguageExtension(file.language);

    const extensions = [
      ...baseExtensions,
      ...(Array.isArray(langExtension) ? langExtension : [langExtension]),
    ];

    // Create MergeView
    const mergeView = new MergeView({
      a: {
        doc: file.oldContent,
        extensions,
      },
      b: {
        doc: file.newContent,
        extensions,
      },
      parent: containerRef.current,
      collapseUnchanged: {
        margin: 3,
        minSize: 4,
      },
      gutter: true,
      highlightChanges: true,
      renderRevertControl: undefined,
    });

    mergeViewRef.current = mergeView;
    setIsReady(true);

    return cleanup;
  }, [file.oldContent, file.newContent, file.language, cleanup]);

  // Handle view mode changes
  useEffect(() => {
    if (!mergeViewRef.current || !containerRef.current) return;

    // The MergeView handles both views internally
    // We can toggle visibility with CSS based on viewMode
    const container = containerRef.current;
    if (viewMode === "unified") {
      container.classList.add("unified-view");
      container.classList.remove("split-view");
    } else {
      container.classList.add("split-view");
      container.classList.remove("unified-view");
    }
  }, [viewMode]);

  return (
    <div className={cn("relative h-full w-full overflow-hidden", className)}>
      <div
        ref={containerRef}
        className={cn(
          "h-full w-full overflow-auto",
          "[&_.cm-mergeView]:h-full",
          "[&_.cm-mergeViewEditors]:h-full [&_.cm-mergeViewEditors]:flex",
          "[&_.cm-mergeViewEditor]:flex-1 [&_.cm-mergeViewEditor]:min-w-0 [&_.cm-mergeViewEditor]:overflow-auto",
          viewMode === "unified" && "[&_.cm-mergeViewEditor:first-child]:hidden",
          viewMode === "split" &&
            "[&_.cm-mergeViewEditor]:border-r [&_.cm-mergeViewEditor]:border-border [&_.cm-mergeViewEditor:last-child]:border-r-0",
        )}
      />
      {!isReady && (
        <div className="absolute inset-0 flex items-center justify-center bg-background/50">
          <span className="text-sm text-muted-foreground">Loading diff...</span>
        </div>
      )}
    </div>
  );
}

// Lightweight unified diff for smaller views
export function UnifiedDiffViewer({ file, className }: { file: FileDiff; className?: string }) {
  return <DiffViewer file={file} viewMode="unified" className={className} />;
}

// Side-by-side diff viewer
export function SplitDiffViewer({ file, className }: { file: FileDiff; className?: string }) {
  return <DiffViewer file={file} viewMode="split" className={className} />;
}
