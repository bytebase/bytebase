import type {
  GuideContext,
  GuideJourney,
  GuideStepDefinition,
  GuideStepId,
  ResolvedGuide,
} from "./types";

const definitionsById = (definitions: readonly GuideStepDefinition[]) => {
  const byId = new Map<GuideStepId, GuideStepDefinition>();
  for (const definition of definitions) {
    if (byId.has(definition.id)) {
      throw new Error(
        `Guide contains duplicate step definition ${definition.id}`
      );
    }
    byId.set(definition.id, definition);
  }
  return byId;
};

const validateGuideJourneyWithDefinitions = (
  journey: GuideJourney,
  definitions: ReadonlyMap<GuideStepId, GuideStepDefinition>
) => {
  if (journey.steps.length === 0) {
    throw new Error(
      `Guide journey ${journey.id} must contain at least one step`
    );
  }

  const indexById = new Map<GuideStepId, number>();
  for (const [index, step] of journey.steps.entries()) {
    if (!definitions.has(step.stepId)) {
      throw new Error(
        `Guide journey ${journey.id} references unregistered step ${step.stepId}`
      );
    }
    if (indexById.has(step.stepId)) {
      throw new Error(
        `Guide journey ${journey.id} contains duplicate step ${step.stepId}`
      );
    }
    indexById.set(step.stepId, index);
  }

  for (const step of journey.steps) {
    for (const dependency of step.dependsOn ?? []) {
      if (dependency === step.stepId) {
        throw new Error(`Guide step ${step.stepId} cannot depend on itself`);
      }
      if (!indexById.has(dependency)) {
        throw new Error(
          `Guide step ${step.stepId} has dependency outside journey: ${dependency}`
        );
      }
    }
  }

  const byId = new Map(journey.steps.map((step) => [step.stepId, step]));
  const visiting = new Set<GuideStepId>();
  const visited = new Set<GuideStepId>();
  const visit = (stepId: GuideStepId) => {
    if (visiting.has(stepId)) {
      throw new Error(
        `Guide journey ${journey.id} contains a dependency cycle`
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
  for (const step of journey.steps) visit(step.stepId);

  for (const [index, step] of journey.steps.entries()) {
    for (const dependency of step.dependsOn ?? []) {
      if ((indexById.get(dependency) ?? -1) >= index) {
        throw new Error(
          `Guide dependency ${dependency} must appear before ${step.stepId}`
        );
      }
    }
  }
};

export const validateGuideJourney = (
  journey: GuideJourney,
  definitions: readonly GuideStepDefinition[]
) => {
  validateGuideJourneyWithDefinitions(journey, definitionsById(definitions));
};

export const validateGuideJourneys = (
  journeys: readonly GuideJourney[],
  definitions: readonly GuideStepDefinition[]
) => {
  const definitionById = definitionsById(definitions);
  const ids = new Set<string>();
  for (const journey of journeys) {
    if (ids.has(journey.id)) {
      throw new Error(
        `Guide registry contains duplicate journey ${journey.id}`
      );
    }
    ids.add(journey.id);
    validateGuideJourneyWithDefinitions(journey, definitionById);
  }
};

export const resolveGuide = ({
  journey,
  definitions,
  context,
  selectedStepId,
}: {
  journey: GuideJourney;
  definitions: readonly GuideStepDefinition[];
  context: GuideContext;
  selectedStepId?: GuideStepId;
}): ResolvedGuide => {
  const definitionById = definitionsById(definitions);
  const doneById = new Map(
    journey.steps.map(({ stepId }) => [
      stepId,
      definitionById.get(stepId)?.isComplete(context) ?? false,
    ])
  );
  const allSteps = journey.steps.map((journeyStep) => {
    const definition = definitionById.get(journeyStep.stepId);
    if (!definition) {
      throw new Error(
        `Guide journey ${journey.id} references unregistered step ${journeyStep.stepId}`
      );
    }
    const done = doneById.get(journeyStep.stepId) ?? false;
    return {
      journeyStep,
      definition,
      done,
      blocked:
        !done &&
        (journeyStep.dependsOn ?? []).some(
          (dependency) => !doneById.get(dependency)
        ),
      actions: definition.resolveActions(context),
    };
  });
  const complete = journey.steps.every(
    ({ stepId }) => doneById.get(stepId) === true
  );
  const steps = allSteps;
  const activeStep = complete
    ? undefined
    : steps.find((step) => !step.done && !step.blocked);
  const selectedStep = selectedStepId
    ? steps.find((step) => step.definition.id === selectedStepId)
    : undefined;
  const routeMatchedStep = steps.find((step) =>
    step.definition.matchesRoute(context.route)
  );
  const highlightedStep = complete
    ? undefined
    : (selectedStep ??
      (routeMatchedStep?.done ? activeStep : routeMatchedStep));
  const actionStep = complete
    ? undefined
    : highlightedStep && !highlightedStep.done
      ? highlightedStep
      : activeStep;

  return {
    steps,
    complete,
    activeStep,
    highlightedStep,
    actionStep,
  };
};
