import { describe, expect, test } from "vitest";
import { getGuideAnalyticsProperties } from "./analytics";
import type { GuideJourney, ResolvedGuide } from "./types";

const journey: GuideJourney = {
  id: "query-data",
  scenarioId: "query-data",
  completionTitleKey: "",
  completionDescriptionKey: "",
  completionActions: [],
  steps: [
    { stepId: "create-project" },
    { stepId: "connect-instance", dependsOn: ["create-project"] },
    { stepId: "explore-database", dependsOn: ["connect-instance"] },
    { stepId: "query-data", dependsOn: ["explore-database"] },
  ],
};

const resolvedStep = (
  index: number,
  overrides: Partial<ResolvedGuide["steps"][number]> = {}
): ResolvedGuide["steps"][number] => ({
  journeyStep: journey.steps[index],
  definition: {
    id: journey.steps[index].stepId,
  } as ResolvedGuide["steps"][number]["definition"],
  done: false,
  blocked: false,
  actions: {},
  ...overrides,
});

const guide: ResolvedGuide = {
  steps: [
    resolvedStep(0, { done: true }),
    resolvedStep(1),
    resolvedStep(2, { blocked: true }),
    resolvedStep(3, { blocked: true }),
  ],
  complete: false,
  activeStep: resolvedStep(1),
  highlightedStep: undefined,
  actionStep: undefined,
};

describe("guide analytics", () => {
  test("records a safe progress snapshot for the selected journey", () => {
    expect(
      getGuideAnalyticsProperties({
        guide,
        journey,
        scenarioId: "query-data",
        workspaceUsage: "team",
      })
    ).toEqual({
      journey: "query-data",
      scenario: "query-data",
      collaboration_type: "team",
      completed_steps: ["create-project"],
      completed_step_count: 1,
      total_step_count: 4,
      next_step: "connect-instance",
    });
  });

  test("uses explicit unselected values and omits the next step after completion", () => {
    expect(
      getGuideAnalyticsProperties({
        guide: { ...guide, complete: true, activeStep: undefined },
        journey,
      })
    ).toEqual({
      journey: "query-data",
      scenario: "unselected",
      collaboration_type: "unselected",
      completed_steps: ["create-project"],
      completed_step_count: 1,
      total_step_count: 4,
    });
  });
});
