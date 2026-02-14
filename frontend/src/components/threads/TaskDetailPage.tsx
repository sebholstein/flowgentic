import { useParams } from "@tanstack/react-router";
import { useState, useMemo } from "react";
import { threadTasks } from "@/data/mockTasksData";
import { threads } from "@/data/mockThreadsData";
import { mockProgressSteps, mockRunningProgressSteps } from "@/data/mockTaskData";
import { mockExecutionDiff, mockComments } from "@/data/mockDiffData";
import { AgentChatPanel, type ChatTarget } from "@/components/chat/AgentChatPanel";
import { CodeReviewView } from "@/components/code-review/CodeReviewView";
import { TaskHeader } from "./TaskHeader";
import { TaskThreePanelLayout } from "./TaskThreePanelLayout";

export function TaskDetailPage() {
  const { threadId, taskId } = useParams({ from: "/app/tasks/$threadId/$taskId/" });
  const [leftPanelWidth, setLeftPanelWidth] = useState(360);

  const thread = threads.find((i) => i.id === threadId);
  const tasks = useMemo(() => threadTasks[threadId] ?? [], [threadId]);
  const task = tasks.find((t) => t.id === taskId);

  const isRunning = task?.status === "running";
  const progressSteps = isRunning ? mockRunningProgressSteps : mockProgressSteps;

  // Chat target for the always-visible chat panel
  const chatTarget = useMemo<ChatTarget | null>(() => {
    if (!task) return null;
    return {
      type: "task_agent",
      entityId: task.id,
      agentName: task.agent || "Agent",
      title: task.name,
      agentColor: "bg-orange-500",
    };
  }, [task]);

  if (!thread || !task) {
    return (
      <div className="flex h-full items-center justify-center text-muted-foreground">
        Task not found
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      <TaskHeader
        task={task}
        threadId={threadId}
        threadTitle={thread.topic}
        progressSteps={progressSteps}
      />

      <TaskThreePanelLayout
        leftPanel={
          chatTarget && <AgentChatPanel target={chatTarget} />
        }
        rightPanel={
          <CodeReviewView
            execution={mockExecutionDiff}
            comments={mockComments}
            onApprove={() => console.log("Approved")}
            onRequestChanges={() => console.log("Requested changes")}
            onDismiss={() => console.log("Dismissed")}
          />
        }
        leftPanelWidth={leftPanelWidth}
        onLeftPanelResize={setLeftPanelWidth}
      />
    </div>
  );
}
