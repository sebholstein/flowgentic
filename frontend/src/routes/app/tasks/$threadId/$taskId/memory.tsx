import { createFileRoute } from "@tanstack/react-router";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Markdown } from "@/components/ui/markdown";
import { Brain, Lightbulb, FileText, AlertCircle } from "lucide-react";
import { useTaskContext } from "./context";

export const Route = createFileRoute("/app/tasks/$threadId/$taskId/memory")({
  component: MemoryTab,
});

// Mock task memory data - in a real app this would come from the backend
const mockTaskMemory = {
  summary: `This task sets up the Stripe payment SDK for the application, establishing both server-side and client-side payment infrastructure.

## Key Decisions Made

1. **Server-side SDK**: Using \`stripe\` npm package for secure API operations
2. **Client-side SDK**: Using \`@stripe/stripe-js\` for Elements and checkout flows
3. **Environment Variables**: Added \`STRIPE_SECRET_KEY\` and \`NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY\`

## Implementation Notes

- Created a singleton pattern for the Stripe client to avoid multiple instances
- Added type-safe wrappers around common payment operations
- Configured automatic currency detection based on user locale

## Potential Issues

- Rate limiting not yet implemented on the payment endpoints
- Need to add webhook signature verification in the next iteration`,

  learnings: [
    "The codebase uses Vite for bundling, so environment variables need VITE_ prefix for client-side access",
    "Existing error handling pattern uses a custom Result type - adopted for consistency",
    "Team prefers functional components with hooks over class components",
  ],

  relatedDocs: [
    { title: "Stripe API Documentation", url: "https://stripe.com/docs/api" },
    { title: "Project Payment Architecture", path: "/docs/payments/architecture.md" },
  ],
};

function MemoryTab() {
  const { task } = useTaskContext();

  // In a real app, we'd check if the task has memory
  const hasMemory = task.status === "completed" || task.status === "running";

  if (!hasMemory) {
    return (
      <div className="h-full flex flex-col items-center justify-center text-muted-foreground gap-3">
        <Brain className="size-12 opacity-30" />
        <p>No memory recorded yet.</p>
        <p className="text-sm">Memory notes will appear here once the agent starts working.</p>
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
            <Markdown>{mockTaskMemory.summary}</Markdown>
          </div>
        </div>

        {/* Learnings */}
        {mockTaskMemory.learnings.length > 0 && (
          <div className="rounded-lg border">
            <div className="flex items-center gap-2 p-4 border-b">
              <Lightbulb className="size-4 text-amber-500" />
              <h3 className="font-medium">Learnings</h3>
            </div>
            <div className="p-4 space-y-3">
              {mockTaskMemory.learnings.map((learning, index) => (
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
        {mockTaskMemory.relatedDocs.length > 0 && (
          <div className="rounded-lg border">
            <div className="flex items-center gap-2 p-4 border-b">
              <FileText className="size-4 text-blue-500" />
              <h3 className="font-medium">Related Documentation</h3>
            </div>
            <div className="p-4 space-y-2">
              {mockTaskMemory.relatedDocs.map((doc, index) => (
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
