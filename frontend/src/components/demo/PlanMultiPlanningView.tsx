import { useState, useCallback, useRef } from "react";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  CheckCircle2,
  ArrowRight,
  FileText,
  Clock,
  Sparkles,
  AlertTriangle,
} from "lucide-react";
import { AgentChatPanel } from "@/components/chat/AgentChatPanel";
import type { DemoThread } from "@/data/mockFlowgenticData";
import { getDemoPlan, getDemoMessages, type DemoPlan } from "@/data/mockFlowgenticData";

const complexityColors = {
  low: "text-emerald-400 border-emerald-500/30",
  medium: "text-amber-400 border-amber-500/30",
  high: "text-red-400 border-red-500/30",
} as const;

function FullPlanView({ plan, onSelect }: { plan: DemoPlan; onSelect: () => void }) {
  return (
    <ScrollArea className="flex-1 min-h-0">
      <div className="p-5 space-y-5">
        {/* Plan header */}
        <div className="space-y-3">
          <div className="flex items-start justify-between gap-4">
            <div className="space-y-1 flex-1">
              <h2 className="text-base font-semibold">{plan.summary}</h2>
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <span>by {plan.agentName}</span>
                <span className="text-muted-foreground/30">|</span>
                <span>{plan.agentModel}</span>
              </div>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <Badge
                variant="outline"
                className={`text-[10px] px-1.5 py-0 h-5 ${complexityColors[plan.estimatedComplexity]}`}
              >
                {plan.estimatedComplexity}
              </Badge>
              {plan.estimatedDuration && (
                <Badge
                  variant="outline"
                  className="text-[10px] px-1.5 py-0 h-5 text-muted-foreground border-border gap-0.5"
                >
                  <Clock className="size-2.5" />
                  {plan.estimatedDuration}
                </Badge>
              )}
            </div>
          </div>

          <Button size="sm" className="gap-1.5" onClick={onSelect}>
            <CheckCircle2 className="size-3.5" />
            Select this plan
          </Button>
        </div>

        {/* Approach */}
        <div className="space-y-2">
          <h3 className="text-sm font-medium flex items-center gap-1.5">
            <Sparkles className="size-3.5 text-violet-400" />
            Approach
          </h3>
          <div className="text-sm text-muted-foreground whitespace-pre-line leading-relaxed">
            {plan.approach}
          </div>
        </div>

        {/* Tasks */}
        <div className="space-y-3">
          <h3 className="text-sm font-medium">
            Planned Tasks ({plan.tasks.length})
          </h3>
          <div className="space-y-2">
            {plan.tasks.map((task, i) => (
              <div key={task.id} className="rounded-lg border p-3 space-y-2">
                <div className="flex items-start gap-3">
                  <span className="text-xs font-mono text-muted-foreground mt-0.5 w-5 shrink-0 text-right">
                    {i + 1}.
                  </span>
                  <div className="flex-1 min-w-0 space-y-1.5">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium">{task.name}</span>
                      {task.estimatedDuration && (
                        <span className="text-[10px] text-muted-foreground flex items-center gap-0.5">
                          <Clock className="size-2.5" />
                          {task.estimatedDuration}
                        </span>
                      )}
                      {task.agent && (
                        <span className="text-[10px] text-muted-foreground">
                          ({task.agent})
                        </span>
                      )}
                    </div>
                    <p className="text-xs text-muted-foreground">{task.description}</p>

                    {task.subtasks && task.subtasks.length > 0 && (
                      <div className="space-y-0.5 pt-1">
                        {task.subtasks.map((st, j) => (
                          <div key={j} className="flex items-start gap-1.5 text-xs text-muted-foreground">
                            <span className="mt-1 h-1 w-1 rounded-full bg-muted-foreground/30 shrink-0" />
                            {st}
                          </div>
                        ))}
                      </div>
                    )}

                    {task.dependencies.length > 0 && (
                      <div className="flex items-center gap-1 pt-1">
                        <ArrowRight className="size-3 text-muted-foreground/40" />
                        <span className="text-[10px] text-muted-foreground">
                          depends on:{" "}
                          {task.dependencies
                            .map((d) => plan.tasks.find((t) => t.id === d)?.name ?? d)
                            .join(", ")}
                        </span>
                      </div>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Affected files */}
        {plan.affectedFiles && plan.affectedFiles.length > 0 && (
          <div className="space-y-2">
            <h3 className="text-sm font-medium flex items-center gap-1.5">
              <FileText className="size-3.5 text-blue-400" />
              Affected Files ({plan.affectedFiles.length})
            </h3>
            <div className="rounded-lg border p-3">
              <div className="grid grid-cols-2 gap-1">
                {plan.affectedFiles.map((file) => (
                  <span key={file} className="text-xs text-muted-foreground font-mono truncate">
                    {file}
                  </span>
                ))}
              </div>
            </div>
          </div>
        )}

        {/* Considerations */}
        <div className="space-y-2">
          <h3 className="text-sm font-medium flex items-center gap-1.5">
            <AlertTriangle className="size-3.5 text-amber-400" />
            Considerations
          </h3>
          <div className="rounded-lg border p-3 space-y-1.5">
            {plan.considerations.map((c, i) => (
              <div key={i} className="flex items-start gap-2 text-xs text-muted-foreground">
                <span className="mt-1 h-1 w-1 rounded-full bg-amber-400/50 shrink-0" />
                {c}
              </div>
            ))}
          </div>
        </div>
      </div>
    </ScrollArea>
  );
}

export function PlanMultiPlanningView({ thread }: { thread: DemoThread }) {
  const [leftPanelPercent, setLeftPanelPercent] = useState(40);
  const containerRef = useRef<HTMLDivElement>(null);

  // Build plan tabs from agents that have plans
  const planEntries = thread.agents
    .filter((a) => a.role === "planner")
    .map((agent) => {
      const planKey = `${thread.id}-${agent.id === "agent-claude-planner" ? "claude" : "gpt5"}`;
      const plan = getDemoPlan(planKey);
      return { agent, plan };
    })
    .filter((e): e is { agent: typeof e.agent; plan: DemoPlan } => e.plan !== null);

  // Fall back to overseer plan if no planner plans
  if (planEntries.length === 0) {
    const overseer = thread.agents.find((a) => a.role === "overseer");
    if (overseer) {
      const plan = getDemoPlan(thread.id);
      if (plan) {
        planEntries.push({ agent: overseer, plan });
      }
    }
  }

  const [activeIndex, setActiveIndex] = useState(0);
  const activePlan = planEntries[activeIndex]?.plan;

  // Get chat messages for the overseer
  const overseer = thread.agents.find((a) => a.role === "overseer");
  const overseerMessages = getDemoMessages(`${thread.id}-overseer`);

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      const startX = e.clientX;
      const startPercent = leftPanelPercent;
      const containerWidth = containerRef.current?.offsetWidth ?? 1;

      const handleMouseMove = (moveEvent: MouseEvent) => {
        const deltaX = moveEvent.clientX - startX;
        const deltaPercent = (deltaX / containerWidth) * 100;
        setLeftPanelPercent(Math.min(70, Math.max(25, startPercent + deltaPercent)));
      };

      const handleMouseUp = () => {
        document.removeEventListener("mousemove", handleMouseMove);
        document.removeEventListener("mouseup", handleMouseUp);
      };

      document.addEventListener("mousemove", handleMouseMove);
      document.addEventListener("mouseup", handleMouseUp);
    },
    [leftPanelPercent],
  );

  if (!activePlan) {
    return (
      <div className="flex h-full items-center justify-center text-muted-foreground">
        No plans available
      </div>
    );
  }

  return (
    <div className="flex flex-1 min-h-0" ref={containerRef}>
      {/* Chat panel */}
      <div
        className="flex-shrink-0 h-full overflow-hidden"
        style={{ width: `${leftPanelPercent}%` }}
      >
        <AgentChatPanel
          target={{
            type: "thread_overseer",
            entityId: thread.id,
            agentName: overseer?.name ?? "Overseer",
            title: thread.topic,
            agentColor: overseer?.color ?? "bg-violet-500",
          }}
          hideHeader
          externalMessages={overseerMessages}
        />
      </div>

      {/* Resize handle */}
      <div
        className="w-3 -ml-[6px] -mr-[5px] flex-shrink-0 cursor-col-resize flex justify-center group relative z-10"
        onMouseDown={handleMouseDown}
      >
        <div className="w-px h-full bg-border group-hover:bg-primary/30 transition-colors" />
      </div>

      {/* Right pane: plan tabs + plan content */}
      <div className="min-w-0 flex-1 overflow-hidden flex flex-col">
        {/* Plan tabs */}
        <div className="flex items-center border-b px-3 gap-1 shrink-0">
          {planEntries.map((entry, i) => (
            <button
              key={entry.agent.id}
              type="button"
              onClick={() => setActiveIndex(i)}
              className={cn(
                "flex items-center gap-2 px-3 py-2 text-xs transition-colors border-b-2 -mb-px cursor-pointer",
                activeIndex === i
                  ? "border-primary text-foreground"
                  : "border-transparent text-muted-foreground hover:text-foreground",
              )}
            >
              <span
                className={cn(
                  "h-1.5 w-1.5 rounded-full shrink-0",
                  activeIndex === i ? entry.agent.color : "bg-muted-foreground/40",
                )}
              />
              <span className="font-medium">{entry.agent.name}</span>
              <span className="text-[10px] text-muted-foreground">{entry.agent.model}</span>
              <Badge
                variant="outline"
                className={`text-[10px] px-1 py-0 h-4 ${complexityColors[entry.plan.estimatedComplexity]}`}
              >
                {entry.plan.tasks.length} tasks
              </Badge>
            </button>
          ))}
        </div>

        {/* Active plan content */}
        <FullPlanView plan={activePlan} onSelect={() => {}} />
      </div>
    </div>
  );
}
