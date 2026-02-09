import { createFileRoute } from "@tanstack/react-router";
import { ScrollArea } from "@/components/ui/scroll-area";
import { ResourceCard } from "@/components/resources/ResourcePreview";
import { useThreadContext } from "./route";

export const Route = createFileRoute("/app/threads/$threadId/resources")({
  component: ResourcesTab,
});

function ResourcesTab() {
  const { thread } = useThreadContext();

  if (!thread.resources || thread.resources.length === 0) {
    return (
      <div className="h-full flex items-center justify-center text-muted-foreground">
        No resources available for this thread.
      </div>
    );
  }

  return (
    <ScrollArea className="h-full">
      <div className="max-w-4xl mx-auto p-8">
        <div className="space-y-3">
          <p className="text-sm text-muted-foreground">
            Resources available to tasks in this thread.
          </p>
          <div className="grid gap-1.5">
            {thread.resources.map((resource) => (
              <ResourceCard key={resource.resourceId} resource={resource} />
            ))}
          </div>
        </div>
      </div>
    </ScrollArea>
  );
}
