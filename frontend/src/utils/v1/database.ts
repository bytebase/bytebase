import { getProjectByName } from "@/stores/app/projectAccess";
import {
  databaseNamePrefix,
  instanceNamePrefix,
} from "@/stores/modules/v1/common";
import type { QueryPermission } from "@/types";
import {
  isValidDatabaseName,
  QueryPermissionQueryAny,
  UNKNOWN_ID,
  unknownInstanceResource,
} from "@/types";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { InstanceResource } from "@/types/proto-es/v1/instance_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { type Environment, unknownEnvironment } from "@/types/v1/environment";
import { appStoreUtilBridge } from "@/utils/app-store-bridge";
import { hasWorkspacePermissionV2 } from "../iam";
import { checkQuerierPermission } from "./iam";
import { extractProjectResourceName } from "./project";

export const databaseV1UrlWithSuffix = (db: Database, suffix: string) => {
  return databaseV1UrlWithProject(db.project, db.name, suffix);
};

const databaseV1UrlWithProject = (
  project: string,
  database: string,
  suffix = ""
) => {
  const projectId = extractProjectResourceName(project);
  const { databaseName, instanceName } = extractDatabaseResourceName(database);
  const parent = extractDatabaseParentResourceName(database);

  return `/projects/${encodeURIComponent(projectId)}/${instanceNamePrefix}${encodeURIComponent(instanceName)}/${databaseNamePrefix}${encodeURIComponent(databaseName)}${suffix}?parent=${encodeURIComponent(parent)}`;
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

// Extracts the database parent while preserving its full scope.
// For example, "projects/p/instances/i/databases/d" returns
// "projects/p/instances/i".
export const extractDatabaseParentResourceName = (resource: string): string => {
  const marker = "/databases/";
  const index = resource.indexOf(marker);
  return index < 0 ? "" : resource.slice(0, index);
};

// isDatabaseV1Queryable checks if database allowed to query in SQL Editor.
export const isDatabaseV1Queryable = (
  database: Database,
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

// Get instance resource with fallback to unknown
export const getInstanceResource = (database: Database): InstanceResource => {
  if (database.instanceResource) {
    return database.instanceResource;
  }
  const { instance } = extractDatabaseResourceName(database.name);
  return {
    ...unknownInstanceResource(),
    name: instance,
  };
};

// Get database engine
export const getDatabaseEngine = (database: Database): Engine => {
  return getInstanceResource(database).engine;
};

// Get project entity (sync - assumes cached)
export const getDatabaseProject = (database: Database): Project => {
  return getProjectByName(database.project);
};

// Get effective environment entity
export const getDatabaseEnvironment = (database: Database): Environment => {
  return (
    appStoreUtilBridge()?.getEnvironmentByName(
      database.effectiveEnvironment ?? ""
    ) ?? unknownEnvironment()
  );
};
