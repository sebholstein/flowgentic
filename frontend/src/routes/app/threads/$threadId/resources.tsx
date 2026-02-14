import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/app/threads/$threadId/resources")({
  component: ResourcesTab,
});

function ResourcesTab() {
  return (
    <div className="h-full flex items-center justify-center text-muted-foreground">
      No resources available for this thread.
    </div>
  );
}
