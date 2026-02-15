import { useState, useRef, useEffect, useCallback, type ReactNode } from "react";
import { PatchDiff, type DiffLineAnnotation } from "@pierre/diffs/react";
import { cn } from "@/lib/utils";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Button } from "@/components/ui/button";
import { MessageSquare, Send, FileText, Folder } from "lucide-react";
import { demoPatchData, demoComments, type DemoComment } from "@/data/mockAgentFlowData";

// --- Derive changed file tree from patch data ---

interface ChangedFileNode {
  name: string;
  path: string;
  type: "file" | "directory";
  children?: ChangedFileNode[];
}

function buildChangedTree(files: string[]): ChangedFileNode[] {
  const root: ChangedFileNode[] = [];

  for (const filePath of files) {
    const parts = filePath.split("/");
    let currentLevel = root;

    for (let i = 0; i < parts.length; i++) {
      const part = parts[i];
      const isFile = i === parts.length - 1;
      const fullPath = parts.slice(0, i + 1).join("/");

      const existing = currentLevel.find((n) => n.name === part);
      if (existing) {
        if (existing.type === "directory" && existing.children) {
          currentLevel = existing.children;
        }
      } else {
        const node: ChangedFileNode = {
          name: part,
          path: fullPath,
          type: isFile ? "file" : "directory",
          ...(isFile ? {} : { children: [] }),
        };
        currentLevel.push(node);
        if (!isFile && node.children) {
          currentLevel = node.children;
        }
      }
    }
  }

  return root;
}

// --- Comment UI ---

interface CommentAnnotation {
  comment: DemoComment;
}

function CommentBubble({ comment }: { comment: DemoComment }) {
  return (
    <div className="flex gap-2 py-2 px-3 text-xs">
      <div className="h-5 w-5 rounded-full bg-violet-500 flex items-center justify-center text-white text-[10px] font-medium shrink-0">
        {comment.author[0]}
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-1.5 mb-0.5">
          <span className="font-medium text-foreground">{comment.author}</span>
          <span className="text-muted-foreground text-[10px]">
            {new Date(comment.timestamp).toLocaleTimeString([], {
              hour: "2-digit",
              minute: "2-digit",
            })}
          </span>
        </div>
        <p className="text-foreground/80">{comment.content}</p>
      </div>
    </div>
  );
}

function InlineCommentForm({ onSubmit }: { onSubmit: (text: string) => void }) {
  const [text, setText] = useState("");

  return (
    <div className="flex gap-2 py-2 px-3">
      <textarea
        value={text}
        onChange={(e) => setText(e.target.value)}
        placeholder="Add a comment..."
        className="flex-1 text-xs bg-muted/50 border rounded-md px-2 py-1.5 resize-none focus:outline-none focus:ring-1 focus:ring-ring"
        rows={2}
      />
      <Button
        size="sm"
        className="h-7 w-7 p-0 self-end"
        disabled={!text.trim()}
        onClick={() => {
          onSubmit(text.trim());
          setText("");
        }}
      >
        <Send className="size-3" />
      </Button>
    </div>
  );
}

// --- File tree sidebar node ---

function TreeNode({
  node,
  depth,
  activeFile,
  onClickFile,
}: {
  node: ChangedFileNode;
  depth: number;
  activeFile: string;
  onClickFile: (path: string) => void;
}) {
  if (node.type === "directory") {
    return (
      <>
        <div
          className="flex items-center gap-1 py-1 px-1.5 text-[11px] text-muted-foreground"
          style={{ paddingLeft: `${4 + depth * 12}px` }}
        >
          <Folder className="size-3 text-amber-400/70 shrink-0" />
          <span className="truncate">{node.name}</span>
        </div>
        {node.children?.map((child) => (
          <TreeNode
            key={child.path}
            node={child}
            depth={depth + 1}
            activeFile={activeFile}
            onClickFile={onClickFile}
          />
        ))}
      </>
    );
  }

  const isActive = node.path === activeFile;

  return (
    <button
      type="button"
      onClick={() => onClickFile(node.path)}
      className={cn(
        "flex items-center gap-1 w-full py-1 px-1.5 text-[11px] transition-colors text-left cursor-pointer",
        isActive
          ? "bg-muted text-foreground font-medium"
          : "text-muted-foreground hover:bg-muted/50 hover:text-foreground",
      )}
      style={{ paddingLeft: `${4 + depth * 12}px` }}
    >
      <FileText className="size-3 shrink-0" />
      <span className="truncate flex-1">{node.name}</span>
      <span className="h-1.5 w-1.5 rounded-full bg-amber-400 shrink-0" />
    </button>
  );
}

// --- Main panel ---

export function FileChangesPanel() {
  const changedFiles = Object.keys(demoPatchData);
  const tree = buildChangedTree(changedFiles);

  const [activeFile, setActiveFile] = useState(changedFiles[0]);
  const [comments, setComments] = useState<DemoComment[]>(demoComments);
  const [commentingLine, setCommentingLine] = useState<{
    line: number;
    side: "additions" | "deletions";
    file: string;
  } | null>(null);

  // Refs for each file diff section (for scroll-to and intersection)
  const sectionRefs = useRef<Record<string, HTMLDivElement | null>>({});
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const isScrollingTo = useRef(false);

  // IntersectionObserver: update activeFile based on scroll position
  useEffect(() => {
    const container = scrollContainerRef.current;
    if (!container) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (isScrollingTo.current) return;

        // Find the topmost visible file section
        let topFile: string | null = null;
        let topY = Infinity;

        for (const entry of entries) {
          if (entry.isIntersecting) {
            const file = entry.target.getAttribute("data-file");
            const rect = entry.boundingClientRect;
            if (file && rect.top < topY) {
              topY = rect.top;
              topFile = file;
            }
          }
        }

        if (topFile) {
          setActiveFile(topFile);
        }
      },
      {
        root: container,
        rootMargin: "0px 0px -70% 0px",
        threshold: 0,
      },
    );

    for (const file of changedFiles) {
      const el = sectionRefs.current[file];
      if (el) observer.observe(el);
    }

    return () => observer.disconnect();
  }, [changedFiles]);

  // Click file in tree -> scroll to its diff
  const handleClickFile = useCallback((file: string) => {
    setActiveFile(file);
    const el = sectionRefs.current[file];
    if (el) {
      isScrollingTo.current = true;
      el.scrollIntoView({ behavior: "smooth", block: "start" });
      // Reset after scroll settles
      setTimeout(() => {
        isScrollingTo.current = false;
      }, 600);
    }
  }, []);

  const handleAddComment = (
    file: string,
    lineNumber: number,
    side: "additions" | "deletions",
    text: string,
  ) => {
    setComments((prev) => [
      ...prev,
      {
        id: `comment-${Date.now()}`,
        lineNumber,
        side,
        author: "You",
        content: text,
        timestamp: new Date().toISOString(),
        file,
      },
    ]);
    setCommentingLine(null);
  };

  return (
    <div className="flex h-full">
      {/* File tree sidebar */}
      <div className="w-48 shrink-0 border-r flex flex-col">
        <div className="flex items-center justify-between px-3 py-2 border-b shrink-0">
          <span className="text-[11px] font-medium">Changed files</span>
          <span className="text-[10px] text-muted-foreground">{changedFiles.length}</span>
        </div>
        <ScrollArea className="flex-1">
          <div className="py-1">
            {tree.map((node) => (
              <TreeNode
                key={node.path}
                node={node}
                depth={0}
                activeFile={activeFile}
                onClickFile={handleClickFile}
              />
            ))}
          </div>
        </ScrollArea>
      </div>

      {/* All diffs stacked */}
      <ScrollArea className="flex-1 min-w-0" viewportRef={scrollContainerRef}>
        <div className="divide-y">
          {changedFiles.map((file) => {
            const patch = demoPatchData[file];
            const fileComments = comments.filter((c) => c.file === file);

            const annotations: DiffLineAnnotation<CommentAnnotation>[] =
              fileComments.map((comment) => ({
                lineNumber: comment.lineNumber,
                side: comment.side,
                metadata: { comment },
              }));

            if (commentingLine && commentingLine.file === file) {
              annotations.push({
                lineNumber: commentingLine.line,
                side: commentingLine.side,
                metadata: {
                  comment: {
                    id: "new",
                    lineNumber: commentingLine.line,
                    side: commentingLine.side,
                    author: "You",
                    content: "",
                    timestamp: new Date().toISOString(),
                    file,
                  },
                },
              });
            }

            const renderAnnotation = (
              annotation: DiffLineAnnotation<CommentAnnotation>,
            ): ReactNode => {
              const { comment } = annotation.metadata;
              if (comment.id === "new") {
                return (
                  <InlineCommentForm
                    onSubmit={(text) =>
                      handleAddComment(
                        file,
                        annotation.lineNumber,
                        annotation.side,
                        text,
                      )
                    }
                  />
                );
              }
              return <CommentBubble comment={comment} />;
            };

            return (
              <div
                key={file}
                ref={(el) => {
                  sectionRefs.current[file] = el;
                }}
                data-file={file}
              >
                <PatchDiff
                  patch={patch}
                  options={{
                    theme: "pierre-dark",
                    diffStyle: "unified",
                    lineDiffType: "word",
                  }}
                  lineAnnotations={annotations}
                  renderAnnotation={renderAnnotation}
                  renderHoverUtility={(getHoveredLine) => {
                    const info = getHoveredLine?.();
                    if (!info) return null;
                    return (
                      <button
                        type="button"
                        className="flex items-center gap-1 px-1.5 py-0.5 text-[10px] text-muted-foreground hover:text-foreground bg-muted rounded transition-colors cursor-pointer"
                        onClick={() =>
                          setCommentingLine({
                            line: info.lineNumber,
                            side: info.side,
                            file,
                          })
                        }
                      >
                        <MessageSquare className="size-3" />
                      </button>
                    );
                  }}
                />
              </div>
            );
          })}
        </div>
      </ScrollArea>
    </div>
  );
}
