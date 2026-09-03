import { describe, expect, test } from "vitest";
import {
  resolveGuide,
  validateGuideJourney,
  validateGuideJourneys,
} from "./resolve";
import type {
  GuideContext,
  GuideJourney,
  GuideStepDefinition,
  GuideStepId,
  GuideStepRegistry,
} from "./types";

const STEP_IDS: GuideStepId[] = [
  "create-project",
  "connect-instance",
  "explore-database",
  "query-data",
  "create-database-change",
  "create-user",
  "grant-access",
];

const createContext = (
  overrides: Partial<GuideContext> = {}
): GuideContext => ({
  hasProject: false,
  hasInstance: false,
  hasExploredDatabase: false,
  hasRunStatement: false,
  hasCreatedChangeIssue: false,
  isSaaS: false,
  hasOtherHumanUser: false,
  hasOtherWorkspaceMember: false,
  projectName: "",
  instanceName: "",
  databaseProjectName: "",
  databaseName: "",
  route: { name: "workspace.home", params: {} },
  ...overrides,
});

const completionById: Record<GuideStepId, keyof GuideContext> = {
  "create-project": "hasProject",
  "connect-instance": "hasInstance",
  "explore-database": "hasExploredDatabase",
  "query-data": "hasRunStatement",
  "create-database-change": "hasCreatedChangeIssue",
  "create-user": "hasOtherWorkspaceMember",
  "grant-access": "hasOtherWorkspaceMember",
};

const definition = (id: GuideStepId): GuideStepDefinition => ({
  id,
  analyticsKey:
    id === "create-user" || id === "grant-access" ? "add-teammate" : id,
  labelKey: `label.${id}`,
  descriptionKey: `description.${id}`,
  isComplete: (context) => Boolean(context[completionById[id]]),
  matchesRoute: (route) => route.name === `route.${id}`,
  resolveActions: () => ({}),
});

const registry = Object.fromEntries(
  STEP_IDS.map((id) => [id, definition(id)])
) as GuideStepRegistry;

const generic: GuideJourney = {
  id: "workspace-setup",
  completionTitleKey: "completion.generic.title",
  completionDescriptionKey: "completion.generic.description",
  completionActions: ["open-sql-editor", "create-change"],
  steps: [
    { stepId: "create-project" },
    { stepId: "connect-instance", dependsOn: ["create-project"] },
    { stepId: "explore-database", dependsOn: ["connect-instance"] },
  ],
};

const queryData: GuideJourney = {
  id: "query-data",
  scenarioId: "query-data",
  completionTitleKey: "completion.query.title",
  completionDescriptionKey: "completion.query.description",
  completionActions: ["create-change"],
  steps: [
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
  ],
};

describe("validateGuideJourney", () => {
  test("accepts selected and generic graphs", () => {
    expect(() => validateGuideJourney(generic, registry)).not.toThrow();
    expect(() => validateGuideJourney(queryData, registry)).not.toThrow();
  });

  test.each([
    ["empty", { ...generic, steps: [] }, "at least one step"],
    [
      "duplicate",
      {
        ...generic,
        steps: [{ stepId: "create-project" }, { stepId: "create-project" }],
      },
      "duplicate step",
    ],
    [
      "unknown",
      { ...generic, steps: [{ stepId: "missing" as GuideStepId }] },
      "unregistered step",
    ],
    [
      "external dependency",
      {
        ...generic,
        steps: [{ stepId: "create-project", dependsOn: ["connect-instance"] }],
      },
      "dependency outside journey",
    ],
    [
      "self dependency",
      {
        ...generic,
        steps: [{ stepId: "create-project", dependsOn: ["create-project"] }],
      },
      "cannot depend on itself",
    ],
  ])("rejects an %s graph", (_name, journey, message) => {
    expect(() =>
      validateGuideJourney(journey as GuideJourney, registry)
    ).toThrow(message);
  });

  test("rejects duplicate journey ids", () => {
    expect(() => validateGuideJourneys([generic, generic], registry)).toThrow(
      "duplicate journey"
    );
  });
});

describe("resolveGuide", () => {
  test("uses generic setup order and dependencies", () => {
    const guide = resolveGuide({
      journey: generic,
      registry,
      context: createContext(),
    });
    expect(guide.steps.map((step) => step.definition.id)).toEqual([
      "create-project",
      "connect-instance",
      "explore-database",
    ]);
    expect(guide.activeStep?.definition.id).toBe("create-project");
  });

  test("keeps satisfied prerequisites visible", () => {
    const guide = resolveGuide({
      journey: queryData,
      registry,
      context: createContext({
        hasProject: true,
        hasInstance: true,
        hasExploredDatabase: true,
      }),
    });
    expect(guide.steps.map((step) => step.definition.id)).toEqual([
      "create-project",
      "connect-instance",
      "explore-database",
      "query-data",
    ]);
    expect(guide.steps.slice(0, 3).every((step) => step.done)).toBe(true);
    expect(guide.activeStep?.definition.id).toBe("query-data");
  });

  test("shows the full chain and blocks on missing prerequisites", () => {
    const guide = resolveGuide({
      journey: queryData,
      registry,
      context: createContext(),
    });
    expect(guide.steps.map((step) => step.definition.id)).toEqual([
      "create-project",
      "connect-instance",
      "explore-database",
      "query-data",
    ]);
    expect(guide.steps[1].blocked).toBe(true);
    expect(guide.steps[3].blocked).toBe(true);
  });

  test("allows a completed visible prerequisite to stay selected", () => {
    const guide = resolveGuide({
      journey: queryData,
      registry,
      context: createContext({ hasProject: true }),
      selectedStepId: "create-project",
    });
    expect(guide.highlightedStep?.definition.id).toBe("create-project");
    expect(guide.actionStep?.definition.id).toBe("connect-instance");
  });

  test("returns completion without an active action", () => {
    const guide = resolveGuide({
      journey: queryData,
      registry,
      context: createContext({
        hasProject: true,
        hasInstance: true,
        hasExploredDatabase: true,
        hasRunStatement: true,
      }),
    });
    expect(guide.complete).toBe(true);
    expect(guide.activeStep).toBeUndefined();
    expect(guide.actionStep).toBeUndefined();
  });
});
