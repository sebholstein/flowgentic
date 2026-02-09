import { useState } from "react";
import { HelpCircle, Check, Send } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import type { QuestionnaireQuestion, InboxItem } from "@/types/inbox";

interface QuestionnaireViewProps {
  inboxItem: InboxItem;
  onSubmit?: (
    answers: Record<string, string[]>,
    otherAnswers: Record<string, string>,
    additionalContext?: string,
  ) => void;
}

interface QuestionCardProps {
  question: QuestionnaireQuestion;
  selectedIds: string[];
  otherText: string;
  onSelect: (optionId: string) => void;
  onOtherChange: (text: string) => void;
  isSubmitted: boolean;
}

function QuestionCard({
  question,
  selectedIds,
  otherText,
  onSelect,
  onOtherChange,
  isSubmitted,
}: QuestionCardProps) {
  const isOtherSelected = selectedIds.includes("__other__");

  const hasSelection = selectedIds.length > 0;

  return (
    <div className="space-y-2">
      <div className="flex items-baseline gap-2">
        <span className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
          {question.header}
        </span>
        {question.multiSelect && (
          <span className="text-[0.6rem] text-muted-foreground/70">multiple</span>
        )}
        {!hasSelection && !isSubmitted && (
          <span className="text-[0.6rem] text-muted-foreground/50 italic">optional</span>
        )}
      </div>
      <p className="text-sm font-medium text-foreground">{question.question}</p>
      <div className="space-y-1">
        {question.options.map((option) => {
          const isSelected = selectedIds.includes(option.id);
          return (
            <button
              key={option.id}
              type="button"
              disabled={isSubmitted}
              onClick={() => onSelect(option.id)}
              className={cn(
                "flex w-full items-center gap-2 rounded px-2 py-1 text-left text-xs transition-all",
                isSelected
                  ? "bg-primary/10 text-primary"
                  : "text-muted-foreground hover:bg-muted/50 hover:text-foreground",
                isSubmitted && "cursor-not-allowed opacity-60",
              )}
            >
              <div
                className={cn(
                  "flex size-3.5 shrink-0 items-center justify-center rounded-full border",
                  question.multiSelect && "rounded-sm",
                  isSelected
                    ? "border-primary bg-primary text-primary-foreground"
                    : "border-muted-foreground/40",
                )}
              >
                {isSelected && <Check className="size-2" />}
              </div>
              <span>{option.label}</span>
              {option.description && (
                <span className="text-[0.6rem] text-muted-foreground/60">
                  — {option.description}
                </span>
              )}
            </button>
          );
        })}
        <button
          type="button"
          disabled={isSubmitted}
          onClick={() => onSelect("__other__")}
          className={cn(
            "flex w-full items-center gap-2 rounded px-2 py-1 text-left text-xs transition-all",
            isOtherSelected
              ? "bg-primary/10 text-primary"
              : "text-muted-foreground hover:bg-muted/50 hover:text-foreground",
            isSubmitted && "cursor-not-allowed opacity-60",
          )}
        >
          <div
            className={cn(
              "flex size-3.5 shrink-0 items-center justify-center rounded-full border border-dashed",
              question.multiSelect && "rounded-sm",
              isOtherSelected
                ? "border-primary bg-primary text-primary-foreground border-solid"
                : "border-muted-foreground/40",
            )}
          >
            {isOtherSelected && <Check className="size-2" />}
          </div>
          <span>Other...</span>
        </button>
      </div>
      {isOtherSelected && (
        <Input
          placeholder="Specify..."
          value={otherText}
          onChange={(e) => onOtherChange(e.target.value)}
          disabled={isSubmitted}
          className="h-7 text-xs"
        />
      )}
    </div>
  );
}

export function QuestionnaireView({ inboxItem, onSubmit }: QuestionnaireViewProps) {
  const questions = inboxItem.questions ?? [];

  const [answers, setAnswers] = useState<Record<string, string[]>>(() => {
    const initial: Record<string, string[]> = {};
    for (const q of questions) {
      initial[q.id] = q.selectedOptionIds ?? [];
    }
    return initial;
  });
  const [otherAnswers, setOtherAnswers] = useState<Record<string, string>>({});
  const [additionalContext, setAdditionalContext] = useState(inboxItem.customResponse ?? "");
  const [isSubmitted, setIsSubmitted] = useState(inboxItem.status === "resolved");

  const handleSelect = (questionId: string, optionId: string, multiSelect: boolean) => {
    if (isSubmitted) return;

    setAnswers((prev) => {
      const current = prev[questionId] ?? [];
      if (multiSelect) {
        if (current.includes(optionId)) {
          return { ...prev, [questionId]: current.filter((id) => id !== optionId) };
        }
        return { ...prev, [questionId]: [...current, optionId] };
      }
      // Single select: if clicking "other", keep it; otherwise replace
      if (optionId === "__other__" && current.includes("__other__")) {
        return { ...prev, [questionId]: [] };
      }
      return { ...prev, [questionId]: [optionId] };
    });
  };

  const handleOtherChange = (questionId: string, text: string) => {
    setOtherAnswers((prev) => ({ ...prev, [questionId]: text }));
  };

  const handleSubmit = () => {
    setIsSubmitted(true);
    onSubmit?.(answers, otherAnswers, additionalContext || undefined);
  };

  const hasSelections = Object.values(answers).some((a) => a.length > 0);
  const hasContext = additionalContext.trim().length > 0;
  const answeredCount = Object.values(answers).filter((a) => a.length > 0).length;
  const skippedCount = questions.length - answeredCount;

  return (
    <div className="rounded-lg border border-border bg-card/50 p-4 space-y-4">
      {/* Card header */}
      <div className="flex items-center gap-2">
        <div className="rounded-md bg-violet-500/10 p-1.5">
          <HelpCircle className="size-4 text-violet-400" />
        </div>
        <div className="min-w-0 flex-1">
          <h3 className="text-sm font-semibold truncate">{inboxItem.title}</h3>
          <p className="text-xs text-muted-foreground truncate">{inboxItem.description}</p>
        </div>
        <Badge className="bg-violet-500/20 text-violet-400 border-violet-500/30 text-xs">
          Questionnaire
        </Badge>
      </div>

      {/* Questions */}
      <div className="space-y-4">
        {questions.map((question) => (
          <QuestionCard
            key={question.id}
            question={question}
            selectedIds={answers[question.id] ?? []}
            otherText={otherAnswers[question.id] ?? ""}
            onSelect={(optionId) => handleSelect(question.id, optionId, question.multiSelect)}
            onOtherChange={(text) => handleOtherChange(question.id, text)}
            isSubmitted={isSubmitted}
          />
        ))}
      </div>

      {/* Additional context */}
      <div className="space-y-1.5 pt-2 border-t">
        <label className="text-[0.65rem] font-medium text-muted-foreground uppercase tracking-wide">
          Additional context
        </label>
        <Textarea
          placeholder="Provide any additional context or clarification..."
          value={additionalContext}
          onChange={(e) => setAdditionalContext(e.target.value)}
          disabled={isSubmitted}
          rows={2}
          className="text-xs resize-none"
        />
      </div>

      {/* Submit button or submitted state */}
      {isSubmitted ? (
        <div className="rounded-md border border-emerald-500/30 bg-emerald-500/10 px-3 py-2">
          <div className="flex items-center gap-1.5 text-emerald-400">
            <Check className="size-3.5" />
            <span className="text-xs font-medium">Response submitted</span>
          </div>
        </div>
      ) : (
        <div className="space-y-2">
          {skippedCount > 0 && (
            <p className="text-[0.65rem] text-center text-muted-foreground">
              {skippedCount === questions.length
                ? "All questions skipped"
                : `${skippedCount} question${skippedCount > 1 ? "s" : ""} skipped`}
            </p>
          )}
          <Button onClick={handleSubmit} className="w-full" size="sm">
            <Send className="size-3.5" />
            {hasSelections || hasContext ? "Submit" : "Skip all"}
          </Button>
        </div>
      )}
    </div>
  );
}
