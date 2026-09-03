import type { ReactRoute, RouteTarget } from "@/app/router";

export type GuideStepId =
  | "create-project"
  | "connect-instance"
  | "explore-database"
  | "query-data"
  | "create-database-change"
  | "create-user"
  | "grant-access";

export type GuideScenarioId = "query-data" | "create-database-change";

export type GuideWorkspaceUsage = "team" | "solo";

export type GuideJourneyId = "workspace-setup" | GuideScenarioId;

export type GuideAnalyticsKey =
  | Exclude<GuideStepId, "create-user" | "grant-access">
  | "add-teammate";

export type GuideRoute = Pick<ReactRoute, "name" | "params">;

export type GuideContext = {
  hasProject: boolean;
  hasInstance: boolean;
  hasExploredDatabase: boolean;
  hasRunStatement: boolean;
  hasCreatedChangeIssue: boolean;
  isSaaS: boolean;
  hasOtherHumanUser: boolean;
  hasOtherWorkspaceMember: boolean;
  projectName: string;
  instanceName: string;
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

export type GuideJourneyStep = {
  stepId: GuideStepId;
  kind?: "prerequisite" | "learning" | "modifier";
  dependsOn?: readonly GuideStepId[];
};

export type GuideCompletionActionId = "open-sql-editor" | "create-change";

export type GuideJourney = {
  id: GuideJourneyId;
  scenarioId?: GuideScenarioId;
  completionTitleKey: string;
  completionDescriptionKey: string;
  completionActions: readonly GuideCompletionActionId[];
  steps: readonly GuideJourneyStep[];
};

export type ResolvedGuideStep = {
  journeyStep: GuideJourneyStep;
  definition: GuideStepDefinition;
  done: boolean;
  blocked: boolean;
  actions: GuideStepActions;
};

export type ResolvedGuide = {
  steps: ResolvedGuideStep[];
  complete: boolean;
  activeStep: ResolvedGuideStep | undefined;
  highlightedStep: ResolvedGuideStep | undefined;
  actionStep: ResolvedGuideStep | undefined;
};
