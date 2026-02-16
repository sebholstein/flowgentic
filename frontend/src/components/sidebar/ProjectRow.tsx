import { memo } from "react";
import { useSortable } from "@dnd-kit/sortable";
import type { DraggableAttributes } from "@dnd-kit/core";
import { cn } from "@/lib/utils";
import {
  ChevronRight,
  ChevronDown,
  Folder,
  FolderOpen,
  Plus,
} from "lucide-react";
import type { Project } from "@/types/project";

export const ProjectRow = memo(function ProjectRow({
  project,
  threadCount,
  isExpanded,
  isDragging,
  onToggle,
  onAddThread,
  hideAddButton,
  sortableRef,
  dragListeners,
  dragAttributes,
}: {
  project: Project;
  threadCount: number;
  isExpanded: boolean;
  isDragging?: boolean;
  onToggle: () => void;
  onAddThread: () => void;
  hideAddButton?: boolean;
  sortableRef?: (node: HTMLElement | null) => void;
  dragListeners?: Record<string, Function>;
  dragAttributes?: DraggableAttributes;
}) {
  return (
    <div
      ref={sortableRef}
      className={cn(
        "group/project flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-sm hover:bg-muted/50 transition-colors text-left select-none",
        "text-foreground",
        isDragging && "opacity-30",
      )}
      style={{ paddingLeft: "8px" }}
    >
      <button
        type="button"
        onClick={onToggle}
        className="size-4 flex items-center justify-center shrink-0"
      >
        {isExpanded ? <ChevronDown className="size-3.5" /> : <ChevronRight className="size-3.5" />}
      </button>
      <span
        className="flex flex-1 items-center gap-1.5 min-w-0 cursor-grab active:cursor-grabbing"
        {...dragListeners}
        {...dragAttributes}
      >
        {isExpanded ? (
          <FolderOpen className={cn("size-4 shrink-0", project.color || "text-amber-400")} />
        ) : (
          <Folder className={cn("size-4 shrink-0", project.color || "text-amber-400")} />
        )}
        <span className="truncate flex-1 text-left">{project.name}</span>
      </span>
      <span className={cn("text-xs text-muted-foreground tabular-nums", !hideAddButton && "group-hover/project:hidden")}>
        {threadCount}
      </span>
      {!hideAddButton && (
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            onAddThread();
          }}
          className="hidden group-hover/project:flex items-center justify-center rounded p-0.5 text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
          aria-label={`Add thread to ${project.name}`}
          title="Add thread"
        >
          <Plus className="size-3.5" />
        </button>
      )}
    </div>
  );
});

export const ProjectRowOverlay = memo(function ProjectRowOverlay({
  project,
  threadCount,
}: {
  project: Project;
  threadCount: number;
}) {
  return (
    <div
      className={cn(
        "flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-sm",
        "text-foreground",
        "bg-sidebar shadow-lg border border-border cursor-grabbing",
      )}
      style={{ paddingLeft: "8px" }}
    >
      <span className="size-4 shrink-0" />
      <Folder className={cn("size-4 shrink-0", project.color || "text-amber-400")} />
      <span className="truncate flex-1 text-left">{project.name}</span>
      <span className="text-xs text-muted-foreground tabular-nums">{threadCount}</span>
    </div>
  );
});

export const SortableProjectRow = memo(function SortableProjectRow({
  project,
  threadCount,
  isExpanded,
  onToggle,
  onAddThread,
}: {
  project: Project;
  threadCount: number;
  isExpanded: boolean;
  onToggle: () => void;
  onAddThread: () => void;
}) {
  const { setNodeRef, attributes, listeners, isDragging } = useSortable({
    id: project.id,
  });
  return (
    <ProjectRow
      project={project}
      threadCount={threadCount}
      isExpanded={isExpanded}
      isDragging={isDragging}
      onToggle={onToggle}
      onAddThread={onAddThread}
      sortableRef={setNodeRef}
      dragListeners={listeners}
      dragAttributes={attributes}
    />
  );
});
