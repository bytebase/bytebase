import {
  DATABASE_ROUTE_DASHBOARD,
  INSTANCE_ROUTE_DASHBOARD,
  INSTANCE_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_INSTANCES,
  SQL_EDITOR_DATABASE_MODULE,
} from "@/app/router/handles";
import {
  CREATE_INSTANCE_PRODUCT_INTRO,
  CREATE_PROJECT_PRODUCT_INTRO,
  PREPARE_DATABASE_PRODUCT_INTRO,
  PREPARE_DATABASE_TRANSFER_TIP,
  PRODUCT_INTRO_QUERY_KEY,
  PRODUCT_INTRO_TIP_QUERY_KEY,
  PROJECT_INSTANCE_SYNCED_PRODUCT_INTRO,
} from "@/lib/productIntro";
import { extractProjectResourceName } from "@/utils/v1/project";
import type { GuideStepRegistry } from "./types";

const isRouteInside = (name: string | undefined, parent: string) =>
  name === parent || !!name?.startsWith(`${parent}.`);

export const GUIDE_STEP_REGISTRY: GuideStepRegistry = {
  "create-project": {
    id: "create-project",
    analyticsKey: "hasProject",
    labelKey: "workspace-setup-guide.steps.project",
    descriptionKey: "workspace-setup-guide.descriptions.project",
    isComplete: (context) => context.hasProject,
    matchesRoute: (route) => route.name === PROJECT_V1_ROUTE_DASHBOARD,
    resolveActions: (context) =>
      context.hasProject
        ? {}
        : {
            select: {
              type: "navigate",
              target: {
                name: PROJECT_V1_ROUTE_DASHBOARD,
                query: {
                  [PRODUCT_INTRO_QUERY_KEY]: CREATE_PROJECT_PRODUCT_INTRO,
                },
              },
            },
          },
  },
  "connect-instance": {
    id: "connect-instance",
    analyticsKey: "hasInstance",
    labelKey: "workspace-setup-guide.steps.instance",
    descriptionKey: "workspace-setup-guide.descriptions.instance",
    isComplete: (context) => context.hasInstance,
    matchesRoute: (route) =>
      isRouteInside(route.name, INSTANCE_ROUTE_DASHBOARD) ||
      isRouteInside(route.name, PROJECT_V1_ROUTE_INSTANCES),
    resolveActions: (context) => {
      if (context.hasInstance) return {};
      return {
        select: {
          type: "navigate",
          target: context.projectName
            ? {
                name: PROJECT_V1_ROUTE_INSTANCES,
                params: {
                  projectId: extractProjectResourceName(context.projectName),
                },
                query: {
                  [PRODUCT_INTRO_QUERY_KEY]: CREATE_INSTANCE_PRODUCT_INTRO,
                },
              }
            : {
                name: INSTANCE_ROUTE_DASHBOARD,
                query: {
                  [PRODUCT_INTRO_QUERY_KEY]: CREATE_INSTANCE_PRODUCT_INTRO,
                },
              },
        },
      };
    },
  },
  "explore-database": {
    id: "explore-database",
    analyticsKey: "hasExploredDatabase",
    labelKey: "workspace-setup-guide.steps.database",
    descriptionKey: "workspace-setup-guide.descriptions.database",
    isComplete: (context) => context.hasExploredDatabase,
    matchesRoute: (route) =>
      isRouteInside(route.name, DATABASE_ROUTE_DASHBOARD) ||
      isRouteInside(route.name, PROJECT_V1_ROUTE_DATABASES) ||
      route.name === INSTANCE_ROUTE_DATABASE_DETAIL,
    resolveActions: (context) => ({
      select: {
        type: "navigate",
        target:
          context.databaseName && context.databaseProjectName
            ? {
                name: PROJECT_V1_ROUTE_DATABASES,
                params: {
                  projectId: extractProjectResourceName(
                    context.databaseProjectName
                  ),
                },
                query: {
                  [PRODUCT_INTRO_QUERY_KEY]:
                    PROJECT_INSTANCE_SYNCED_PRODUCT_INTRO,
                },
              }
            : {
                name: DATABASE_ROUTE_DASHBOARD,
                query: {
                  [PRODUCT_INTRO_QUERY_KEY]: PREPARE_DATABASE_PRODUCT_INTRO,
                  [PRODUCT_INTRO_TIP_QUERY_KEY]: PREPARE_DATABASE_TRANSFER_TIP,
                },
              },
      },
    }),
  },
  "query-data": {
    id: "query-data",
    analyticsKey: "hasFirstQuery",
    labelKey: "workspace-setup-guide.steps.query",
    descriptionKey: "workspace-setup-guide.descriptions.sql-editor",
    isComplete: (context) => context.hasFirstQuery,
    matchesRoute: (route) =>
      isRouteInside(route.name, SQL_EDITOR_DATABASE_MODULE),
    resolveActions: (context) => {
      if (!context.databaseName || !context.databaseProjectName) return {};
      return {
        primary: {
          type: "open-sql-editor",
          database: {
            name: context.databaseName,
            project: context.databaseProjectName,
          },
        },
        secondary: {
          type: "create-change",
          project: context.databaseProjectName,
          database: context.databaseName,
        },
      };
    },
  },
};
