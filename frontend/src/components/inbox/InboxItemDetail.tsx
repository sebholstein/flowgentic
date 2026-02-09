import { ScrollArea } from "@/components/ui/scroll-area";
import type { InboxItem, TaskExecution, ViewMode } from "@/types/inbox";
import { ExecutionComparison } from "./ExecutionComparison";
import { OverseerConversation } from "./OverseerConversation";
import { QuestionnaireView } from "./QuestionnaireView";
import { DecisionEscalation } from "./DecisionEscalation";
import { DirectionClarification } from "./DirectionClarification";
import { PlanningApproval, type PlanningData } from "./PlanningApproval";
import { TaskPlanApproval } from "./TaskPlanApproval";
import { ThreadReview, type ThreadReviewData } from "./ThreadReview";
import { InboxChatLayout } from "./InboxChatLayout";

interface InboxItemDetailProps {
  inboxItem: InboxItem;
  executions: TaskExecution[];
  viewMode: ViewMode;
  planningData?: PlanningData;
  threadReviewData?: ThreadReviewData;
  onSelectExecution?: (executionId: string) => void;
  onOverride?: () => void;
  onQuestionnaireSubmit?: (
    answers: Record<string, string[]>,
    otherAnswers: Record<string, string>,
    additionalContext?: string,
  ) => void;
  onDecisionSubmit?: (decisionId: string, rationale?: string) => void;
  onClarificationSubmit?: (response: string, delegateToOverseer?: boolean) => void;
  onPlanApprove?: () => void;
  onPlanRequestChanges?: (feedback: string) => void;
  onTaskPlanApprove?: () => void;
  onTaskPlanRequestChanges?: (feedback: string) => void;
  onThreadApprove?: () => void;
  onThreadRequestChanges?: (feedback: string) => void;
  onSendMessage?: (message: string) => void;
}

export function InboxItemDetail({
  inboxItem,
  executions,
  viewMode,
  planningData,
  threadReviewData,
  onSelectExecution,
  onOverride,
  onQuestionnaireSubmit,
  onDecisionSubmit,
  onClarificationSubmit,
  onPlanApprove,
  onPlanRequestChanges,
  onTaskPlanApprove,
  onTaskPlanRequestChanges,
  onThreadApprove,
  onThreadRequestChanges,
  onSendMessage,
}: InboxItemDetailProps) {
  // Filter executions relevant to this inbox item
  const relevantExecutions = inboxItem.executionIds
    ? executions.filter((e) => inboxItem.executionIds?.includes(e.id))
    : [];

  if (viewMode === "overseer" && inboxItem.overseerMessages) {
    return (
      <OverseerConversation
        key={inboxItem.id}
        messages={inboxItem.overseerMessages}
        executions={relevantExecutions}
        selectedExecutionId={inboxItem.selectedExecutionId}
        decidedBy={inboxItem.decidedBy}
        onOverride={onOverride}
      />
    );
  }

  // User mode - show execution comparison for execution_selection type
  if (inboxItem.type === "execution_selection") {
    return (
      <ScrollArea key={inboxItem.id} className="h-full">
        <div className="p-6">
          <ExecutionComparison
            title={inboxItem.title}
            taskName={inboxItem.description}
            executions={relevantExecutions}
            selectedExecutionId={inboxItem.selectedExecutionId}
            onSelectExecution={onSelectExecution}
          />
        </div>
      </ScrollArea>
    );
  }

  // Thread review type
  if (inboxItem.type === "thread_review" && threadReviewData) {
    return (
      <InboxChatLayout key={inboxItem.id} inboxItem={inboxItem} onSendMessage={onSendMessage}>
        <ThreadReview
          inboxItem={inboxItem}
          reviewData={threadReviewData}
          onApprove={onThreadApprove}
          onRequestChanges={onThreadRequestChanges}
        />
      </InboxChatLayout>
    );
  }

  // Planning approval type
  if (inboxItem.type === "planning_approval" && planningData) {
    return (
      <InboxChatLayout key={inboxItem.id} inboxItem={inboxItem} onSendMessage={onSendMessage}>
        <PlanningApproval
          inboxItem={inboxItem}
          planningData={planningData}
          onApprove={onPlanApprove}
          onRequestChanges={onPlanRequestChanges}
        />
      </InboxChatLayout>
    );
  }

  // Task plan approval type
  if (inboxItem.type === "task_plan_approval") {
    return (
      <InboxChatLayout key={inboxItem.id} inboxItem={inboxItem} onSendMessage={onSendMessage}>
        <TaskPlanApproval
          inboxItem={inboxItem}
          onApprove={onTaskPlanApprove}
          onRequestChanges={onTaskPlanRequestChanges}
        />
      </InboxChatLayout>
    );
  }

  // Questionnaire type
  if (inboxItem.type === "questionnaire") {
    return (
      <InboxChatLayout key={inboxItem.id} inboxItem={inboxItem} onSendMessage={onSendMessage}>
        <QuestionnaireView inboxItem={inboxItem} onSubmit={onQuestionnaireSubmit} />
      </InboxChatLayout>
    );
  }

  // Decision escalation type
  if (inboxItem.type === "decision_escalation") {
    return (
      <InboxChatLayout key={inboxItem.id} inboxItem={inboxItem} onSendMessage={onSendMessage}>
        <DecisionEscalation inboxItem={inboxItem} onSubmit={onDecisionSubmit} />
      </InboxChatLayout>
    );
  }

  // Direction clarification type
  if (inboxItem.type === "direction_clarification") {
    return (
      <InboxChatLayout key={inboxItem.id} inboxItem={inboxItem} onSendMessage={onSendMessage}>
        <DirectionClarification inboxItem={inboxItem} onSubmit={onClarificationSubmit} />
      </InboxChatLayout>
    );
  }

  return null;
}
