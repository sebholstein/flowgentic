import { useMemo } from "react";
import { cn } from "@/lib/utils";
import { ScrollArea } from "@/components/ui/scroll-area";
import { FileTreeNode } from "./FileTreeNode";
import type { FileDiff, LineComment } from "@/types/code-review";

interface FileTreeProps {
  files: FileDiff[];
  selectedPath: string | null;
  onSelectFile: (path: string) => void;
  comments?: LineComment[];
  className?: string;
}

interface TreeNode {
  name: string;
  path: string;
  isFolder: boolean;
  children?: TreeNode[];
  file?: FileDiff;
}

function buildFileTree(files: FileDiff[]): TreeNode[] {
  const root: Record<string, TreeNode> = {};

  for (const file of files) {
    const parts = file.path.split("/");
    let current = root;

    for (let i = 0; i < parts.length; i++) {
      const part = parts[i];
      const isLast = i === parts.length - 1;
      const pathSoFar = parts.slice(0, i + 1).join("/");

      if (!current[part]) {
        current[part] = {
          name: part,
          path: pathSoFar,
          isFolder: !isLast,
          children: isLast ? undefined : {},
          file: isLast ? file : undefined,
        };
      }

      if (!isLast) {
        current = current[part].children as Record<string, TreeNode>;
      }
    }
  }

  function convertToArray(nodes: Record<string, TreeNode>): TreeNode[] {
    return Object.values(nodes)
      .map((node) => ({
        ...node,
        children: node.children
          ? convertToArray(node.children as unknown as Record<string, TreeNode>)
          : undefined,
      }))
      .sort((a, b) => {
        // Folders first, then alphabetical
        if (a.isFolder && !b.isFolder) return -1;
        if (!a.isFolder && b.isFolder) return 1;
        return a.name.localeCompare(b.name);
      });
  }

  return convertToArray(root);
}

function getCommentCountForPath(comments: LineComment[], path: string): number {
  return comments.filter((c) => c.filePath === path && !c.parentId).length;
}

export function FileTree({
  files,
  selectedPath,
  onSelectFile,
  comments = [],
  className,
}: FileTreeProps) {
  const tree = useMemo(() => buildFileTree(files), [files]);

  const totalAdditions = files.reduce((sum, f) => sum + f.additions, 0);
  const totalDeletions = files.reduce((sum, f) => sum + f.deletions, 0);

  const renderNode = (node: TreeNode, depth: number = 0): React.ReactNode => {
    const commentCount = node.file ? getCommentCountForPath(comments, node.path) : 0;

    return (
      <FileTreeNode
        key={node.path}
        name={node.name}
        path={node.path}
        isFolder={node.isFolder}
        file={node.file}
        isSelected={selectedPath === node.path}
        onSelect={onSelectFile}
        commentCount={commentCount}
        depth={depth}
      >
        {node.children?.map((child) => renderNode(child, depth + 1))}
      </FileTreeNode>
    );
  };

  return (
    <div className={cn("flex flex-col h-full", className)}>
      <div className="px-3 py-2 border-b">
        <div className="text-xs font-medium text-muted-foreground">Changed Files</div>
        <div className="flex items-center gap-2 mt-1">
          <span className="text-xs text-muted-foreground">
            {files.length} {files.length === 1 ? "file" : "files"}
          </span>
          <span className="text-xs text-emerald-400 tabular-nums">+{totalAdditions}</span>
          <span className="text-xs text-red-400 tabular-nums">-{totalDeletions}</span>
        </div>
      </div>
      <ScrollArea className="flex-1">
        <div className="p-2 space-y-0.5">{tree.map(renderNode)}</div>
      </ScrollArea>
    </div>
  );
}

// Simple flat file list (alternative view)
export function FileList({
  files,
  selectedPath,
  onSelectFile,
  comments = [],
  className,
}: FileTreeProps) {
  return (
    <div className={cn("flex flex-col h-full", className)}>
      <div className="px-3 py-2 border-b">
        <div className="text-xs font-medium text-muted-foreground">
          Changed Files ({files.length})
        </div>
      </div>
      <ScrollArea className="flex-1">
        <div className="p-2 space-y-0.5">
          {files.map((file) => {
            const commentCount = getCommentCountForPath(comments, file.path);
            return (
              <FileTreeNode
                key={file.path}
                name={file.path}
                path={file.path}
                isFolder={false}
                file={file}
                isSelected={selectedPath === file.path}
                onSelect={onSelectFile}
                commentCount={commentCount}
                depth={0}
              />
            );
          })}
        </div>
      </ScrollArea>
    </div>
  );
}
