import { validateGuideJourneys } from "./resolve";
import { GUIDE_STEP_DEFINITIONS } from "./steps";
import type {
  GuideJourney,
  GuideScenarioId,
  GuideWorkspaceUsage,
} from "./types";

export const WORKSPACE_SETUP_JOURNEY: GuideJourney = {
  id: "workspace-setup",
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
  steps: [
    ...SCENARIO_PREREQUISITES,
    { stepId: "query-data", dependsOn: ["explore-database"] },
  ],
};

export const CREATE_DATABASE_CHANGE_SCENARIO: GuideJourney = {
  id: "create-database-change",
  scenarioId: "create-database-change",
  steps: [
    ...SCENARIO_PREREQUISITES,
    {
      stepId: "create-database-change",
      dependsOn: ["explore-database"],
    },
  ],
};

export const MARK_SENSITIVE_DATA_SCENARIO: GuideJourney = {
  id: "mark-sensitive-data",
  scenarioId: "mark-sensitive-data",
  steps: [
    ...SCENARIO_PREREQUISITES,
    {
      stepId: "mark-sensitive-data",
      dependsOn: ["explore-database"],
    },
    { stepId: "query-data", dependsOn: ["mark-sensitive-data"] },
  ],
};

export const GUIDE_SCENARIO_REGISTRY: Readonly<
  Record<GuideScenarioId, GuideJourney>
> = {
  "query-data": QUERY_DATA_SCENARIO,
  "create-database-change": CREATE_DATABASE_CHANGE_SCENARIO,
  "mark-sensitive-data": MARK_SENSITIVE_DATA_SCENARIO,
};

export const isGuideScenarioId = (value: unknown): value is GuideScenarioId =>
  typeof value === "string" && Object.hasOwn(GUIDE_SCENARIO_REGISTRY, value);

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
  GUIDE_STEP_DEFINITIONS
);
