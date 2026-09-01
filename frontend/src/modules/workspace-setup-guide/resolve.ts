import type {
  GuideContext,
  GuideScenario,
  GuideStepId,
  GuideStepRegistry,
  ResolvedGuide,
} from "./types";

export const validateGuideScenario = (
  scenario: GuideScenario,
  registry: GuideStepRegistry
) => {
  if (scenario.steps.length === 0) {
    throw new Error(
      `Guide scenario ${scenario.id} must contain at least one step`
    );
  }

  const indexById = new Map<GuideStepId, number>();
  for (const [index, step] of scenario.steps.entries()) {
    if (!registry[step.stepId]) {
      throw new Error(
        `Guide scenario ${scenario.id} references unregistered step ${step.stepId}`
      );
    }
    if (indexById.has(step.stepId)) {
      throw new Error(
        `Guide scenario ${scenario.id} contains duplicate step ${step.stepId}`
      );
    }
    indexById.set(step.stepId, index);
  }

  for (const step of scenario.steps) {
    for (const dependency of step.dependsOn ?? []) {
      if (dependency === step.stepId) {
        throw new Error(`Guide step ${step.stepId} cannot depend on itself`);
      }
      if (!indexById.has(dependency)) {
        throw new Error(
          `Guide step ${step.stepId} has dependency outside scenario: ${dependency}`
        );
      }
    }
  }

  const byId = new Map(scenario.steps.map((step) => [step.stepId, step]));
  const visiting = new Set<GuideStepId>();
  const visited = new Set<GuideStepId>();
  const visit = (stepId: GuideStepId) => {
    if (visiting.has(stepId)) {
      throw new Error(
        `Guide scenario ${scenario.id} contains a dependency cycle`
      );
    }
    if (visited.has(stepId)) return;
    visiting.add(stepId);
    for (const dependency of byId.get(stepId)?.dependsOn ?? []) {
      visit(dependency);
    }
    visiting.delete(stepId);
    visited.add(stepId);
  };
  for (const step of scenario.steps) visit(step.stepId);

  for (const [index, step] of scenario.steps.entries()) {
    for (const dependency of step.dependsOn ?? []) {
      if ((indexById.get(dependency) ?? -1) >= index) {
        throw new Error(
          `Guide dependency ${dependency} must appear before ${step.stepId}`
        );
      }
    }
  }
};

export const validateGuideScenarios = (
  scenarios: readonly GuideScenario[],
  registry: GuideStepRegistry
) => {
  const ids = new Set<string>();
  for (const scenario of scenarios) {
    if (ids.has(scenario.id)) {
      throw new Error(
        `Guide registry contains duplicate scenario ${scenario.id}`
      );
    }
    ids.add(scenario.id);
    validateGuideScenario(scenario, registry);
  }
};

export const resolveGuide = ({
  scenario,
  registry,
  context,
  selectedStepId,
}: {
  scenario: GuideScenario;
  registry: GuideStepRegistry;
  context: GuideContext;
  selectedStepId?: GuideStepId;
}): ResolvedGuide => {
  const doneById = new Map(
    scenario.steps.map(({ stepId }) => [
      stepId,
      registry[stepId].isComplete(context),
    ])
  );
  const steps = scenario.steps.map((scenarioStep) => {
    const definition = registry[scenarioStep.stepId];
    const done = doneById.get(scenarioStep.stepId) ?? false;
    return {
      scenarioStep,
      definition,
      done,
      blocked:
        !done &&
        (scenarioStep.dependsOn ?? []).some(
          (dependency) => !doneById.get(dependency)
        ),
      actions: definition.resolveActions(context),
    };
  });
  const activeStep =
    steps.find((step) => !step.done && !step.blocked) ?? steps.at(-1)!;
  const selectedStep = selectedStepId
    ? steps.find((step) => step.definition.id === selectedStepId)
    : undefined;
  const routeMatchedStep = steps.find((step) =>
    step.definition.matchesRoute(context.route)
  );
  const highlightedStep =
    selectedStep ?? (routeMatchedStep?.done ? activeStep : routeMatchedStep);
  const actionStep =
    highlightedStep && !highlightedStep.done ? highlightedStep : activeStep;

  return { steps, activeStep, highlightedStep, actionStep };
};
