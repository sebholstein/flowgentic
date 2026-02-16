import { memo } from "react";
import { ProjectRow, SortableProjectRow } from "./ProjectRow";
import { ThreadRow } from "./ThreadRow";
import { TaskRow } from "./TaskRow";
import type { FlatTreeNode } from "./sidebar-types";

export const TreeNodeRow = memo(function TreeNodeRow({
  node,
  selectedThreadId,
  selectedTaskId,
  expandedProjects,
  expandedThreads,
  pinnedThreads,
  archivedThreads,
  onToggleProject,
  onToggleThread,
  onTogglePin,
  onToggleArchive,
  onAddThread,
}: {
  node: FlatTreeNode;
  selectedThreadId: string | null;
  selectedTaskId: string | null;
  expandedProjects: Set<string>;
  expandedThreads: Set<string>;
  pinnedThreads: Set<string>;
  archivedThreads: Set<string>;
  onToggleProject: (id: string) => void;
  onToggleThread: (id: string) => void;
  onTogglePin: (id: string) => void;
  onToggleArchive: (id: string) => void;
  onAddThread: (projectId: string) => void;
}) {
  switch (node.type) {
    case "project":
      if (node.isDemo) {
        return (
          <ProjectRow
            project={node.project}
            threadCount={node.threadCount}
            isExpanded={expandedProjects.has(node.project.id)}
            onToggle={() => onToggleProject(node.project.id)}
            onAddThread={() => {}}
            hideAddButton
          />
        );
      }
      return (
        <SortableProjectRow
          project={node.project}
          threadCount={node.threadCount}
          isExpanded={expandedProjects.has(node.project.id)}
          onToggle={() => onToggleProject(node.project.id)}
          onAddThread={() => onAddThread(node.project.id)}
        />
      );
    case "thread":
      return (
        <ThreadRow
          thread={node.thread}
          isSelected={selectedThreadId === node.thread.id && !selectedTaskId}
          isExpanded={expandedThreads.has(node.thread.id)}
          hasChildren={node.hasChildren}
          isPinned={pinnedThreads.has(node.thread.id)}
          isArchived={archivedThreads.has(node.thread.id)}
          isDemo={node.isDemo}
          onToggle={() => onToggleThread(node.thread.id)}
          onTogglePin={() => onTogglePin(node.thread.id)}
          onToggleArchive={() => onToggleArchive(node.thread.id)}
        />
      );
    case "task":
      return (
        <TaskRow
          task={node.task}
          threadId={node.threadId}
          isSelected={selectedTaskId === node.task.id}
        />
      );
  }
});
