import { useState } from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Zap,
  ClipboardList,
  Check,
  ChevronDown,
  Play,
} from "lucide-react";
import {
  demoAvailableAgents,
  demoPlanTemplates,
  type DemoAvailableAgent,
  type DemoPlanTemplate,
} from "@/data/mockAgentFlowData";

type ThreadMode = "quick" | "plan";

function ModeCard({
  label,
  description,
  icon: Icon,
  isSelected,
  onSelect,
}: {
  label: string;
  description: string;
  icon: typeof Zap;
  isSelected: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onSelect}
      className={cn(
        "flex-1 rounded-lg border-2 p-4 text-left transition-all cursor-pointer",
        isSelected
          ? "border-primary bg-primary/5"
          : "border-border hover:border-muted-foreground/30 hover:bg-muted/30",
      )}
    >
      <div className="flex items-center gap-2 mb-2">
        <div
          className={cn(
            "p-1.5 rounded-md",
            isSelected ? "bg-primary/10" : "bg-muted",
          )}
        >
          <Icon
            className={cn(
              "size-4",
              isSelected ? "text-primary" : "text-muted-foreground",
            )}
          />
        </div>
        <span className="text-sm font-medium">{label}</span>
        {isSelected && (
          <Check className="size-3.5 text-primary ml-auto" />
        )}
      </div>
      <p className="text-xs text-muted-foreground leading-relaxed">
        {description}
      </p>
    </button>
  );
}

function AgentCheckbox({
  agent,
  isSelected,
  onToggle,
}: {
  agent: DemoAvailableAgent;
  isSelected: boolean;
  onToggle: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onToggle}
      className={cn(
        "flex items-center gap-3 rounded-lg border p-3 transition-all cursor-pointer w-full text-left",
        isSelected
          ? "border-primary/50 bg-primary/5"
          : "border-border hover:border-muted-foreground/30",
      )}
    >
      <div
        className={cn(
          "size-4 rounded border-2 flex items-center justify-center shrink-0 transition-colors",
          isSelected
            ? "border-primary bg-primary"
            : "border-muted-foreground/40",
        )}
      >
        {isSelected && <Check className="size-2.5 text-primary-foreground" />}
      </div>
      <div
        className={cn(
          "h-6 w-6 rounded-full flex items-center justify-center text-white text-[10px] font-medium shrink-0",
          agent.color,
        )}
      >
        {agent.name[0]}
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-1.5">
          <span className="text-sm font-medium">{agent.name}</span>
          <span className="text-[10px] text-muted-foreground font-mono">
            {agent.model}
          </span>
        </div>
        <p className="text-[11px] text-muted-foreground">{agent.description}</p>
      </div>
    </button>
  );
}

function AgentDropdown({
  agents,
  selectedId,
  onSelect,
}: {
  agents: DemoAvailableAgent[];
  selectedId: string;
  onSelect: (id: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const selected = agents.find((a) => a.id === selectedId) ?? agents[0];

  return (
    <div className="relative">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="flex items-center gap-2 w-full rounded-lg border px-3 py-2.5 text-left transition-colors hover:bg-muted/30 cursor-pointer"
      >
        <div
          className={cn(
            "h-5 w-5 rounded-full flex items-center justify-center text-white text-[9px] font-medium shrink-0",
            selected.color,
          )}
        >
          {selected.name[0]}
        </div>
        <div className="flex-1 min-w-0">
          <span className="text-sm font-medium">{selected.name}</span>
          <span className="text-[10px] text-muted-foreground font-mono ml-1.5">
            {selected.model}
          </span>
        </div>
        <ChevronDown
          className={cn(
            "size-3.5 text-muted-foreground transition-transform",
            open && "rotate-180",
          )}
        />
      </button>

      {open && (
        <div className="absolute top-full left-0 right-0 mt-1 rounded-lg border bg-popover shadow-lg z-20">
          {agents.map((agent) => (
            <button
              key={agent.id}
              type="button"
              onClick={() => {
                onSelect(agent.id);
                setOpen(false);
              }}
              className={cn(
                "flex items-center gap-2 w-full px-3 py-2 text-left transition-colors cursor-pointer first:rounded-t-lg last:rounded-b-lg",
                agent.id === selectedId
                  ? "bg-muted"
                  : "hover:bg-muted/50",
              )}
            >
              <div
                className={cn(
                  "h-5 w-5 rounded-full flex items-center justify-center text-white text-[9px] font-medium shrink-0",
                  agent.color,
                )}
              >
                {agent.name[0]}
              </div>
              <div className="flex-1 min-w-0">
                <span className="text-sm">{agent.name}</span>
                <span className="text-[10px] text-muted-foreground font-mono ml-1.5">
                  {agent.model}
                </span>
              </div>
              {agent.id === selectedId && (
                <Check className="size-3.5 text-primary" />
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

function TemplateSelector({
  templates,
  selectedId,
  onSelect,
}: {
  templates: DemoPlanTemplate[];
  selectedId: string;
  onSelect: (id: string) => void;
}) {
  return (
    <div className="space-y-1.5">
      {templates.map((template) => (
        <button
          key={template.id}
          type="button"
          onClick={() => onSelect(template.id)}
          className={cn(
            "flex items-start gap-3 w-full rounded-lg border p-3 text-left transition-all cursor-pointer",
            selectedId === template.id
              ? "border-primary/50 bg-primary/5"
              : "border-border hover:border-muted-foreground/30",
          )}
        >
          <div
            className={cn(
              "mt-0.5 size-3.5 rounded-full border-2 flex items-center justify-center shrink-0 transition-colors",
              selectedId === template.id
                ? "border-primary bg-primary"
                : "border-muted-foreground/40",
            )}
          >
            {selectedId === template.id && (
              <div className="size-1.5 rounded-full bg-primary-foreground" />
            )}
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-0.5">
              <span className="text-sm font-medium">{template.name}</span>
              <Badge
                variant="outline"
                className="text-[9px] px-1 py-0 h-4 text-muted-foreground border-border"
              >
                {template.source}
              </Badge>
            </div>
            <p className="text-[11px] text-muted-foreground leading-relaxed">
              {template.description}
            </p>
            <div className="flex gap-1 mt-1.5 flex-wrap">
              {template.tags.map((tag) => (
                <Badge
                  key={tag}
                  variant="secondary"
                  className="text-[9px] px-1 py-0 h-4"
                >
                  {tag}
                </Badge>
              ))}
            </div>
          </div>
        </button>
      ))}
    </div>
  );
}

export function NewThreadView() {
  const [mode, setMode] = useState<ThreadMode>("plan");
  const [prompt, setPrompt] = useState("");
  const [selectedAgentId, setSelectedAgentId] = useState(
    demoAvailableAgents[0].id,
  );
  const [selectedPlannerIds, setSelectedPlannerIds] = useState<Set<string>>(
    () => new Set([demoAvailableAgents[0].id]),
  );
  const [selectedTemplateId, setSelectedTemplateId] = useState(
    demoPlanTemplates[0].id,
  );

  const togglePlanner = (id: string) => {
    setSelectedPlannerIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        if (next.size > 1) next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const plannerCount = selectedPlannerIds.size;
  const plannerHint =
    plannerCount === 1
      ? "Single plan proposed for approval"
      : `${plannerCount} competing plans compared side-by-side`;

  return (
    <ScrollArea className="flex-1 min-h-0">
      <div className="max-w-2xl py-8 px-8 space-y-6">
        {/* Title */}
        <div>
          <h1 className="text-lg font-semibold">New Thread</h1>
          <p className="text-sm text-muted-foreground mt-1">
            Describe what you want to accomplish and choose how to approach it.
          </p>
        </div>

        {/* Prompt input */}
        <div className="space-y-1.5">
          <label
            htmlFor="thread-prompt"
            className="text-xs font-medium text-muted-foreground"
          >
            What do you want to do?
          </label>
          <textarea
            id="thread-prompt"
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            placeholder="e.g. Build a payment processing system with Stripe integration..."
            className="w-full rounded-lg border bg-background px-3 py-2.5 text-sm resize-none focus:outline-none focus:ring-2 focus:ring-ring min-h-[80px]"
            rows={3}
          />
        </div>

        {/* Mode selection */}
        <div className="space-y-2">
          <span className="text-xs font-medium text-muted-foreground">
            Mode
          </span>
          <div className="flex gap-3">
            <ModeCard

              label="Quick"
              description="Direct chat + execute. Best for simple, atomic requests."
              icon={Zap}
              isSelected={mode === "quick"}
              onSelect={() => setMode("quick")}
            />
            <ModeCard

              label="Plan"
              description="Agent(s) create a plan first. Best for complex, multi-step work."
              icon={ClipboardList}
              isSelected={mode === "plan"}
              onSelect={() => setMode("plan")}
            />
          </div>
        </div>

        {/* Agent selection */}
        {mode === "quick" ? (
          <div className="space-y-2">
            <span className="text-xs font-medium text-muted-foreground">
              Agent
            </span>
            <AgentDropdown
              agents={demoAvailableAgents}
              selectedId={selectedAgentId}
              onSelect={setSelectedAgentId}
            />
          </div>
        ) : (
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium text-muted-foreground">
                Planning agents
              </span>
              <span className="text-[10px] text-muted-foreground">
                {plannerHint}
              </span>
            </div>
            <div className="space-y-1.5">
              {demoAvailableAgents.map((agent) => (
                <AgentCheckbox
                  key={agent.id}
                  agent={agent}
                  isSelected={selectedPlannerIds.has(agent.id)}
                  onToggle={() => togglePlanner(agent.id)}
                />
              ))}
            </div>
          </div>
        )}

        {/* Template selection (Plan mode only) */}
        {mode === "plan" && (
          <div className="space-y-2">
            <span className="text-xs font-medium text-muted-foreground">
              Plan template
            </span>
            <TemplateSelector
              templates={demoPlanTemplates}
              selectedId={selectedTemplateId}
              onSelect={setSelectedTemplateId}
            />
          </div>
        )}

        {/* Start button */}
        <div className="pt-2">
          <Button className="w-full gap-2" size="lg" disabled={!prompt.trim()}>
            <Play className="size-4" />
            {mode === "quick" ? "Start" : "Start Planning"}
          </Button>
        </div>
      </div>
    </ScrollArea>
  );
}
