import {
  PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DATABASE_REVISION_DETAIL,
  PROJECT_V1_ROUTE_DATABASES,
  SETTING_ROUTE_WORKSPACE_SUBSCRIPTION,
  SQL_EDITOR_DATABASE_MODULE,
} from "@/app/router/handles";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { extractDatabaseResourceName } from "./v1/database";
import { extractProjectResourceName } from "./v1/project";

export const autoDatabaseRoute = (database: Database) => {
  const name = PROJECT_V1_ROUTE_DATABASE_DETAIL;

  const projectId = extractProjectResourceName(database.project);
  const {
    instance,
    instanceName: instanceId,
    databaseName,
  } = extractDatabaseResourceName(database.name);
  return {
    name,
    params: {
      projectId,
      instanceId,
      databaseName,
    },
    query: {
      parent: instance,
    },
  };
};

export const autoDatabaseChangelogRoute = (
  database: Database,
  changelogId: string
) => {
  const route = autoDatabaseRoute(database);
  return {
    ...route,
    name: PROJECT_V1_ROUTE_DATABASE_CHANGELOG_DETAIL,
    params: {
      ...route.params,
      changelogId,
    },
  };
};

export const autoDatabaseRevisionRoute = (
  database: Database,
  revisionId: string
) => {
  const route = autoDatabaseRoute(database);
  return {
    ...route,
    name: PROJECT_V1_ROUTE_DATABASE_REVISION_DETAIL,
    params: {
      ...route.params,
      revisionId,
    },
  };
};

export const autoSQLEditorDatabaseRoute = (
  database: Pick<Database, "name" | "project">
) => {
  const { instanceName, databaseName } = extractDatabaseResourceName(
    database.name
  );
  return {
    name: SQL_EDITOR_DATABASE_MODULE,
    params: {
      project: extractProjectResourceName(database.project),
      instance: instanceName,
      database: databaseName,
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
