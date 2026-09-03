import {
  PROJECT_V1_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_INSTANCES,
  PROJECT_V1_ROUTE_PLAN_DETAIL,
  SQL_EDITOR_DATABASE_MODULE,
  WORKSPACE_ROUTE_MEMBERS,
  WORKSPACE_ROUTE_USERS,
} from "@/app/router/handles";
import {
  CREATE_INSTANCE_PRODUCT_INTRO,
  CREATE_PROJECT_PRODUCT_INTRO,
  CREATE_USER_PRODUCT_INTRO,
  GRANT_ACCESS_PRODUCT_INTRO,
  PRODUCT_INTRO_QUERY_KEY,
  PROJECT_INSTANCE_SYNCED_PRODUCT_INTRO,
} from "@/lib/productIntro";
import { extractProjectResourceName } from "@/utils/v1/project";
import type {
  GuideContext,
  GuideStepActions,
  GuideStepRegistry,
} from "./types";

const isRouteInside = (name: string | undefined, parent: string) =>
  name === parent || !!name?.startsWith(`${parent}.`);

const connectInstanceActions = (context: GuideContext): GuideStepActions => {
  return {
    select: {
      type: "navigate",
      target: {
        name: PROJECT_V1_ROUTE_INSTANCES,
        params: {
          projectId: extractProjectResourceName(context.projectName),
        },
        query: {
          [PRODUCT_INTRO_QUERY_KEY]: CREATE_INSTANCE_PRODUCT_INTRO,
        },
      },
    },
  };
};

const databaseActions = (context: GuideContext): GuideStepActions => ({
  select: {
    type: "navigate",
    target: {
      name: PROJECT_V1_ROUTE_DATABASES,
      params: {
        projectId: extractProjectResourceName(context.projectName),
      },
      query: {
        [PRODUCT_INTRO_QUERY_KEY]: PROJECT_INSTANCE_SYNCED_PRODUCT_INTRO,
      },
    },
  },
});

export const GUIDE_STEP_REGISTRY: GuideStepRegistry = {
  "create-project": {
    id: "create-project",
    analyticsKey: "create-project",
    labelKey: "workspace-setup-guide.steps.project",
    descriptionKey: "workspace-setup-guide.descriptions.project",
    isComplete: (context) => context.hasProject,
    matchesRoute: (route) => route.name === PROJECT_V1_ROUTE_DASHBOARD,
    resolveActions: () => ({
      select: {
        type: "navigate",
        target: {
          name: PROJECT_V1_ROUTE_DASHBOARD,
          query: {
            [PRODUCT_INTRO_QUERY_KEY]: CREATE_PROJECT_PRODUCT_INTRO,
          },
        },
      },
    }),
  },
  "connect-instance": {
    id: "connect-instance",
    analyticsKey: "connect-instance",
    labelKey: "workspace-setup-guide.steps.instance",
    descriptionKey: "workspace-setup-guide.descriptions.instance",
    isComplete: (context) => context.hasInstance,
    matchesRoute: (route) =>
      isRouteInside(route.name, PROJECT_V1_ROUTE_INSTANCES),
    resolveActions: connectInstanceActions,
  },
  "explore-database": {
    id: "explore-database",
    analyticsKey: "explore-database",
    labelKey: "workspace-setup-guide.steps.database",
    descriptionKey: "workspace-setup-guide.descriptions.database",
    isComplete: (context) => context.hasExploredDatabase,
    matchesRoute: (route) =>
      isRouteInside(route.name, PROJECT_V1_ROUTE_DATABASES),
    resolveActions: databaseActions,
  },
  "query-data": {
    id: "query-data",
    analyticsKey: "query-data",
    labelKey: "workspace-setup-guide.steps.query-data",
    descriptionKey: "workspace-setup-guide.descriptions.query-data",
    isComplete: (context) => context.hasRunStatement,
    matchesRoute: (route) =>
      isRouteInside(route.name, SQL_EDITOR_DATABASE_MODULE),
    resolveActions: (context) => ({
      primary: {
        type: "open-sql-editor",
        database: {
          name: context.databaseName,
          project: context.databaseProjectName,
        },
      },
    }),
  },
  "create-database-change": {
    id: "create-database-change",
    analyticsKey: "create-database-change",
    labelKey: "workspace-setup-guide.steps.create-database-change",
    descriptionKey: "workspace-setup-guide.descriptions.create-database-change",
    isComplete: (context) => context.hasCreatedChangeIssue,
    matchesRoute: (route) =>
      isRouteInside(route.name, PROJECT_V1_ROUTE_PLAN_DETAIL),
    resolveActions: (context) => ({
      select: {
        type: "create-change",
        project: context.databaseProjectName,
        database: context.databaseName,
      },
    }),
  },
  "create-user": {
    id: "create-user",
    analyticsKey: "add-teammate",
    labelKey: "workspace-setup-guide.steps.add-teammate",
    descriptionKey: "workspace-setup-guide.descriptions.add-teammate",
    isComplete: (context) => context.hasOtherWorkspaceMember,
    matchesRoute: (route) => isRouteInside(route.name, WORKSPACE_ROUTE_USERS),
    resolveActions: () => ({
      select: {
        type: "navigate",
        target: {
          name: WORKSPACE_ROUTE_USERS,
          query: {
            [PRODUCT_INTRO_QUERY_KEY]: CREATE_USER_PRODUCT_INTRO,
          },
        },
      },
    }),
  },
  "grant-access": {
    id: "grant-access",
    analyticsKey: "add-teammate",
    labelKey: "workspace-setup-guide.steps.add-teammate",
    descriptionKey: "workspace-setup-guide.descriptions.add-teammate",
    isComplete: (context) => context.hasOtherWorkspaceMember,
    matchesRoute: (route) => isRouteInside(route.name, WORKSPACE_ROUTE_MEMBERS),
    resolveActions: () => ({
      select: {
        type: "navigate",
        target: {
          name: WORKSPACE_ROUTE_MEMBERS,
          query: {
            [PRODUCT_INTRO_QUERY_KEY]: GRANT_ACCESS_PRODUCT_INTRO,
          },
        },
      },
    }),
  },
};
