import { createFileRoute } from "@tanstack/react-router";
import { GitBranch } from "lucide-react";

export const Route = createFileRoute("/app/threads/")({
  component: ThreadsIndexPage,
});

function ThreadsIndexPage() {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-4 text-muted-foreground">
      <GitBranch className="size-12 opacity-20" />
      <div className="text-center">
        <p className="text-sm font-medium">No thread selected</p>
        <p className="text-xs">
          Select a thread from the sidebar to view its details and task graph
        </p>
      </div>
    </div>
  );
}
