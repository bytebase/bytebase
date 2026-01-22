import {
  checkQuerierPermission,
  useEnvironmentV1Store,
  useProjectV1Store,
} from "@/store";
import {
  databaseNamePrefix,
  instanceNamePrefix,
} from "@/store/modules/v1/common";
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
import type { Environment } from "@/types/v1/environment";
import { hasWorkspacePermissionV2 } from "../iam";
import { extractProjectResourceName } from "./project";

export const databaseV1Url = (db: Database) => {
  return databaseV1UrlWithProject(db.project, db.name);
};

const databaseV1UrlWithProject = (project: string, database: string) => {
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
  return useProjectV1Store().getProjectByName(database.project);
};

// Get effective environment entity
export const getDatabaseEnvironment = (database: Database): Environment => {
  return useEnvironmentV1Store().getEnvironmentByName(
    database.effectiveEnvironment ?? ""
  );
};
