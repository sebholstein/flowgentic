import { useState } from "react";
import { Compass, Check, Send, FileCode, ArrowUpRight, ChevronDown } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import type { InboxItem } from "@/types/inbox";

interface DirectionClarificationProps {
  inboxItem: InboxItem;
  onSubmit?: (response: string, delegateToOverseer?: boolean) => void;
}

export function DirectionClarification({ inboxItem, onSubmit }: DirectionClarificationProps) {
  const context = inboxItem.clarificationContext;
  const [response, setResponse] = useState(inboxItem.clarificationResponse ?? "");
  const [isSubmitted, setIsSubmitted] = useState(inboxItem.status === "resolved");
  const [codeOpen, setCodeOpen] = useState(false);
  const [selectedApproach, setSelectedApproach] = useState<string | null>(null);

  const handleSubmit = (delegateToOverseer = false) => {
    if (!response.trim() && !delegateToOverseer) return;
    setIsSubmitted(true);
    onSubmit?.(response, delegateToOverseer);
  };

  const handleApproachSelect = (approachId: string, description: string) => {
    setSelectedApproach(approachId);
    // Pre-fill the response with the selected approach
    setResponse((prev) =>
      prev ? `${prev}\n\nSelected approach: ${description}` : `Selected approach: ${description}`,
    );
  };

  return (
    <div className="rounded-lg border border-border bg-card/50 p-4 space-y-4">
      {/* Card header */}
      <div className="flex items-center gap-2">
        <div className="rounded-md bg-blue-500/10 p-1.5">
          <Compass className="size-4 text-blue-400" />
        </div>
        <div className="min-w-0 flex-1">
          <h3 className="text-sm font-semibold truncate">{inboxItem.title}</h3>
          <p className="text-xs text-muted-foreground truncate">{inboxItem.description}</p>
        </div>
        <Badge className="bg-blue-500/20 text-blue-400 border-blue-500/30 text-xs">
          Guidance Needed
        </Badge>
      </div>

      {/* Current understanding */}
      {context?.currentUnderstanding && (
        <div className="space-y-1.5">
          <span className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
            Agent's Current Understanding
          </span>
          <div className="rounded-md border border-border bg-muted/50 p-3">
            <p className="text-sm text-foreground leading-relaxed whitespace-pre-wrap">
              {context.currentUnderstanding}
            </p>
          </div>
        </div>
      )}

      {/* Relevant files */}
      {context?.relevantFiles && context.relevantFiles.length > 0 && (
        <div className="space-y-1.5">
          <span className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
            Relevant Files
          </span>
          <div className="flex flex-wrap gap-1.5">
            {context.relevantFiles.map((file) => (
              <Badge key={file} variant="outline" className="text-xs font-mono">
                <FileCode className="size-3 mr-1" />
                {file}
              </Badge>
            ))}
          </div>
        </div>
      )}

      {/* Relevant code */}
      {context?.relevantCode && (
        <Collapsible open={codeOpen} onOpenChange={setCodeOpen}>
          <CollapsibleTrigger className="flex items-center gap-2 text-xs text-muted-foreground hover:text-foreground transition-colors">
            <FileCode className="size-3.5" />
            <span>View relevant code</span>
            <ChevronDown
              className={cn("size-3.5 transition-transform", codeOpen && "rotate-180")}
            />
          </CollapsibleTrigger>
          <CollapsibleContent>
            <pre className="mt-2 p-3 bg-muted rounded-md text-xs text-foreground overflow-x-auto max-h-48 overflow-y-auto">
              <code>{context.relevantCode}</code>
            </pre>
          </CollapsibleContent>
        </Collapsible>
      )}

      {/* Approach options */}
      {context?.approachOptions && context.approachOptions.length > 0 && (
        <div className="space-y-2">
          <span className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
            Possible Approaches
          </span>
          <div className="space-y-2">
            {context.approachOptions.map((approach) => {
              const isSelected = selectedApproach === approach.id;
              return (
                <button
                  key={approach.id}
                  type="button"
                  disabled={isSubmitted}
                  onClick={() => handleApproachSelect(approach.id, approach.description)}
                  className={cn(
                    "w-full rounded-lg border p-3 text-left transition-all",
                    isSelected
                      ? "border-primary bg-primary/5 ring-1 ring-primary"
                      : "border-border hover:border-muted-foreground/50 hover:bg-muted/50",
                    isSubmitted && "cursor-not-allowed opacity-60",
                  )}
                >
                  <div className="flex items-start gap-3">
                    <div
                      className={cn(
                        "flex size-5 shrink-0 items-center justify-center rounded-full border mt-0.5",
                        isSelected
                          ? "border-primary bg-primary text-primary-foreground"
                          : "border-muted-foreground/40",
                      )}
                    >
                      {isSelected && <Check className="size-3" />}
                    </div>
                    <div className="flex-1 min-w-0 space-y-1">
                      <p className="text-sm text-foreground">{approach.description}</p>
                      {approach.tradeoffs && (
                        <p className="text-xs text-muted-foreground">
                          <span className="font-medium">Trade-offs:</span> {approach.tradeoffs}
                        </p>
                      )}
                    </div>
                  </div>
                </button>
              );
            })}
          </div>
        </div>
      )}

      {/* Guidance response */}
      <div className="space-y-1.5 pt-2 border-t">
        <label className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
          Your Guidance
        </label>
        <Textarea
          placeholder="Provide direction on how the agent should proceed..."
          value={response}
          onChange={(e) => setResponse(e.target.value)}
          disabled={isSubmitted}
          rows={3}
          className="text-xs resize-none"
        />
      </div>

      {/* Submitted state or action buttons */}
      {isSubmitted ? (
        <div className="rounded-md border border-emerald-500/30 bg-emerald-500/10 px-3 py-2">
          <div className="flex items-center gap-1.5 text-emerald-400">
            <Check className="size-3.5" />
            <span className="text-xs font-medium">
              {inboxItem.delegatedToIssueOverseer
                ? "Delegated to issue overseer"
                : "Guidance provided"}
            </span>
          </div>
          {response && (
            <p className="text-xs text-muted-foreground mt-1 whitespace-pre-wrap">{response}</p>
          )}
        </div>
      ) : (
        <div className="space-y-2">
          <Button
            onClick={() => handleSubmit(false)}
            disabled={!response.trim()}
            className="w-full"
            size="sm"
          >
            <Send className="size-3.5" />
            Send Guidance
          </Button>
          <Button
            variant="outline"
            onClick={() => handleSubmit(true)}
            className="w-full text-muted-foreground"
            size="sm"
          >
            <ArrowUpRight className="size-3.5" />
            Delegate to Issue Overseer
          </Button>
        </div>
      )}
    </div>
  );
}
