import { describe, expect, test } from "vitest";
import {
  resolveGuide,
  validateGuideScenario,
  validateGuideScenarios,
} from "./resolve";
import type {
  GuideContext,
  GuideScenario,
  GuideStepDefinition,
  GuideStepId,
  GuideStepRegistry,
} from "./types";

const context = (
  completed: Partial<Record<GuideStepId, boolean>> = {}
): GuideContext => ({
  hasProject: completed["create-project"] ?? false,
  hasInstance: completed["connect-instance"] ?? false,
  hasExploredDatabase: completed["explore-database"] ?? false,
  hasFirstQuery: completed["query-data"] ?? false,
  projectName: "",
  databaseProjectName: "",
  databaseName: "",
  route: { name: "workspace.home", params: {} },
});

const definition = (
  id: GuideStepId,
  isComplete: GuideStepDefinition["isComplete"]
): GuideStepDefinition => ({
  id,
  analyticsKey: (
    {
      "create-project": "hasProject",
      "connect-instance": "hasInstance",
      "explore-database": "hasExploredDatabase",
      "query-data": "hasFirstQuery",
    } as const
  )[id],
  labelKey: `label.${id}`,
  descriptionKey: `description.${id}`,
  isComplete,
  matchesRoute: (route) => route.name === `route.${id}`,
  resolveActions: () => ({}),
});

const registry: GuideStepRegistry = {
  "create-project": definition("create-project", (value) => value.hasProject),
  "connect-instance": definition(
    "connect-instance",
    (value) => value.hasInstance
  ),
  "explore-database": definition(
    "explore-database",
    (value) => value.hasExploredDatabase
  ),
  "query-data": definition("query-data", (value) => value.hasFirstQuery),
};

const scenario: GuideScenario = {
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

describe("validateGuideScenario", () => {
  test("accepts the basics dependency graph", () => {
    expect(() => validateGuideScenario(scenario, registry)).not.toThrow();
  });

  test.each([
    [
      "empty scenario",
      { ...scenario, steps: [] },
      "must contain at least one step",
    ],
    [
      "duplicate step",
      {
        ...scenario,
        steps: [{ stepId: "create-project" }, { stepId: "create-project" }],
      },
      "duplicate step",
    ],
    [
      "unknown step",
      { ...scenario, steps: [{ stepId: "missing" as GuideStepId }] },
      "unregistered step",
    ],
    [
      "dependency outside scenario",
      {
        ...scenario,
        steps: [{ stepId: "query-data", dependsOn: ["explore-database"] }],
      },
      "dependency outside scenario",
    ],
    [
      "self dependency",
      {
        ...scenario,
        steps: [{ stepId: "create-project", dependsOn: ["create-project"] }],
      },
      "cannot depend on itself",
    ],
    [
      "forward dependency",
      {
        ...scenario,
        steps: [
          { stepId: "query-data", dependsOn: ["explore-database"] },
          { stepId: "explore-database" },
        ],
      },
      "must appear before",
    ],
  ])("rejects %s", (_name, invalid, message) => {
    expect(() =>
      validateGuideScenario(invalid as GuideScenario, registry)
    ).toThrow(message);
  });

  test("rejects cycles before reporting display order", () => {
    const invalid: GuideScenario = {
      id: "learn-bytebase-basics",
      steps: [
        { stepId: "create-project", dependsOn: ["connect-instance"] },
        { stepId: "connect-instance", dependsOn: ["create-project"] },
      ],
    };
    expect(() => validateGuideScenario(invalid, registry)).toThrow("cycle");
  });

  test("rejects duplicate scenario IDs", () => {
    expect(() =>
      validateGuideScenarios([scenario, scenario], registry)
    ).toThrow("duplicate scenario");
  });
});

describe("resolveGuide", () => {
  test("uses scenario order and dependencies to find the active step", () => {
    const resolved = resolveGuide({ scenario, registry, context: context() });
    expect(resolved.steps.map(({ definition }) => definition.id)).toEqual([
      "create-project",
      "connect-instance",
      "explore-database",
      "query-data",
    ]);
    expect(resolved.steps.map(({ blocked }) => blocked)).toEqual([
      false,
      false,
      true,
      true,
    ]);
    expect(resolved.activeStep.definition.id).toBe("create-project");
  });

  test("allows the second independent root step to become active", () => {
    const resolved = resolveGuide({
      scenario,
      registry,
      context: context({ "create-project": true }),
    });
    expect(resolved.activeStep.definition.id).toBe("connect-instance");
  });

  test("uses a valid manual selection for highlight and action", () => {
    const resolved = resolveGuide({
      scenario,
      registry,
      context: context(),
      selectedStepId: "connect-instance",
    });
    expect(resolved.highlightedStep?.definition.id).toBe("connect-instance");
    expect(resolved.actionStep.definition.id).toBe("connect-instance");
  });

  test("falls from a completed route match to the active step", () => {
    const value = context({
      "create-project": true,
      "connect-instance": true,
      "explore-database": true,
    });
    value.route = { name: "route.explore-database", params: {} };
    const resolved = resolveGuide({ scenario, registry, context: value });
    expect(resolved.highlightedStep?.definition.id).toBe("query-data");
  });

  test("does not highlight a step on unrelated routes", () => {
    const resolved = resolveGuide({ scenario, registry, context: context() });
    expect(resolved.highlightedStep).toBeUndefined();
  });

  test("ignores a selected ID that is not part of the scenario", () => {
    const resolved = resolveGuide({
      scenario,
      registry,
      context: context(),
      selectedStepId: "missing" as GuideStepId,
    });
    expect(resolved.highlightedStep).toBeUndefined();
    expect(resolved.actionStep.definition.id).toBe("create-project");
  });

  test("retains the final step after all steps complete", () => {
    const resolved = resolveGuide({
      scenario,
      registry,
      context: context({
        "create-project": true,
        "connect-instance": true,
        "explore-database": true,
        "query-data": true,
      }),
    });
    expect(resolved.activeStep.definition.id).toBe("query-data");
    expect(resolved.actionStep.definition.id).toBe("query-data");
  });
});
