// @vitest-environment node
import { describe, expect, test } from "vitest";
import { resolveGuide, validateGuideJourney } from "./resolve";
import {
  CREATE_DATABASE_CHANGE_SCENARIO,
  GUIDE_SCENARIO_REGISTRY,
  getGuideJourney,
  MARK_SENSITIVE_DATA_SCENARIO,
  QUERY_DATA_SCENARIO,
  WORKSPACE_SETUP_JOURNEY,
} from "./scenarios";
import { GUIDE_STEP_DEFINITIONS } from "./steps";

describe("workspace setup guide journeys", () => {
  test("registers the customer scenarios", () => {
    expect(Object.keys(GUIDE_SCENARIO_REGISTRY)).toEqual([
      "query-data",
      "create-database-change",
      "mark-sensitive-data",
    ]);
  });

  test("uses workspace setup as visible sensitive-data prerequisites", () => {
    const journey = getGuideJourney("mark-sensitive-data");

    expect(journey.steps).toEqual([
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
        stepId: "mark-sensitive-data",
        dependsOn: ["explore-database"],
      },
      { stepId: "query-data", dependsOn: ["mark-sensitive-data"] },
    ]);
  });

  test.each([
    [false, false, true, false],
    [true, false, false, false],
    [true, true, false, true],
  ])(
    "resolves masking verification with marked=%s and queried=%s",
    (hasMarkedSensitiveData, hasRunStatement, blocked, complete) => {
      const databaseName = "projects/test/instances/test/databases/test";
      const guide = resolveGuide({
        journey: MARK_SENSITIVE_DATA_SCENARIO,
        definitions: GUIDE_STEP_DEFINITIONS,
        context: {
          hasProject: true,
          hasInstance: true,
          hasExploredDatabase: true,
          hasMarkedSensitiveData,
          hasRunStatement,
          hasCreatedChangeIssue: false,
          isSaaS: true,
          hasOtherHumanUser: false,
          hasOtherWorkspaceMember: false,
          projectName: "projects/test",
          instanceName: "projects/test/instances/test",
          databaseProjectName: "projects/test",
          databaseName,
          route: { name: "workspace.home", params: {} },
        },
      });

      expect(guide.steps.at(-1)).toMatchObject({
        definition: { id: "query-data" },
        blocked,
        actions: {
          primary: {
            type: "open-sql-editor",
            database: { name: databaseName, project: "projects/test" },
          },
        },
      });
      expect(guide.complete).toBe(complete);
    }
  );

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
    [
      "mark-sensitive-data" as const,
      [
        "create-project",
        "connect-instance",
        "explore-database",
        "mark-sensitive-data",
        "query-data",
      ],
    ],
  ])(
    "appends the team modifier after the %s journey",
    (scenarioId, baseSteps) => {
      const journey = getGuideJourney(scenarioId, "team");

      expect(journey.steps.map(({ stepId }) => stepId)).toEqual([
        ...baseSteps,
        "add-member",
      ]);
      expect(journey.steps.at(-1)).toEqual({
        stepId: "add-member",
        kind: "modifier",
        dependsOn: [baseSteps.at(-1)],
      });
    }
  );

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
    MARK_SENSITIVE_DATA_SCENARIO,
  ])("validates $id", (journey) => {
    expect(() =>
      validateGuideJourney(journey, GUIDE_STEP_DEFINITIONS)
    ).not.toThrow();
  });
});
