import { useState } from "react";
import { useParams, useSearch } from "@tanstack/react-router";
import { MessageSquare } from "lucide-react";

import { InboxItemDetail } from "@/components/inbox/InboxItemDetail";
import {
  inboxItems,
  executionsData,
  planningDataByItemId,
  threadReviewDataByItemId,
} from "@/data/mockInboxData";
import type { ViewMode } from "@/types/inbox";

export function InboxItemDetailPage() {
  const { itemId } = useParams({ from: "/app/inbox/$itemId/" });
  const { mode: viewMode = "user" } = useSearch({ from: "/app/inbox/$itemId/" }) as {
    mode?: ViewMode;
  };
  const [localSelectedExecution, setLocalSelectedExecution] = useState<string | undefined>();

  const inboxItem = inboxItems.find((item) => item.id === itemId);

  if (!inboxItem) {
    return (
      <div className="flex h-full items-center justify-center text-muted-foreground">
        <div className="text-center">
          <MessageSquare className="mx-auto size-12 opacity-50" />
          <p className="mt-2">Inbox item not found</p>
        </div>
      </div>
    );
  }

  const selectedExecutionId = localSelectedExecution ?? inboxItem.selectedExecutionId;

  const handleSelectExecution = (executionId: string) => {
    setLocalSelectedExecution(executionId);
  };

  const handleOverride = () => {
    setLocalSelectedExecution(undefined);
  };

  const handleQuestionnaireSubmit = (
    answers: Record<string, string[]>,
    otherAnswers: Record<string, string>,
    additionalContext?: string,
  ) => {
    console.log("Questionnaire submitted:", { answers, otherAnswers, additionalContext });
  };

  const handleDecisionSubmit = (decisionId: string, rationale?: string) => {
    console.log("Decision submitted:", { decisionId, rationale });
  };

  const handleClarificationSubmit = (response: string, delegateToOverseer?: boolean) => {
    console.log("Clarification submitted:", { response, delegateToOverseer });
  };

  const handlePlanApprove = () => {
    console.log("Plan approved for item:", itemId);
  };

  const handlePlanRequestChanges = (feedback: string) => {
    console.log("Plan changes requested:", { itemId, feedback });
  };

  const handleThreadApprove = () => {
    console.log("Thread approved for item:", itemId);
  };

  const handleThreadRequestChanges = (feedback: string) => {
    console.log("Thread changes requested:", { itemId, feedback });
  };

  const handleSendMessage = (message: string) => {
    console.log("Message sent:", { itemId, message });
  };

  const planningData = planningDataByItemId[itemId];
  const threadReviewData = threadReviewDataByItemId[itemId];

  return (
    <InboxItemDetail
      inboxItem={{
        ...inboxItem,
        selectedExecutionId,
      }}
      executions={executionsData}
      viewMode={viewMode}
      planningData={planningData}
      threadReviewData={threadReviewData}
      onSelectExecution={handleSelectExecution}
      onOverride={handleOverride}
      onQuestionnaireSubmit={handleQuestionnaireSubmit}
      onDecisionSubmit={handleDecisionSubmit}
      onClarificationSubmit={handleClarificationSubmit}
      onPlanApprove={handlePlanApprove}
      onPlanRequestChanges={handlePlanRequestChanges}
      onThreadApprove={handleThreadApprove}
      onThreadRequestChanges={handleThreadRequestChanges}
      onSendMessage={handleSendMessage}
    />
  );
}
