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

// Extracts database resource parts while preserving the canonical instance.
// For example, "projects/p/instances/i/databases/d" has instance
// "projects/p/instances/i", while "instances/i/databases/d" has instance
// "instances/i".
export const extractDatabaseResourceName = (
  resource: string
): {
  // instance full name
  instance: string;
  // database full name preserving its project parent
  database: string;
  databaseName: string;
  instanceName: string;
} => {
  const pattern =
    /(?:^|\/)(?<parent>(?:projects\/[^/]+\/)?instances\/(?<instanceName>[^/]+))\/databases\/(?<databaseName>[^/]+)(?:$|\/)/;
  const matches = resource.match(pattern);

  const {
    parent: matchedParent,
    databaseName = String(UNKNOWN_ID),
    instanceName = String(UNKNOWN_ID),
  } = matches?.groups ?? {};
  const workspaceInstance = `${instanceNamePrefix}${instanceName}`;
  const instance = matchedParent || workspaceInstance;
  const database = `${instance}/${databaseNamePrefix}${databaseName}`;
  return {
    instance,
    instanceName,
    database,
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
