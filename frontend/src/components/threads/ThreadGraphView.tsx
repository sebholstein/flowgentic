import { useCallback, useMemo, useEffect } from "react";
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
  Panel,
  type Node,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Play, Pause, RotateCcw } from "lucide-react";
import type { Task } from "@/types/task";
import { TaskNode, type TaskNodeData } from "./TaskNode";
import { ExecutionNote, type ExecutionNoteData } from "./ExecutionNote";
import { taskStatusConfig } from "@/constants/taskStatusConfig";
import { generateDAGLayout } from "@/lib/dag-layout";

const nodeTypes = {
  taskNode: TaskNode,
  executionNote: ExecutionNote,
};

interface ThreadGraphViewProps {
  tasks: Task[];
  selectedTaskId?: string;
  isSimulating: boolean;
  onStart: () => void;
  onPause: () => void;
  onReset: () => void;
  onNodeClick: (taskId: string) => void;
}

/**
 * ReactFlow-based graph visualization of tasks and their dependencies.
 * Shows tasks as nodes with edges representing dependencies, plus execution notes.
 */
export function ThreadGraphView({
  tasks,
  selectedTaskId,
  isSimulating,
  onStart,
  onPause,
  onReset,
  onNodeClick,
}: ThreadGraphViewProps) {
  const { nodes: layoutNodes, edges: layoutEdges } = useMemo(
    () => generateDAGLayout(tasks, selectedTaskId),
    [tasks, selectedTaskId],
  );

  const [nodes, setNodes, onNodesChange] = useNodesState(layoutNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(layoutEdges);

  useEffect(() => {
    const { nodes: newNodes, edges: newEdges } = generateDAGLayout(tasks, selectedTaskId);
    setNodes(newNodes);
    setEdges(newEdges);
  }, [tasks, selectedTaskId, setNodes, setEdges]);

  const handleNodeClick = useCallback(
    (_: React.MouseEvent, node: Node) => {
      onNodeClick(node.id);
    },
    [onNodeClick],
  );

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      onNodeClick={handleNodeClick}
      nodeTypes={nodeTypes}
      fitView
      fitViewOptions={{ padding: 0.3 }}
      minZoom={0.3}
      maxZoom={1.5}
      defaultEdgeOptions={{
        type: "smoothstep",
      }}
    >
      <Background className="!bg-muted/50" color="var(--color-border)" gap={20} size={1} />
      <Controls className="!bg-card !border-border !rounded-lg !shadow-sm [&>button]:!bg-card [&>button]:!border-border [&>button]:!text-foreground [&>button:hover]:!bg-muted" />
      <MiniMap
        className="!bg-card !border-border !rounded-lg !shadow-sm"
        nodeColor={(node) => {
          // Execution notes are smaller and use agent colors
          if (node.type === "executionNote") {
            const execData = node.data as ExecutionNoteData;
            switch (execData.execution.status) {
              case "completed":
                return "#34d399";
              case "running":
                return "#60a5fa";
              case "failed":
                return "#f87171";
              default:
                return "#64748b";
            }
          }
          // Task nodes
          const status = (node.data as TaskNodeData).task.status;
          switch (status) {
            case "completed":
              return "#34d399";
            case "running":
              return "#60a5fa";
            case "failed":
              return "#f87171";
            case "blocked":
              return "#fbbf24";
            case "needs_feedback":
              return "#a78bfa";
            default:
              return "#64748b";
          }
        }}
        maskColor="var(--color-background)"
      />
      <Panel position="top-left" className="!m-4">
        <div className="flex flex-wrap gap-3 rounded-lg border border-border bg-card/95 px-3 py-2 text-xs backdrop-blur shadow-sm">
          {Object.entries(taskStatusConfig).map(([status, config]) => {
            const Icon = config.icon;
            return (
              <div key={status} className="flex items-center gap-1.5">
                <Icon className={cn("size-3", config.color)} />
                <span className="text-foreground">{config.label}</span>
              </div>
            );
          })}
        </div>
      </Panel>
      <Panel position="top-right" className="!m-4">
        <div className="flex gap-2 rounded-lg border border-border bg-card/95 p-2 backdrop-blur shadow-sm">
          {isSimulating ? (
            <Button size="sm" variant="outline" onClick={onPause} className="gap-1.5">
              <Pause className="size-3.5" />
              Pause
            </Button>
          ) : (
            <Button
              size="sm"
              onClick={onStart}
              className="gap-1.5 bg-blue-600 hover:bg-blue-700 text-white"
            >
              <Play className="size-3.5" />
              Simulate
            </Button>
          )}
          <Button size="sm" variant="outline" onClick={onReset} className="gap-1.5">
            <RotateCcw className="size-3.5" />
            Reset
          </Button>
        </div>
      </Panel>
    </ReactFlow>
  );
}
