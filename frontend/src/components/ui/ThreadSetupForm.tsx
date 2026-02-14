import { cn } from "@/lib/utils";
import { ServerStatusDot } from "@/components/servers/ServerStatusDot";
import {
  Bot,
  Cpu,
  Folder,
  Shield,
  ShieldOff,
  User,
  Users,
} from "lucide-react";
import { Agent } from "@/proto/gen/worker/v1/agent_pb";
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

interface ThreadSetupFormProps {
  threadMode: "plan" | "build";
  onModeChange: (mode: "plan" | "build") => void;
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
  sessionMode: string;
  onSessionModeChange: (mode: string) => void;
  availableModels: string[];
  defaultModel: string;
  modelsLoading: boolean;
  modelsError: string | null;
}

const availableAgents: { id: Agent; name: string; icon: typeof ClaudeIcon }[] = [
  { id: Agent.CLAUDE_CODE, name: "Claude Code", icon: ClaudeIcon },
  { id: Agent.CODEX, name: "Codex", icon: CodexIcon },
  { id: Agent.OPENCODE, name: "OpenCode", icon: OpenCodeIcon },
  { id: Agent.AMP, name: "Amp", icon: AmpIcon },
  { id: Agent.GEMINI, name: "Gemini", icon: GeminiIcon },
];

export function ThreadSetupForm({
  threadMode,
  onModeChange,
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
  sessionMode,
  onSessionModeChange,
  availableModels,
  defaultModel,
  modelsLoading,
  modelsError,
}: ThreadSetupFormProps) {
  const modelItems = availableModels.map((model) => ({
    id: model,
    name: model,
    icon: <Bot className="size-3" />,
  }));

  return (
    <div className="w-full max-w-[480px] grid grid-cols-2 gap-x-3 gap-y-3 mb-5 text-left">
      {project && projects && (
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
      <ConfigField label="Mode">
        <SearchableSelect
          items={[
            { id: "plan", name: "Plan", icon: <Users className="size-3" /> },
            { id: "build", name: "Build", icon: <User className="size-3" /> },
          ]}
          selectedId={threadMode}
          onSelect={(id) => onModeChange(id as "plan" | "build")}
          placeholder="Search modes…"
        />
      </ConfigField>
      <ConfigField label="Session Mode">
        <SearchableSelect
          items={[
            { id: "ask", name: "Ask", icon: <Shield className="size-3" /> },
            { id: "architect", name: "Architect", icon: <Shield className="size-3" /> },
            { id: "code", name: "Code", icon: <ShieldOff className="size-3" /> },
          ]}
          selectedId={sessionMode}
          onSelect={onSessionModeChange}
          placeholder="Search modes…"
        />
      </ConfigField>
      {workers.length > 0 && (
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
      <ConfigField label="Agent">
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
      <ConfigField label="Model">
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
            placeholder={defaultModel || "model-id"}
            className="h-7 w-full rounded-md border border-input bg-input/20 dark:bg-input/30 px-2 text-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
          />
        )}
        {modelsError && (
          <span className="text-[11px] text-muted-foreground">{modelsError}</span>
        )}
      </ConfigField>
    </div>
  );
}

function ConfigField({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <span className="text-[0.6rem] font-medium uppercase tracking-wider text-muted-foreground/60">
        {label}
      </span>
      {children}
    </div>
  );
}
