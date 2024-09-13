import { type Router } from "vue-router";
import {
  PROJECT_V1_ROUTE_DATABASE_DETAIL,
  PROJECT_V1_ROUTE_DATABASES,
} from "@/router/dashboard/projectV1";
import {
  SQL_EDITOR_SETTING_DATABASE_DETAIL_MODULE,
  SQL_EDITOR_SETTING_PROJECT_MODULE,
} from "@/router/sqlEditor";
import type { ComposedDatabase, ComposedProject } from "@/types";
import { extractDatabaseResourceName, extractProjectResourceName } from "./v1";

const isSQLEditorRoute = (router: Router) => {
  return router.currentRoute.value.name?.toString().startsWith("sql-editor");
};

export const autoDatabaseRoute = (
  router: Router,
  database: ComposedDatabase
) => {
  const name = isSQLEditorRoute(router)
    ? SQL_EDITOR_SETTING_DATABASE_DETAIL_MODULE
    : PROJECT_V1_ROUTE_DATABASE_DETAIL;

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

export const autoProjectRoute = (router: Router, project: ComposedProject) => {
  if (isSQLEditorRoute(router)) {
    return {
      name: SQL_EDITOR_SETTING_PROJECT_MODULE,
      hash: `#${project.name}`,
    };
  }
  return {
    name: PROJECT_V1_ROUTE_DATABASES,
    params: {
      projectId: extractProjectResourceName(project.name),
    },
  };
};
