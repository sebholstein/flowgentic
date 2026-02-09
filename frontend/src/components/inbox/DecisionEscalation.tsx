import { useState } from "react";
import { AlertTriangle, Check, Send, ThumbsUp, AlertCircle } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import type { InboxItem } from "@/types/inbox";

interface DecisionEscalationProps {
  inboxItem: InboxItem;
  onSubmit?: (decisionId: string, rationale?: string) => void;
}

export function DecisionEscalation({ inboxItem, onSubmit }: DecisionEscalationProps) {
  const options = inboxItem.decisionOptions ?? [];
  const [selectedOptionId, setSelectedOptionId] = useState<string | null>(
    inboxItem.selectedDecisionId ?? null,
  );
  const [rationale, setRationale] = useState(inboxItem.decisionRationale ?? "");
  const [isSubmitted, setIsSubmitted] = useState(inboxItem.status === "resolved");

  const handleSubmit = () => {
    if (!selectedOptionId) return;
    setIsSubmitted(true);
    onSubmit?.(selectedOptionId, rationale || undefined);
  };

  const selectedOption = options.find((o) => o.id === selectedOptionId);

  return (
    <div className="rounded-lg border border-border bg-card/50 p-4 space-y-4">
      {/* Card header */}
      <div className="flex items-center gap-2">
        <div className="rounded-md bg-amber-500/10 p-1.5">
          <AlertTriangle className="size-4 text-amber-400" />
        </div>
        <div className="min-w-0 flex-1">
          <h3 className="text-sm font-semibold truncate">{inboxItem.title}</h3>
          <p className="text-xs text-muted-foreground truncate">{inboxItem.description}</p>
        </div>
        <Badge className="bg-amber-500/20 text-amber-400 border-amber-500/30 text-xs">
          Decision Needed
        </Badge>
      </div>

      {/* Context */}
      {inboxItem.decisionContext && (
        <div className="space-y-1.5">
          <span className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
            Context
          </span>
          <div className="rounded-md border border-border bg-muted/50 p-3">
            <p className="text-sm text-foreground leading-relaxed whitespace-pre-wrap">
              {inboxItem.decisionContext}
            </p>
          </div>
        </div>
      )}

      {/* Decision options */}
      <div className="space-y-2">
        <span className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
          Options
        </span>
        <div className="space-y-2">
          {options.map((option) => {
            const isSelected = selectedOptionId === option.id;
            return (
              <button
                key={option.id}
                type="button"
                disabled={isSubmitted}
                onClick={() => setSelectedOptionId(option.id)}
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
                  <div className="flex-1 min-w-0 space-y-1.5">
                    <div className="flex items-center gap-2">
                      <span className="font-medium text-sm">{option.label}</span>
                      {option.recommended && (
                        <Badge className="bg-emerald-500/20 text-emerald-400 border-emerald-500/30 text-[0.6rem]">
                          Recommended
                        </Badge>
                      )}
                    </div>
                    {option.description && (
                      <p className="text-xs text-muted-foreground">{option.description}</p>
                    )}
                    {/* Benefits */}
                    {option.benefits && option.benefits.length > 0 && (
                      <div className="space-y-0.5 pt-1">
                        {option.benefits.map((benefit, i) => (
                          <div key={i} className="flex items-start gap-1.5 text-xs">
                            <ThumbsUp className="size-3 text-emerald-400 mt-0.5 shrink-0" />
                            <span className="text-muted-foreground">{benefit}</span>
                          </div>
                        ))}
                      </div>
                    )}
                    {/* Risks */}
                    {option.risks && option.risks.length > 0 && (
                      <div className="space-y-0.5 pt-1">
                        {option.risks.map((risk, i) => (
                          <div key={i} className="flex items-start gap-1.5 text-xs">
                            <AlertCircle className="size-3 text-amber-400 mt-0.5 shrink-0" />
                            <span className="text-muted-foreground">{risk}</span>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              </button>
            );
          })}
        </div>
      </div>

      {/* Rationale input */}
      <div className="space-y-1.5 pt-2 border-t">
        <label className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
          Decision rationale (optional)
        </label>
        <Textarea
          placeholder="Explain why you chose this option..."
          value={rationale}
          onChange={(e) => setRationale(e.target.value)}
          disabled={isSubmitted}
          rows={2}
          className="text-xs resize-none"
        />
      </div>

      {/* Submitted state or submit button */}
      {isSubmitted && selectedOption ? (
        <div className="rounded-md border border-emerald-500/30 bg-emerald-500/10 px-3 py-2">
          <div className="flex items-center gap-1.5 text-emerald-400">
            <Check className="size-3.5" />
            <span className="text-xs font-medium">Decision made: {selectedOption.label}</span>
          </div>
          {rationale && <p className="text-xs text-muted-foreground mt-1">{rationale}</p>}
        </div>
      ) : (
        <Button onClick={handleSubmit} disabled={!selectedOptionId} className="w-full" size="sm">
          <Send className="size-3.5" />
          Confirm Decision
        </Button>
      )}
    </div>
  );
}
