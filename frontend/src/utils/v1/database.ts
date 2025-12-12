import { checkQuerierPermission } from "@/store";
import {
  databaseNamePrefix,
  instanceNamePrefix,
} from "@/store/modules/v1/common";
import type { ComposedDatabase, QueryPermission } from "@/types";
import {
  isValidDatabaseName,
  QueryPermissionQueryAny,
  UNKNOWN_ID,
} from "@/types";
import { hasWorkspacePermissionV2 } from "../iam";
import { extractProjectResourceName } from "./project";

export const databaseV1Url = (db: ComposedDatabase) => {
  return databaseV1UrlWithProject(db.project, db.name);
};

export const databaseV1UrlWithProject = (project: string, database: string) => {
  const projectId = extractProjectResourceName(project);
  const { databaseName, instanceName } = extractDatabaseResourceName(database);

  return `/projects/${encodeURIComponent(projectId)}/${instanceNamePrefix}${encodeURIComponent(instanceName)}/${databaseNamePrefix}${encodeURIComponent(databaseName)}`;
};

export const extractDatabaseResourceName = (
  resource: string
): {
  // instance full name
  instance: string;
  // database full name
  database: string;
  databaseName: string;
  instanceName: string;
} => {
  const pattern =
    /(?:^|\/)instances\/(?<instanceName>[^/]+)\/databases\/(?<databaseName>[^/]+)(?:$|\/)/;
  const matches = resource.match(pattern);

  const {
    databaseName = String(UNKNOWN_ID),
    instanceName = String(UNKNOWN_ID),
  } = matches?.groups ?? {};
  return {
    instance: `${instanceNamePrefix}${instanceName}`,
    instanceName,
    database: `${instanceNamePrefix}${instanceName}/${databaseNamePrefix}${databaseName}`,
    databaseName,
  };
};

// isDatabaseV1Queryable checks if database allowed to query in SQL Editor.
export const isDatabaseV1Queryable = (
  database: ComposedDatabase,
  permissions: QueryPermission[] = QueryPermissionQueryAny,
  schema?: string,
  table?: string
): boolean => {
  if (!isValidDatabaseName(database.name)) {
    return false;
  }

  if (permissions.some((permission) => hasWorkspacePermissionV2(permission))) {
    return true;
  }

  if (checkQuerierPermission(database, permissions, schema, table)) {
    return true;
  }

  // denied otherwise
  return false;
};
