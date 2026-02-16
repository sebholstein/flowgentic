import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Bot, Loader2, X, Bell } from "lucide-react";
import type { ChatTarget } from "./chat-types";
import type { InboxItem } from "@/types/inbox";

export function ChatHeader({
  target,
  pendingFeedback,
  onClose,
}: {
  target: ChatTarget;
  pendingFeedback?: InboxItem | null;
  onClose?: () => void;
}) {
  return (
    <div className="flex items-center gap-2 border-b px-4 py-2">
      {target.currentStep ? (
        <>
          <Loader2 className="size-3.5 animate-spin text-emerald-500 shrink-0" />
          <span className="text-xs text-muted-foreground">
            Step {target.currentStep.current}/{target.currentStep.total}
          </span>
          <span className="text-xs font-medium truncate flex-1">{target.currentStep.name}</span>
        </>
      ) : (
        <>
          <Bot className="size-4 text-muted-foreground shrink-0" />
          <span className="text-xs font-medium flex-1">{target.agentName}</span>
        </>
      )}
      {pendingFeedback && (
        <Badge variant="outline" className="text-amber-400 border-amber-500/30 text-xs gap-1">
          <Bell className="size-3" />
          Feedback
        </Badge>
      )}
      {onClose && (
        <Button
          variant="ghost"
          size="sm"
          onClick={onClose}
          className="size-7 p-0 text-muted-foreground hover:text-foreground"
        >
          <X className="size-4" />
        </Button>
      )}
    </div>
  );
}
