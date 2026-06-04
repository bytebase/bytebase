import {
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DATABASES,
  SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
} from "@/react/router/handles";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { extractDatabaseResourceName, extractProjectResourceName } from "./v1";

export const autoDatabaseRoute = (database: Database) => {
  const name = PROJECT_V1_ROUTE_DATABASE_DETAIL;

  const projectId = extractProjectResourceName(database.project);
  const { instanceName: instanceId, databaseName } =
    extractDatabaseResourceName(database.name);
  return {
    name,
    params: {
      projectId,
      instanceId,
      databaseName,
    },
  };
};

export const autoProjectRoute = (project: Project) => {
  return {
    name: PROJECT_V1_ROUTE_DATABASES,
    params: {
      projectId: extractProjectResourceName(project.name),
    },
  };
};

export const autoSubscriptionRoute = () => {
  return { name: SETTING_ROUTE_WORKSPACE_SUBSCRIPTION };
};
