import { createFileRoute } from "@tanstack/react-router";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Markdown } from "@/components/ui/markdown";
import { useThreadContext } from "./route";

export const Route = createFileRoute("/app/threads/$threadId/memory")({
  component: MemoryTab,
});

function MemoryTab() {
  const { thread } = useThreadContext();

  if (!thread.memory) {
    return (
      <div className="h-full flex items-center justify-center text-muted-foreground">
        No memory notes recorded for this thread.
      </div>
    );
  }

  return (
    <ScrollArea className="h-full">
      <div className="max-w-4xl mx-auto p-8">
        <p className="text-sm text-foreground/60 mb-4">Written by {thread.overseer.name}</p>
        <Markdown>{thread.memory}</Markdown>
      </div>
    </ScrollArea>
  );
}
