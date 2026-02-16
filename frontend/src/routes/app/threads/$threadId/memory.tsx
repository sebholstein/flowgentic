import { createFileRoute } from "@tanstack/react-router";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Markdown } from "@/components/ui/markdown";
import { Brain, Lightbulb, FileText, AlertCircle } from "lucide-react";
import { mockThreadMemory } from "@/data/mockThreadMemory";

export const Route = createFileRoute("/app/threads/$threadId/memory")({
  component: MemoryTab,
});

function MemoryTab() {
  // In a real app, we'd check if the thread has any sessions/memory
  const hasMemory = true;

  if (!hasMemory) {
    return (
      <div className="h-full flex flex-col items-center justify-center text-muted-foreground gap-3">
        <Brain className="size-12 opacity-30" />
        <p>No memory recorded yet.</p>
        <p className="text-sm">Memory notes will appear here once agents start working.</p>
      </div>
    );
  }

  return (
    <ScrollArea className="h-full">
      <div className="max-w-3xl mx-auto p-6 space-y-6">
        {/* Agent Summary */}
        <div className="rounded-lg border">
          <div className="flex items-center gap-2 p-4 border-b">
            <Brain className="size-4 text-violet-500" />
            <h3 className="font-medium">Agent Memory</h3>
          </div>
          <div className="p-4">
            <Markdown>{mockThreadMemory.summary}</Markdown>
          </div>
        </div>

        {/* Learnings */}
        {mockThreadMemory.learnings.length > 0 && (
          <div className="rounded-lg border">
            <div className="flex items-center gap-2 p-4 border-b">
              <Lightbulb className="size-4 text-amber-500" />
              <h3 className="font-medium">Learnings</h3>
            </div>
            <div className="p-4 space-y-3">
              {mockThreadMemory.learnings.map((learning, index) => (
                <div
                  key={index}
                  className="flex items-start gap-3 p-3 rounded-lg bg-amber-500/5 border border-amber-500/10"
                >
                  <AlertCircle className="size-4 text-amber-500 mt-0.5 shrink-0" />
                  <p className="text-sm">{learning}</p>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Related Documentation */}
        {mockThreadMemory.relatedDocs.length > 0 && (
          <div className="rounded-lg border">
            <div className="flex items-center gap-2 p-4 border-b">
              <FileText className="size-4 text-blue-500" />
              <h3 className="font-medium">Related Documentation</h3>
            </div>
            <div className="p-4 space-y-2">
              {mockThreadMemory.relatedDocs.map((doc, index) => (
                <a
                  key={index}
                  href={doc.url || doc.path}
                  target={doc.url ? "_blank" : undefined}
                  rel={doc.url ? "noopener noreferrer" : undefined}
                  className="flex items-center gap-3 p-3 rounded-lg border hover:bg-muted/50 transition-colors"
                >
                  <FileText className="size-4 text-muted-foreground" />
                  <span className="text-sm font-medium">{doc.title}</span>
                  <span className="text-xs text-muted-foreground ml-auto">
                    {doc.url ? "External" : doc.path}
                  </span>
                </a>
              ))}
            </div>
          </div>
        )}
      </div>
    </ScrollArea>
  );
}
