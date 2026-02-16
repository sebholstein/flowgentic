import { useState } from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  CheckCircle2,
  XCircle,
  FileText,
  ListChecks,
  GitBranch,
  ArrowRight,
  Clock,
} from "lucide-react";
import type { DemoPlan } from "@/data/mockFlowgenticData";

type PlanTab = "description" | "tasks" | "graph";

const complexityColors = {
  low: "text-emerald-400 border-emerald-500/30",
  medium: "text-amber-400 border-amber-500/30",
  high: "text-red-400 border-red-500/30",
} as const;

export function PlanProposalPanel({ plan }: { plan: DemoPlan }) {
  const [activeTab, setActiveTab] = useState<PlanTab>("description");

  const tabs: { id: PlanTab; label: string; icon: typeof FileText }[] = [
    { id: "description", label: "Description", icon: FileText },
    { id: "tasks", label: "Tasks", icon: ListChecks },
    { id: "graph", label: "Graph", icon: GitBranch },
  ];

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex items-center justify-between border-b px-4 py-2.5">
        <div className="flex items-center gap-2">
          <span className="text-xs font-medium">Plan Proposal</span>
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
        <span className="text-xs text-muted-foreground">by {plan.agentName}</span>
      </div>

      {/* Sub-tabs */}
      <div className="flex border-b px-4 gap-1">
        {tabs.map((tab) => {
          const Icon = tab.icon;
          return (
            <button
              key={tab.id}
              type="button"
              onClick={() => setActiveTab(tab.id)}
              className={cn(
                "flex items-center gap-1 px-2 py-2 text-xs transition-colors border-b-2 -mb-px cursor-pointer",
                activeTab === tab.id
                  ? "border-primary text-foreground"
                  : "border-transparent text-muted-foreground hover:text-foreground",
              )}
            >
              <Icon className="size-3" />
              {tab.label}
            </button>
          );
        })}
      </div>

      {/* Content */}
      <ScrollArea className="flex-1 min-h-0">
        <div className="p-4 space-y-4">
          {activeTab === "description" && (
            <>
              <div>
                <h4 className="text-xs font-medium text-muted-foreground mb-1.5">Summary</h4>
                <p className="text-sm">{plan.summary}</p>
              </div>
              <div>
                <h4 className="text-xs font-medium text-muted-foreground mb-1.5">Approach</h4>
                <p className="text-sm whitespace-pre-line">{plan.approach}</p>
              </div>
              {plan.affectedFiles && plan.affectedFiles.length > 0 && (
                <div>
                  <h4 className="text-xs font-medium text-muted-foreground mb-1.5">
                    Affected Files ({plan.affectedFiles.length})
                  </h4>
                  <div className="space-y-0.5">
                    {plan.affectedFiles.map((file) => (
                      <div key={file} className="text-xs text-muted-foreground font-mono">
                        {file}
                      </div>
                    ))}
                  </div>
                </div>
              )}
              <div>
                <h4 className="text-xs font-medium text-muted-foreground mb-1.5">Considerations</h4>
                <ul className="space-y-1">
                  {plan.considerations.map((c, i) => (
                    <li key={i} className="text-sm text-muted-foreground flex items-start gap-1.5">
                      <span className="mt-1.5 h-1 w-1 rounded-full bg-muted-foreground/50 shrink-0" />
                      {c}
                    </li>
                  ))}
                </ul>
              </div>
            </>
          )}

          {activeTab === "tasks" && (
            <div className="space-y-3">
              {plan.tasks.map((task, i) => (
                <div
                  key={task.id}
                  className="rounded-lg border p-3 space-y-1.5"
                >
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-medium text-muted-foreground w-5">
                      {i + 1}.
                    </span>
                    <span className="text-sm font-medium flex-1">{task.name}</span>
                    {task.estimatedDuration && (
                      <span className="text-[10px] text-muted-foreground flex items-center gap-0.5">
                        <Clock className="size-2.5" />
                        {task.estimatedDuration}
                      </span>
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground pl-7">{task.description}</p>

                  {/* Subtasks */}
                  {task.subtasks && task.subtasks.length > 0 && (
                    <div className="pl-7 space-y-0.5 pt-0.5">
                      {task.subtasks.map((st, j) => (
                        <div key={j} className="flex items-start gap-1.5 text-xs text-muted-foreground">
                          <span className="mt-1 h-1 w-1 rounded-full bg-muted-foreground/30 shrink-0" />
                          {st}
                        </div>
                      ))}
                    </div>
                  )}

                  {task.dependencies.length > 0 && (
                    <div className="flex items-center gap-1 pl-7">
                      <ArrowRight className="size-3 text-muted-foreground/50" />
                      <span className="text-[10px] text-muted-foreground">
                        depends on: {task.dependencies.map((d) => {
                          const dep = plan.tasks.find((t) => t.id === d);
                          return dep?.name ?? d;
                        }).join(", ")}
                      </span>
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}

          {activeTab === "graph" && (
            <div className="flex items-center justify-center h-48 rounded-lg border border-dashed text-muted-foreground text-xs">
              Task dependency graph visualization
            </div>
          )}
        </div>
      </ScrollArea>

      {/* Actions */}
      <div className="flex items-center gap-2 border-t px-4 py-3">
        <Button size="sm" className="flex-1 gap-1.5">
          <CheckCircle2 className="size-3.5" />
          Approve Plan
        </Button>
        <Button size="sm" variant="outline" className="flex-1 gap-1.5">
          <XCircle className="size-3.5" />
          Reject
        </Button>
      </div>
    </div>
  );
}
