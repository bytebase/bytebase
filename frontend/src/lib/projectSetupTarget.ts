import type { RouteTarget } from "@/app/router";
import {
  DATABASE_ROUTE_DASHBOARD,
  PROJECT_V1_ROUTE_DATABASES,
  PROJECT_V1_ROUTE_INSTANCE_CREATE,
} from "@/app/router/handles";
import {
  PREPARE_DATABASE_PRODUCT_INTRO,
  PREPARE_DATABASE_TRANSFER_TIP,
  PRODUCT_INTRO_QUERY_KEY,
  PRODUCT_INTRO_TIP_QUERY_KEY,
} from "./productIntro";
import type { ProjectSetupStep } from "./projectSetup";

export function projectSetupTarget(
  step: ProjectSetupStep,
  projectId: string
): RouteTarget {
  const params = { projectId };
  switch (step) {
    case "connect":
      return { name: PROJECT_V1_ROUTE_INSTANCE_CREATE, params };
    case "transferDatabases":
      return {
        name: DATABASE_ROUTE_DASHBOARD,
        query: {
          [PRODUCT_INTRO_QUERY_KEY]: PREPARE_DATABASE_PRODUCT_INTRO,
          [PRODUCT_INTRO_TIP_QUERY_KEY]: PREPARE_DATABASE_TRANSFER_TIP,
          targetProject: `projects/${projectId}`,
        },
      };
    case "createDatabase":
      return {
        name: PROJECT_V1_ROUTE_DATABASES,
        params,
        query: { createDatabase: "1" },
      };
  }
}
