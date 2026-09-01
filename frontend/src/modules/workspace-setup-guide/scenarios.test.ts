import { describe, expect, test } from "vitest";
import { validateGuideScenario } from "./resolve";
import {
  GUIDE_SCENARIO_REGISTRY,
  LEARN_BYTEBASE_BASICS_SCENARIO,
} from "./scenarios";
import { GUIDE_STEP_REGISTRY } from "./steps";

describe("learn-bytebase-basics", () => {
  test("is the only registered scenario and has the approved graph", () => {
    expect(Object.keys(GUIDE_SCENARIO_REGISTRY)).toEqual([
      "learn-bytebase-basics",
    ]);
    expect(LEARN_BYTEBASE_BASICS_SCENARIO.steps).toEqual([
      { stepId: "create-project" },
      { stepId: "connect-instance" },
      {
        stepId: "explore-database",
        dependsOn: ["create-project", "connect-instance"],
      },
      { stepId: "query-data", dependsOn: ["explore-database"] },
    ]);
    expect(() =>
      validateGuideScenario(LEARN_BYTEBASE_BASICS_SCENARIO, GUIDE_STEP_REGISTRY)
    ).not.toThrow();
  });
});
