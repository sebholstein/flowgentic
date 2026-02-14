import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/app/threads/$threadId/memory")({
  component: MemoryTab,
});

function MemoryTab() {
  return (
    <div className="h-full flex items-center justify-center text-muted-foreground">
      No memory notes recorded for this thread.
    </div>
  );
}
