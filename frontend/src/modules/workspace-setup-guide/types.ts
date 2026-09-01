import type { ReactRoute, RouteTarget } from "@/app/router";

export type GuideStepId =
  | "create-project"
  | "connect-instance"
  | "explore-database"
  | "query-data";

export type GuideScenarioId = "learn-bytebase-basics";

export type GuideAnalyticsKey =
  | "hasProject"
  | "hasInstance"
  | "hasExploredDatabase"
  | "hasFirstQuery";

export type GuideRoute = Pick<ReactRoute, "name" | "params">;

export type GuideContext = {
  hasProject: boolean;
  hasInstance: boolean;
  hasExploredDatabase: boolean;
  hasFirstQuery: boolean;
  projectName: string;
  databaseProjectName: string;
  databaseName: string;
  route: GuideRoute;
};

export type GuideDatabase = {
  name: string;
  project: string;
};

export type GuideAction =
  | { type: "navigate"; target: RouteTarget }
  | { type: "open-sql-editor"; database: GuideDatabase }
  | {
      type: "create-change";
      project: string;
      database: string;
    };

export type GuideStepActions = {
  select?: GuideAction;
  primary?: GuideAction;
  secondary?: GuideAction;
};

export type GuideStepDefinition = {
  id: GuideStepId;
  analyticsKey: GuideAnalyticsKey;
  labelKey: string;
  descriptionKey: string;
  isComplete: (context: GuideContext) => boolean;
  matchesRoute: (route: GuideRoute) => boolean;
  resolveActions: (context: GuideContext) => GuideStepActions;
};

export type GuideStepRegistry = Readonly<
  Record<GuideStepId, GuideStepDefinition>
>;

export type ScenarioStep = {
  stepId: GuideStepId;
  dependsOn?: readonly GuideStepId[];
};

export type GuideScenario = {
  id: GuideScenarioId;
  steps: readonly ScenarioStep[];
};

export type ResolvedGuideStep = {
  scenarioStep: ScenarioStep;
  definition: GuideStepDefinition;
  done: boolean;
  blocked: boolean;
  actions: GuideStepActions;
};

export type ResolvedGuide = {
  steps: ResolvedGuideStep[];
  activeStep: ResolvedGuideStep;
  highlightedStep: ResolvedGuideStep | undefined;
  actionStep: ResolvedGuideStep;
};
