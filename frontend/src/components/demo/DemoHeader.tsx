import { Badge } from "@/components/ui/badge";
import { Zap, ClipboardList, Play, Bot } from "lucide-react";
import { cn } from "@/lib/utils";
import type { DemoThread, DemoAgent } from "@/data/mockAgentFlowData";
import { ClaudeIcon, CodexIcon, GeminiIcon } from "@/components/icons/agent-icons";
import type { SVGProps } from "react";

type IconComponent = React.ComponentType<SVGProps<SVGSVGElement>>;

/** Map demo agent IDs to icon components */
const demoAgentIcons: Record<string, IconComponent> = {
  "agent-claude": ClaudeIcon,
  "agent-claude-planner": ClaudeIcon,
  "agent-claude-opus": ClaudeIcon,
  "agent-claude-sonnet": ClaudeIcon,
  "agent-gpt5": CodexIcon,
  "agent-gemini": GeminiIcon,
};

function getDemoAgentIcon(agent: DemoAgent): IconComponent {
  return demoAgentIcons[agent.id] ?? Bot;
}

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

interface DemoHeaderProps {
  thread: DemoThread;
  activeAgentId?: string;
  onSelectAgent?: (agentId: string) => void;
}

export function DemoHeader({ thread, activeAgentId, onSelectAgent }: DemoHeaderProps) {
  const mode = modeConfig[thread.mode];
  const phase = phaseConfig[thread.phase];
  const ModeIcon = mode.icon;

  const resolvedActiveId = activeAgentId ?? thread.agents[0]?.id;

  return (
    <>
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
        <span className="font-medium text-sm truncate">{thread.topic}</span>
      </div>

      {thread.agents.length > 0 && (
        <div className="flex items-center gap-0.5 px-2 h-9 border-b shrink-0 select-none">
          {thread.agents.map((agent) => {
            const Icon = getDemoAgentIcon(agent);
            const isActive = agent.id === resolvedActiveId;
            return (
              <button
                key={agent.id}
                type="button"
                className={cn(
                  "h-full flex items-center gap-1.5 px-2 border-b-2 transition-colors cursor-pointer text-xs",
                  isActive
                    ? "border-primary text-foreground"
                    : "border-transparent text-muted-foreground hover:text-foreground",
                )}
                onClick={() => onSelectAgent?.(agent.id)}
                title={`${agent.name} — ${agent.model}`}
              >
                <Icon className="size-3" />
                <span className="truncate max-w-28">{agent.role === "overseer" ? "Overseer" : `Planner (${agent.name})`}</span>
              </button>
            );
          })}
        </div>
      )}
    </>
  );
}
