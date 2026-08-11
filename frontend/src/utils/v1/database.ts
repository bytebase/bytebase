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

// Extracts the database resource parts while preserving the complete parent.
// For example, "projects/p/instances/i/databases/d" has parent
// "projects/p/instances/i", while "instances/i/databases/d" has parent
// "instances/i".
export const extractDatabaseResourceName = (
  resource: string
): {
  // database parent preserving its full scope
  parent: string;
  // instance full name
  instance: string;
  // database full name
  database: string;
  databaseName: string;
  instanceName: string;
} => {
  const pattern =
    /(?:^|\/)(?<parent>(?:projects\/[^/]+\/)?instances\/(?<instanceName>[^/]+))\/databases\/(?<databaseName>[^/]+)(?:$|\/)/;
  const matches = resource.match(pattern);

  const {
    parent = "",
    databaseName = String(UNKNOWN_ID),
    instanceName = String(UNKNOWN_ID),
  } = matches?.groups ?? {};
  return {
    parent,
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
