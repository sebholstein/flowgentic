import { useRef } from "react";
import { useVirtualizer } from "@tanstack/react-virtual";
import {
  DndContext,
  DragOverlay,
  closestCenter,
  type DragStartEvent,
  type DragEndEvent,
  type DragOverEvent,
  type SensorDescriptor,
  type SensorOptions,
} from "@dnd-kit/core";
import {
  SortableContext,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Plus } from "lucide-react";
import { ProjectRowOverlay } from "./ProjectRow";
import { TreeNodeRow } from "./TreeNodeRow";
import type { FlatTreeNode } from "./sidebar-types";
import type { Project } from "@/types/project";

export function ThreadTree({
  flattenedNodes,
  projectIds,
  sensors,
  activeProject,
  activeDragThreadCount,
  dropIndicator,
  selectedThreadId,
  selectedTaskId,
  expandedProjects,
  expandedThreads,
  pinnedThreads,
  archivedThreads,
  showAddProject,
  onDragStart,
  onDragOver,
  onDragEnd,
  onDragCancel,
  onToggleProject,
  onToggleThread,
  onTogglePin,
  onToggleArchive,
  onAddThread,
  onAddProject,
}: {
  flattenedNodes: FlatTreeNode[];
  projectIds: string[];
  sensors: SensorDescriptor<SensorOptions>[];
  activeProject: Project | null | undefined;
  activeDragThreadCount: number;
  dropIndicator: { nodeIndex: number; position: "above" | "below" } | null;
  selectedThreadId: string | null;
  selectedTaskId: string | null;
  expandedProjects: Set<string>;
  expandedThreads: Set<string>;
  pinnedThreads: Set<string>;
  archivedThreads: Set<string>;
  showAddProject: boolean;
  onDragStart: (event: DragStartEvent) => void;
  onDragOver: (event: DragOverEvent) => void;
  onDragEnd: (event: DragEndEvent) => void;
  onDragCancel: () => void;
  onToggleProject: (id: string) => void;
  onToggleThread: (id: string) => void;
  onTogglePin: (id: string) => void;
  onToggleArchive: (id: string) => void;
  onAddThread: (projectId: string) => void;
  onAddProject: () => void;
}) {
  const scrollRef = useRef<HTMLDivElement>(null);

  const virtualizer = useVirtualizer({
    count: flattenedNodes.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: (index) => {
      const node = flattenedNodes[index];
      switch (node.type) {
        case "project":
          return 36;
        case "thread":
          return 28;
        case "task":
          return 28;
      }
    },
    overscan: 10,
    getItemKey: (index) => {
      const node = flattenedNodes[index];
      switch (node.type) {
        case "project":
          return `p-${node.project.id}`;
        case "thread":
          return `t-${node.thread.id}`;
        case "task":
          return `tk-${node.task.id}`;
      }
    },
  });

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCenter}
      onDragStart={onDragStart}
      onDragOver={onDragOver}
      onDragEnd={onDragEnd}
      onDragCancel={onDragCancel}
    >
      <SortableContext items={projectIds} strategy={verticalListSortingStrategy}>
        <ScrollArea
          className="h-full overflow-hidden px-2 pt-2"
          viewportRef={scrollRef}
          viewportClassName="!overflow-y-auto"
        >
          {showAddProject && (
            <button
              type="button"
              onClick={onAddProject}
              className="flex w-full items-center gap-1.5 rounded-md px-2 py-1.5 text-sm hover:bg-muted/50 transition-colors text-left select-none text-muted-foreground mb-1 mx-2"
              style={{ paddingLeft: "8px" }}
            >
              <span className="size-4 flex items-center justify-center shrink-0">
                <Plus className="size-3.5" />
              </span>
              <span className="truncate flex-1">Add Project</span>
            </button>
          )}
          <div
            className="p-2"
            style={{
              height: `${virtualizer.getTotalSize()}px`,
              width: "100%",
              position: "relative",
            }}
          >
            {virtualizer.getVirtualItems().map((virtualRow) => {
              const node = flattenedNodes[virtualRow.index];
              return (
                <div
                  key={virtualRow.key}
                  data-index={virtualRow.index}
                  ref={virtualizer.measureElement}
                  style={{
                    position: "absolute",
                    top: 0,
                    left: 0,
                    width: "100%",
                    transform: `translateY(${virtualRow.start}px)`,
                  }}
                >
                  <TreeNodeRow
                    node={node}
                    selectedThreadId={selectedThreadId}
                    selectedTaskId={selectedTaskId}
                    expandedProjects={expandedProjects}
                    expandedThreads={expandedThreads}
                    pinnedThreads={pinnedThreads}
                    archivedThreads={archivedThreads}
                    onToggleProject={onToggleProject}
                    onToggleThread={onToggleThread}
                    onTogglePin={onTogglePin}
                    onToggleArchive={onToggleArchive}
                    onAddThread={onAddThread}
                  />
                </div>
              );
            })}
            {dropIndicator &&
              (() => {
                const items = virtualizer.getVirtualItems();
                const target = items.find((item) => item.index === dropIndicator.nodeIndex);
                if (!target) return null;
                const y =
                  dropIndicator.position === "above"
                    ? target.start
                    : target.start + target.size;
                return (
                  <div
                    className="absolute left-2 right-2 z-10 pointer-events-none"
                    style={{ top: `${y - 1}px` }}
                  >
                    <div className="h-0.5 bg-primary rounded-full" />
                  </div>
                );
              })()}
          </div>
        </ScrollArea>
      </SortableContext>
      <DragOverlay dropAnimation={null}>
        {activeProject ? (
          <ProjectRowOverlay project={activeProject} threadCount={activeDragThreadCount} />
        ) : null}
      </DragOverlay>
    </DndContext>
  );
}
