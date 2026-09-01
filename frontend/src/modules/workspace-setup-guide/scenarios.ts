import { validateGuideScenarios } from "./resolve";
import { GUIDE_STEP_REGISTRY } from "./steps";
import type { GuideScenario, GuideScenarioId } from "./types";

export const LEARN_BYTEBASE_BASICS_SCENARIO: GuideScenario = {
  id: "learn-bytebase-basics",
  steps: [
    { stepId: "create-project" },
    { stepId: "connect-instance" },
    {
      stepId: "explore-database",
      dependsOn: ["create-project", "connect-instance"],
    },
    { stepId: "query-data", dependsOn: ["explore-database"] },
  ],
};

export const GUIDE_SCENARIO_REGISTRY: Readonly<
  Record<GuideScenarioId, GuideScenario>
> = {
  "learn-bytebase-basics": LEARN_BYTEBASE_BASICS_SCENARIO,
};

validateGuideScenarios(
  Object.values(GUIDE_SCENARIO_REGISTRY),
  GUIDE_STEP_REGISTRY
);
