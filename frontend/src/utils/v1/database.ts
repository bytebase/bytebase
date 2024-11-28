import { orderBy } from "lodash-es";
import { allowGhostForDatabase } from "@/components/IssueV1/components/Sidebar/GhostSection/common";
import { checkQuerierPermission, hasFeature } from "@/store";
import {
  databaseNamePrefix,
  instanceNamePrefix,
} from "@/store/modules/v1/common";
import { QueryPermissionQueryAny, UNKNOWN_ID } from "@/types";
import type { ComposedDatabase, QueryPermission } from "@/types";
import { State } from "@/types/proto/v1/common";
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

export const sortDatabaseV1List = (databaseList: ComposedDatabase[]) => {
  return orderBy(
    databaseList,
    [
      (db) => db.effectiveEnvironmentEntity.order,
      (db) => db.instance,
      (db) => db.projectEntity.key,
      (db) => db.databaseName,
    ],
    ["desc", "asc", "asc", "asc"]
  );
};

export const isArchivedDatabaseV1 = (db: ComposedDatabase): boolean => {
  // TODO(steven): check if the related instance is deleted.
  // if (db.instanceEntity.state === State.DELETED) {
  //   return true;
  // }
  if (db.effectiveEnvironmentEntity.state === State.DELETED) {
    return true;
  }
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

type DatabaseV1FilterFields = "name" | "project" | "instance" | "environment";
export function filterDatabaseV1ByKeyword(
  db: ComposedDatabase,
  keyword: string,
  columns: DatabaseV1FilterFields[] = ["name"]
): boolean {
  keyword = keyword.trim().toLowerCase();
  if (!keyword) {
    // Skip the filter
    return true;
  }

  if (
    columns.includes("name") &&
    db.databaseName.toLowerCase().includes(keyword)
  ) {
    return true;
  }

  if (
    columns.includes("project") &&
    db.projectEntity.title.toLowerCase().includes(keyword)
  ) {
    return true;
  }

  if (
    columns.includes("instance") &&
    db.instanceResource.title.toLowerCase().includes(keyword)
  ) {
    return true;
  }

  if (
    columns.includes("environment") &&
    db.effectiveEnvironmentEntity.title.toLowerCase().includes(keyword)
  ) {
    return true;
  }

  return false;
}

export const MIN_GHOST_SUPPORT_MYSQL_VERSION = "5.6.0";

export const MIN_GHOST_SUPPORT_MARIADB_VERSION = "10.6.0";

export function allowGhostMigrationV1(
  databaseList: ComposedDatabase[]
): boolean {
  return databaseList.every((db) => allowGhostForDatabase(db));
}
