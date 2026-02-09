import { createFileRoute } from "@tanstack/react-router";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Markdown } from "@/components/ui/markdown";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { Clock, Zap, Calendar, ClipboardList, Check, X, Bot } from "lucide-react";
import { Button } from "@/components/ui/button";
import { planStatusConfig } from "@/constants/taskStatusConfig";
import { taskStatusConfig } from "@/constants/taskStatusConfig";
import { useTaskContext } from "./context";

export const Route = createFileRoute("/app/tasks/$threadId/$taskId/")({
  component: OverviewTab,
});

// Mock task plan - in a real app this would come from the task data
const mockTaskPlan = `## Objective

Set up the Stripe payment SDK for both server-side and client-side usage, enabling the application to process payments securely.

## Implementation Plan

### 1. Install Dependencies

Install the required Stripe packages:
- \`stripe\` - Server-side SDK for API operations
- \`@stripe/stripe-js\` - Client-side SDK for Elements and Checkout

### 2. Server-Side Setup

Create a server-side Stripe client with:
- Singleton pattern to avoid multiple instances
- Type-safe configuration
- Environment variable validation

### 3. Client-Side Setup

Create a client-side loader that:
- Lazily loads Stripe.js for performance
- Memoizes the Stripe promise
- Handles loading errors gracefully

### 4. Environment Configuration

Add required environment variables:
- \`STRIPE_SECRET_KEY\` - Server-side API key
- \`NEXT_PUBLIC_STRIPE_PUBLISHABLE_KEY\` - Client-side publishable key

### 5. Testing

- Verify TypeScript compilation
- Run unit tests for both clients
- Test with Stripe test mode credentials

## Acceptance Criteria

- [ ] Dependencies installed without conflicts
- [ ] Server client initializes correctly
- [ ] Client loader works in browser
- [ ] Environment variables documented
- [ ] All tests passing
`;

function OverviewTab() {
  const { task } = useTaskContext();
  const statusConfig = taskStatusConfig[task.status];

  return (
    <ScrollArea className="h-full">
      <div className="max-w-3xl mx-auto p-6 space-y-6">
        {/* Task Header */}
        <div>
          <div className="flex items-center gap-2 mb-2">
            <Badge className={cn("text-xs", statusConfig.bgColor, statusConfig.color)}>
              {statusConfig.label}
            </Badge>
            {task.agent && (
              <span className="text-xs text-muted-foreground">Assigned to {task.agent}</span>
            )}
          </div>
          <h1 className="text-2xl font-bold mb-2">{task.name}</h1>
          <p className="text-muted-foreground">{task.description}</p>
        </div>

        {/* Task Metadata */}
        <div className="flex flex-wrap items-center gap-4 text-sm text-muted-foreground border-y py-3">
          {task.duration && (
            <div className="flex items-center gap-1.5">
              <Clock className="size-4" />
              <span>{task.duration}</span>
            </div>
          )}
          {task.startedAt && (
            <div className="flex items-center gap-1.5">
              <Calendar className="size-4" />
              <span>Started {task.startedAt}</span>
            </div>
          )}
          {task.executions && task.executions.length > 0 && (
            <div className="flex items-center gap-1.5">
              <Zap className="size-4" />
              <span>{task.executions.length} execution(s)</span>
            </div>
          )}
        </div>

        {/* Task Planning Section */}
        {task.plannerPrompt && (
          <div className="rounded-lg border">
            <div className="px-4 py-3 border-b bg-muted/30 flex items-center justify-between">
              <div className="flex items-center gap-2">
                <ClipboardList className="size-4 text-muted-foreground" />
                <h2 className="font-medium">Task Plan</h2>
              </div>
              {task.planStatus &&
                task.planStatus !== "skipped" &&
                (() => {
                  const config = planStatusConfig[task.planStatus];
                  if (!config.icon) return null;
                  const PlanStatusIcon = config.icon;
                  return (
                    <Badge className={cn("text-xs gap-1", config.bgColor, config.color)}>
                      <PlanStatusIcon className="size-3" />
                      {config.label}
                    </Badge>
                  );
                })()}
            </div>
            <div className="p-4 space-y-4">
              {/* Planner prompt */}
              <div className="space-y-1.5">
                <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                  Planner Instructions
                </span>
                <div className="rounded-md border border-border bg-muted/30 p-3">
                  <p className="text-sm text-foreground leading-relaxed">{task.plannerPrompt}</p>
                </div>
              </div>

              {/* Planner info */}
              {task.planner && (
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Bot className="size-4" />
                  <span>
                    Planned by{" "}
                    <span className="font-medium text-foreground">{task.planner.name}</span>
                  </span>
                  {task.planApproval && (
                    <Badge variant="outline" className="text-xs ml-2">
                      {task.planApproval === "user"
                        ? "User approves"
                        : task.planApproval === "overseer"
                          ? "Overseer approves"
                          : "Auto-approved"}
                    </Badge>
                  )}
                </div>
              )}

              {/* Plan output */}
              {task.plan && (
                <div className="space-y-3">
                  {/* Summary */}
                  <div className="space-y-1">
                    <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                      Summary
                    </span>
                    <p className="text-sm">{task.plan.summary}</p>
                  </div>

                  {/* Steps */}
                  {task.plan.steps.length > 0 && (
                    <div className="space-y-1.5">
                      <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                        Steps
                      </span>
                      <ol className="space-y-1.5">
                        {task.plan.steps.map((step, i) => (
                          <li key={i} className="flex items-start gap-2 text-sm">
                            <span className="flex size-5 shrink-0 items-center justify-center rounded bg-muted text-[0.65rem] font-medium text-muted-foreground mt-0.5">
                              {i + 1}
                            </span>
                            <span className="leading-relaxed">{step}</span>
                          </li>
                        ))}
                      </ol>
                    </div>
                  )}

                  {/* Approach */}
                  {task.plan.approach && (
                    <div className="space-y-1">
                      <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                        Approach
                      </span>
                      <p className="text-sm text-muted-foreground">{task.plan.approach}</p>
                    </div>
                  )}

                  {/* Considerations */}
                  {task.plan.considerations && task.plan.considerations.length > 0 && (
                    <div className="space-y-1.5">
                      <span className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                        Considerations
                      </span>
                      <ul className="space-y-1">
                        {task.plan.considerations.map((item, i) => (
                          <li
                            key={i}
                            className="flex items-start gap-2 text-sm text-muted-foreground"
                          >
                            <span className="text-primary mt-0.5">•</span>
                            <span>{item}</span>
                          </li>
                        ))}
                      </ul>
                    </div>
                  )}

                  {/* Complexity */}
                  {task.plan.estimatedComplexity && (
                    <div className="flex items-center gap-2 text-sm">
                      <span className="text-muted-foreground">Complexity:</span>
                      <Badge
                        variant="outline"
                        className={cn(
                          "text-xs",
                          task.plan.estimatedComplexity === "low" &&
                            "text-emerald-400 border-emerald-500/30",
                          task.plan.estimatedComplexity === "medium" &&
                            "text-amber-400 border-amber-500/30",
                          task.plan.estimatedComplexity === "high" &&
                            "text-red-400 border-red-500/30",
                        )}
                      >
                        {task.plan.estimatedComplexity}
                      </Badge>
                    </div>
                  )}
                </div>
              )}

              {/* Approve/Reject buttons for awaiting_approval + user approval */}
              {task.planStatus === "awaiting_approval" && task.planApproval === "user" && (
                <div className="flex gap-2 pt-2 border-t">
                  <Button variant="outline" size="sm" className="flex-1">
                    <X className="size-3.5" />
                    Reject Plan
                  </Button>
                  <Button size="sm" className="flex-1">
                    <Check className="size-3.5" />
                    Approve Plan
                  </Button>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Task Plan (fallback for tasks without planning data) */}
        {!task.plannerPrompt && (
          <div className="rounded-lg border">
            <div className="px-4 py-3 border-b bg-muted/30">
              <h2 className="font-medium">Task Plan</h2>
            </div>
            <div className="p-4">
              <Markdown>{mockTaskPlan}</Markdown>
            </div>
          </div>
        )}
      </div>
    </ScrollArea>
  );
}
