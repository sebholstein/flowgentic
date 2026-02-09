import { createFileRoute } from "@tanstack/react-router";
import { CodeReviewView } from "@/components/code-review/CodeReviewView";
import { mockExecutionDiff, mockComments } from "@/data/mockDiffData";

export const Route = createFileRoute("/app/tasks/$threadId/$taskId/changes")({
  component: ChangesTab,
});

function ChangesTab() {
  return (
    <CodeReviewView
      execution={mockExecutionDiff}
      comments={mockComments}
      onApprove={() => console.log("Approved")}
      onRequestChanges={() => console.log("Requested changes")}
      onDismiss={() => console.log("Dismissed")}
    />
  );
}
