import { createFileRoute } from "@tanstack/react-router";
import { getDemoThread, getDemoPlan } from "@/data/mockFlowgenticData";
import { DemoHeader } from "@/components/demo/DemoHeader";
import { QuickModeView } from "@/components/demo/QuickModeView";
import { PlanSinglePlanningView } from "@/components/demo/PlanSinglePlanningView";
import { PlanMultiPlanningView } from "@/components/demo/PlanMultiPlanningView";
import { PlanExecutionView } from "@/components/demo/PlanExecutionView";
import { NewThreadView } from "@/components/demo/NewThreadView";

export const Route = createFileRoute("/app/demo/$scenarioId")({
  component: DemoScenarioPage,
});

function DemoScenarioPage() {
  const { scenarioId } = Route.useParams();
  const thread = getDemoThread(scenarioId);

  if (!thread) {
    return (
      <div className="flex h-full items-center justify-center text-muted-foreground">
        Demo scenario not found
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col bg-surface rounded-lg overflow-hidden">
      <DemoHeader thread={thread} />
      <ScenarioView thread={thread} />
    </div>
  );
}

function ScenarioView({ thread }: { thread: NonNullable<ReturnType<typeof getDemoThread>> }) {
  // Scenario 0: Thread creation
  if (thread.phase === "creation") {
    return <NewThreadView />;
  }

  // Scenario 1: Quick mode, completed
  if (thread.mode === "quick") {
    return <QuickModeView thread={thread} />;
  }

  // Scenario 4: Plan mode, execution phase
  if (thread.phase === "execution") {
    return <PlanExecutionView thread={thread} />;
  }

  // Scenario 3: Plan mode, multiple planners
  if (thread.agents.length > 1) {
    return <PlanMultiPlanningView thread={thread} />;
  }

  // Scenario 2: Plan mode, single planner
  const plan = getDemoPlan(thread.id);
  if (!plan) {
    return (
      <div className="flex h-full items-center justify-center text-muted-foreground">
        Plan data not found
      </div>
    );
  }

  return <PlanSinglePlanningView thread={thread} plan={plan} />;
}
