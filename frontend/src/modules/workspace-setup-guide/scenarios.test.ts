import { describe, expect, test } from "vitest";
import { validateGuideJourney } from "./resolve";
import {
  CREATE_DATABASE_CHANGE_SCENARIO,
  GUIDE_SCENARIO_REGISTRY,
  getGuideJourney,
  QUERY_DATA_SCENARIO,
  WORKSPACE_SETUP_JOURNEY,
} from "./scenarios";
import { GUIDE_STEP_REGISTRY } from "./steps";

describe("workspace setup guide journeys", () => {
  test("registers only the two customer scenarios", () => {
    expect(Object.keys(GUIDE_SCENARIO_REGISTRY)).toEqual([
      "query-data",
      "create-database-change",
    ]);
  });

  test("uses an unassigned generic journey when no scenario was selected", () => {
    expect(getGuideJourney(undefined)).toBe(WORKSPACE_SETUP_JOURNEY);
    expect(WORKSPACE_SETUP_JOURNEY.scenarioId).toBeUndefined();
    expect(WORKSPACE_SETUP_JOURNEY).toMatchObject({
      id: "workspace-setup",
      steps: [
        { stepId: "create-project" },
        { stepId: "connect-instance", dependsOn: ["create-project"] },
        { stepId: "explore-database", dependsOn: ["connect-instance"] },
      ],
    });
  });

  test("uses workspace setup as visible Query Data prerequisites", () => {
    expect(QUERY_DATA_SCENARIO.steps).toEqual([
      { stepId: "create-project", kind: "prerequisite" },
      {
        stepId: "connect-instance",
        kind: "prerequisite",
        dependsOn: ["create-project"],
      },
      {
        stepId: "explore-database",
        kind: "prerequisite",
        dependsOn: ["connect-instance"],
      },
      { stepId: "query-data", dependsOn: ["explore-database"] },
    ]);
  });

  test("uses workspace setup as visible database-change prerequisites", () => {
    expect(CREATE_DATABASE_CHANGE_SCENARIO.steps).toEqual([
      { stepId: "create-project", kind: "prerequisite" },
      {
        stepId: "connect-instance",
        kind: "prerequisite",
        dependsOn: ["create-project"],
      },
      {
        stepId: "explore-database",
        kind: "prerequisite",
        dependsOn: ["connect-instance"],
      },
      {
        stepId: "create-database-change",
        dependsOn: ["explore-database"],
      },
    ]);
  });

  test.each([
    [undefined, ["create-project", "connect-instance", "explore-database"]],
    [
      "query-data" as const,
      ["create-project", "connect-instance", "explore-database", "query-data"],
    ],
    [
      "create-database-change" as const,
      [
        "create-project",
        "connect-instance",
        "explore-database",
        "create-database-change",
      ],
    ],
  ])(
    "appends the team modifier after the %s journey",
    (scenarioId, baseSteps) => {
      const journey = getGuideJourney(scenarioId, "team", false);

      expect(journey.steps.map(({ stepId }) => stepId)).toEqual([
        ...baseSteps,
        "create-user",
      ]);
      expect(journey.steps.at(-1)).toEqual({
        stepId: "create-user",
        kind: "modifier",
        dependsOn: [baseSteps.at(-1)],
      });
    }
  );

  test("uses a fixed grant-access step for SaaS team journeys", () => {
    const journey = getGuideJourney("query-data", "team", true);

    expect(journey.steps.at(-1)).toEqual({
      stepId: "grant-access",
      kind: "modifier",
      dependsOn: ["query-data"],
    });
  });

  test.each([undefined, "solo" as const])(
    "does not append the team modifier for workspace usage %s",
    (workspaceUsage) => {
      expect(
        getGuideJourney("query-data", workspaceUsage).steps.map(
          ({ stepId }) => stepId
        )
      ).toEqual([
        "create-project",
        "connect-instance",
        "explore-database",
        "query-data",
      ]);
    }
  );

  test.each([
    WORKSPACE_SETUP_JOURNEY,
    QUERY_DATA_SCENARIO,
    CREATE_DATABASE_CHANGE_SCENARIO,
  ])("validates $id", (journey) => {
    expect(() =>
      validateGuideJourney(journey, GUIDE_STEP_REGISTRY)
    ).not.toThrow();
  });
});
