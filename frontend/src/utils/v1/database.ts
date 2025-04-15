import { checkQuerierPermission, hasFeature } from "@/store";
import {
  databaseNamePrefix,
  instanceNamePrefix,
} from "@/store/modules/v1/common";
import { QueryPermissionQueryAny, UNKNOWN_ID } from "@/types";
import type { ComposedDatabase, QueryPermission } from "@/types";
import {
  hasPermissionToCreateChangeDatabaseIssue,
  hasWorkspacePermissionV2,
} from "../iam";
import { extractProjectResourceName } from "./project";

export const databaseV1Url = (db: ComposedDatabase) => {
  const project = extractProjectResourceName(db.project);
  const { databaseName, instanceName } = extractDatabaseResourceName(db.name);
  return `/projects/${encodeURIComponent(project)}/${instanceNamePrefix}${encodeURIComponent(instanceName)}/${databaseNamePrefix}${encodeURIComponent(databaseName)}`;
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

// TODO(p0ny): remove
export const isArchivedDatabaseV1 = (_db: ComposedDatabase): boolean => {
  return false;
};

// isDatabaseV1Alterable checks if database alterable for user.
export const isDatabaseV1Alterable = (database: ComposedDatabase): boolean => {
  if (!hasFeature("bb.feature.access-control")) {
    // The current plan doesn't have access control feature.
    // Fallback to true.
    return true;
  }

  if (hasPermissionToCreateChangeDatabaseIssue(database)) {
    return true;
  }

  return false;
};

// isDatabaseV1Queryable checks if database allowed to query in SQL Editor.
export const isDatabaseV1Queryable = (
  database: ComposedDatabase,
  permissions: QueryPermission[] = QueryPermissionQueryAny,
  schema?: string,
  table?: string
): boolean => {
  if (permissions.some((permission) => hasWorkspacePermissionV2(permission))) {
    return true;
  }

  if (checkQuerierPermission(database, permissions, schema, table)) {
    return true;
  }

  // denied otherwise
  return false;
};
