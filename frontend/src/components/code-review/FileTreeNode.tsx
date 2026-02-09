import { useState, type ReactNode } from "react";
import { cn } from "@/lib/utils";
import {
  ChevronRight,
  ChevronDown,
  FileCode2,
  FileJson,
  FileType,
  FileText,
  Folder,
  FolderOpen,
  Palette,
} from "lucide-react";
import type { FileDiff, FileChangeStatus } from "@/types/code-review";

interface FileTreeNodeProps {
  name: string;
  path: string;
  isFolder: boolean;
  children?: ReactNode;
  file?: FileDiff;
  isSelected?: boolean;
  onSelect?: (path: string) => void;
  depth?: number;
  defaultExpanded?: boolean;
  commentCount?: number;
}

function getFileIcon(name: string | undefined) {
  if (!name) return FileType;
  const ext = name.split(".").pop()?.toLowerCase();
  switch (ext) {
    case "ts":
    case "tsx":
    case "js":
    case "jsx":
      return FileCode2;
    case "json":
      return FileJson;
    case "css":
    case "scss":
    case "less":
      return Palette;
    case "md":
    case "txt":
      return FileText;
    default:
      return FileType;
  }
}

function getStatusIndicator(status: FileChangeStatus) {
  switch (status) {
    case "added":
      return { label: "A", color: "text-emerald-400", bgColor: "bg-emerald-500/20" };
    case "modified":
      return { label: "M", color: "text-amber-400", bgColor: "bg-amber-500/20" };
    case "deleted":
      return { label: "D", color: "text-red-400", bgColor: "bg-red-500/20" };
    case "renamed":
      return { label: "R", color: "text-blue-400", bgColor: "bg-blue-500/20" };
    default:
      return null;
  }
}

export function FileTreeNode({
  name,
  path,
  isFolder,
  children,
  file,
  isSelected = false,
  onSelect,
  depth = 0,
  defaultExpanded = true,
  commentCount = 0,
}: FileTreeNodeProps) {
  const [isExpanded, setIsExpanded] = useState(defaultExpanded);

  const handleClick = () => {
    if (isFolder) {
      setIsExpanded(!isExpanded);
    } else if (onSelect) {
      onSelect(path);
    }
  };

  const FileIcon = isFolder ? (isExpanded ? FolderOpen : Folder) : getFileIcon(name);

  const statusIndicator = file ? getStatusIndicator(file.status) : null;

  return (
    <div>
      <button
        type="button"
        onClick={handleClick}
        className={cn(
          "flex w-full items-center gap-1.5 rounded-md px-2 py-1 text-sm hover:bg-muted/50 transition-colors text-left",
          isSelected && "bg-muted text-foreground",
          !isSelected && "text-muted-foreground",
        )}
        style={{ paddingLeft: `${depth * 12 + 8}px` }}
      >
        {isFolder ? (
          <span className="size-4 flex items-center justify-center shrink-0">
            {isExpanded ? (
              <ChevronDown className="size-3.5" />
            ) : (
              <ChevronRight className="size-3.5" />
            )}
          </span>
        ) : (
          <span className="size-4 shrink-0" />
        )}

        <FileIcon
          className={cn("size-4 shrink-0", isFolder ? "text-amber-400" : "text-muted-foreground")}
        />

        <span className="truncate flex-1">{name}</span>

        {statusIndicator && (
          <span
            className={cn(
              "text-xs font-medium px-1.5 py-0.5 rounded shrink-0",
              statusIndicator.color,
              statusIndicator.bgColor,
            )}
          >
            {statusIndicator.label}
          </span>
        )}

        {commentCount > 0 && (
          <span className="text-xs bg-blue-500/20 text-blue-400 px-1.5 py-0.5 rounded shrink-0">
            {commentCount}
          </span>
        )}

        {file && (
          <span className="text-xs text-muted-foreground shrink-0 tabular-nums">
            <span className="text-emerald-400">+{file.additions}</span>{" "}
            <span className="text-red-400">-{file.deletions}</span>
          </span>
        )}
      </button>

      {isFolder && isExpanded && children && <div>{children}</div>}
    </div>
  );
}
