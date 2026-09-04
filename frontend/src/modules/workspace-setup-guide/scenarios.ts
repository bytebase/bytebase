import { validateGuideJourneys } from "./resolve";
import { GUIDE_STEP_REGISTRY } from "./steps";
import type {
  GuideJourney,
  GuideScenarioId,
  GuideWorkspaceUsage,
} from "./types";

export const WORKSPACE_SETUP_JOURNEY: GuideJourney = {
  id: "workspace-setup",
  completionTitleKey: "workspace-setup-guide.generic.completion-title",
  completionDescriptionKey:
    "workspace-setup-guide.generic.completion-description",
  completionActions: ["open-sql-editor", "create-change"],
  steps: [
    { stepId: "create-project" },
    { stepId: "connect-instance", dependsOn: ["create-project"] },
    { stepId: "explore-database", dependsOn: ["connect-instance"] },
  ],
};

const SCENARIO_PREREQUISITES = WORKSPACE_SETUP_JOURNEY.steps.map((step) => ({
  ...step,
  kind: "prerequisite" as const,
}));

export const QUERY_DATA_SCENARIO: GuideJourney = {
  id: "query-data",
  scenarioId: "query-data",
  completionTitleKey:
    "workspace-setup-guide.scenarios.query-data.completion-title",
  completionDescriptionKey:
    "workspace-setup-guide.scenarios.query-data.completion-description",
  completionActions: ["create-change"],
  steps: [
    ...SCENARIO_PREREQUISITES,
    { stepId: "query-data", dependsOn: ["explore-database"] },
  ],
};

export const CREATE_DATABASE_CHANGE_SCENARIO: GuideJourney = {
  id: "create-database-change",
  scenarioId: "create-database-change",
  completionTitleKey:
    "workspace-setup-guide.scenarios.create-database-change.completion-title",
  completionDescriptionKey:
    "workspace-setup-guide.scenarios.create-database-change.completion-description",
  completionActions: ["open-sql-editor"],
  steps: [
    ...SCENARIO_PREREQUISITES,
    {
      stepId: "create-database-change",
      dependsOn: ["explore-database"],
    },
  ],
};

export const GUIDE_SCENARIO_REGISTRY: Readonly<
  Record<GuideScenarioId, GuideJourney>
> = {
  "query-data": QUERY_DATA_SCENARIO,
  "create-database-change": CREATE_DATABASE_CHANGE_SCENARIO,
};

export const getGuideJourney = (
  scenarioId: GuideScenarioId | undefined,
  workspaceUsage?: GuideWorkspaceUsage
): GuideJourney => {
  const journey = scenarioId
    ? GUIDE_SCENARIO_REGISTRY[scenarioId]
    : WORKSPACE_SETUP_JOURNEY;
  if (workspaceUsage !== "team") return journey;

  const previousStep = journey.steps.at(-1);
  if (!previousStep) return journey;
  return {
    ...journey,
    steps: [
      ...journey.steps,
      {
        stepId: "add-member",
        kind: "modifier",
        dependsOn: [previousStep.stepId],
      },
    ],
  };
};

validateGuideJourneys(
  [WORKSPACE_SETUP_JOURNEY, ...Object.values(GUIDE_SCENARIO_REGISTRY)],
  GUIDE_STEP_REGISTRY
);
