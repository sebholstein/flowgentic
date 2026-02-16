import { cn } from "@/lib/utils";
import { ServerStatusDot } from "@/components/servers/ServerStatusDot";
import { Bot, Cpu, Folder } from "lucide-react";
import { Agent } from "@/proto/gen/worker/v1/agent_pb";
import type { ModelInfo } from "@/proto/gen/worker/v1/system_service_pb";
import type { Project } from "@/types/project";
import type { Worker } from "@/types/server";
import {
  AmpIcon,
  ClaudeIcon,
  CodexIcon,
  GeminiIcon,
  OpenCodeIcon,
} from "@/components/icons/agent-icons";
import { SearchableSelect } from "@/components/ui/searchable-select";
import { Button } from "@/components/ui/button";

interface ThreadSetupFormProps {
  threadModel: string;
  onModelChange: (model: string) => void;
  project?: Project;
  projects?: Project[];
  onProjectChange: (projectId: string) => void;
  workerId: string;
  workers: Worker[];
  onWorkerChange: (workerId: string) => void;
  agent: Agent;
  onAgentChange: (agent: Agent) => void;
  availableModels: ModelInfo[];
  defaultModel: string;
  modelsLoading: boolean;
  modelsError: string | null;
  /** Hide project and worker fields (e.g. in the new-session popover) */
  compact?: boolean;
  /** Show a submit button with this label */
  submitLabel?: string;
  onSubmit?: () => void;
}

const availableAgents: { id: Agent; name: string; icon: typeof ClaudeIcon }[] = [
  { id: Agent.CLAUDE_CODE, name: "Claude Code", icon: ClaudeIcon },
  { id: Agent.CODEX, name: "Codex", icon: CodexIcon },
  { id: Agent.OPENCODE, name: "OpenCode", icon: OpenCodeIcon },
  { id: Agent.AMP, name: "Amp", icon: AmpIcon },
  { id: Agent.GEMINI, name: "Gemini", icon: GeminiIcon },
];

export function ThreadSetupForm({
  threadModel,
  onModelChange,
  project,
  projects,
  onProjectChange,
  workerId,
  workers,
  onWorkerChange,
  agent,
  onAgentChange,
  availableModels,
  defaultModel,
  modelsLoading,
  modelsError,
  compact,
  submitLabel,
  onSubmit,
}: ThreadSetupFormProps) {
  const modelItems = availableModels.map((model) => ({
    id: model.id,
    name: model.displayName || model.id,
    description: model.description,
    icon: <Bot className="size-3" />,
  }));

  return (
    <div className="w-full max-w-[480px] grid grid-cols-2 gap-x-3 gap-y-3 mb-5 text-left">
      {!compact && project && projects && (
        <div className="col-span-2">
          <ConfigField label="Project">
            <SearchableSelect
              items={projects.map((p) => ({
                id: p.id,
                name: p.name,
                icon: <Folder className={cn("size-3", p.color ?? "text-amber-400")} />,
              }))}
              selectedId={project.id}
              onSelect={onProjectChange}
              placeholder="Search projects…"
            />
          </ConfigField>
        </div>
      )}
      {!compact && workers.length > 0 && (
        <ConfigField label="Worker">
          <SearchableSelect
            items={workers.map((w) => ({
              id: w.id,
              name: w.name,
              icon: <Cpu className="size-3" />,
              trailing: <ServerStatusDot status={w.status} />,
            }))}
            selectedId={workerId}
            onSelect={onWorkerChange}
            placeholder="Search workers…"
          />
        </ConfigField>
      )}
      <ConfigField label="Agent" className={compact ? "col-span-2" : undefined}>
        <SearchableSelect
          items={availableAgents.map((a) => ({
            id: String(a.id),
            name: a.name,
            icon: <a.icon className="size-3" />,
          }))}
          selectedId={String(agent)}
          onSelect={(id) => onAgentChange(Number(id) as Agent)}
          placeholder="Search agents…"
        />
      </ConfigField>
      <ConfigField label="Model" className="col-span-2">
        {modelsLoading ? (
          <input
            type="text"
            value="Loading models..."
            readOnly
            className="h-7 w-full rounded-md border border-input bg-input/20 dark:bg-input/30 px-2 text-xs text-muted-foreground focus-visible:outline-none"
          />
        ) : modelItems.length > 0 ? (
          <SearchableSelect
            items={modelItems}
            selectedId={threadModel || defaultModel || modelItems[0]?.id || ""}
            onSelect={onModelChange}
            placeholder="Search models…"
          />
        ) : (
          <input
            type="text"
            value={threadModel}
            onChange={(e) => onModelChange(e.target.value)}
            placeholder={defaultModel || "default"}
            className="h-7 w-full rounded-md border border-input bg-input/20 dark:bg-input/30 px-2 text-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
          />
        )}
      </ConfigField>
      {onSubmit && (
        <div className="col-span-2">
          <Button size="sm" className="w-full" onClick={onSubmit}>
            {submitLabel ?? "Create Session"}
          </Button>
        </div>
      )}
    </div>
  );
}

function ConfigField({ label, className, children }: { label: string; className?: string; children: React.ReactNode }) {
  return (
    <div className={cn("flex flex-col gap-1", className)}>
      <span className="text-[0.6rem] font-medium uppercase tracking-wider text-muted-foreground/60">
        {label}
      </span>
      {children}
    </div>
  );
}
