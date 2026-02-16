import { createFileRoute } from "@tanstack/react-router";
import { ThreadChangesPanel } from "@/components/threads/ThreadChangesPanel";
import { mockThreadPatchData, mockThreadComments } from "@/data/mockThreadChanges";

export const Route = createFileRoute("/app/threads/$threadId/changes")({
  component: ChangesTab,
});

function ChangesTab() {
  return (
    <div className="h-full">
      <ThreadChangesPanel patchData={mockThreadPatchData} comments={mockThreadComments} />
    </div>
  );
}
