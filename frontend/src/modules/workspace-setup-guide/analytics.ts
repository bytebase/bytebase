import type {
  GuideJourney,
  GuideScenarioId,
  GuideWorkspaceUsage,
  ResolvedGuide,
} from "./types";

export const getGuideAnalyticsProperties = ({
  guide,
  journey,
  scenarioId,
  workspaceUsage,
}: {
  guide: ResolvedGuide;
  journey: GuideJourney;
  scenarioId?: GuideScenarioId;
  workspaceUsage?: GuideWorkspaceUsage;
}) => {
  const completedSteps = guide.steps
    .filter((step) => step.done)
    .map((step) => step.definition.id);

  return {
    journey: journey.id,
    scenario: scenarioId ?? "unselected",
    collaboration_type: workspaceUsage ?? "unselected",
    completed_steps: completedSteps,
    completed_step_count: completedSteps.length,
    total_step_count: guide.steps.length,
    ...(guide.complete || !guide.activeStep
      ? {}
      : { next_step: guide.activeStep.definition.id }),
  };
};
