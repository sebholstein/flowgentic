import { Badge } from "@/components/ui/badge";
import { Zap, ClipboardList, Play } from "lucide-react";
import type { DemoThread } from "@/data/mockAgentFlowData";

const modeConfig = {
  quick: { label: "Quick", icon: Zap, color: "text-emerald-400 border-emerald-500/30" },
  plan: { label: "Plan", icon: ClipboardList, color: "text-violet-400 border-violet-500/30" },
} as const;

const phaseConfig = {
  creation: { label: "Creating", color: "text-cyan-400 border-cyan-500/30" },
  completed: { label: "Completed", color: "text-emerald-400 border-emerald-500/30" },
  planning: { label: "Planning", color: "text-orange-400 border-orange-500/30" },
  execution: { label: "Executing", color: "text-blue-400 border-blue-500/30" },
} as const;

export function DemoHeader({ thread }: { thread: DemoThread }) {
  const mode = modeConfig[thread.mode];
  const phase = phaseConfig[thread.phase];
  const ModeIcon = mode.icon;

  return (
    <div className="flex items-center gap-2 px-4 h-10 border-b shrink-0 select-none">
      <Badge
        variant="outline"
        className={`text-[10px] px-1.5 py-0 h-5 shrink-0 font-medium gap-0.5 ${mode.color}`}
      >
        <ModeIcon className="size-2.5" />
        {mode.label}
      </Badge>
      <Badge
        variant="outline"
        className={`text-[10px] px-1.5 py-0 h-5 shrink-0 font-medium gap-0.5 ${phase.color}`}
      >
        {thread.phase === "execution" && <Play className="size-2.5" />}
        {phase.label}
      </Badge>
      {thread.agents.length > 1 && (
        <Badge
          variant="outline"
          className="text-[10px] px-1.5 py-0 h-5 shrink-0 font-medium text-slate-400 border-slate-500/30"
        >
          {thread.agents.length} agents
        </Badge>
      )}
      <span className="font-medium text-sm truncate">{thread.topic}</span>
    </div>
  );
}
