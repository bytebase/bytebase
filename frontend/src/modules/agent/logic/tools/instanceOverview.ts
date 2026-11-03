import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { databaseServiceClientConnect } from "@/api";
import { silentContextKey } from "@/api/context-key";
import {
  SearchDatabasesRequestSchema,
  SyncStatus,
} from "@/types/proto-es/v1/database_service_pb";
import { getDefaultPagination } from "@/utils/pagination";
import { celString } from "@/utils/v1/celLiteral";

const MAX_OUT_OF_SYNC_DATABASES = 10;

type InstanceSyncDatabase = {
  name: string;
  syncStatus: string;
};

type InstanceOverview =
  | { instance: string; status: "unavailable" }
  | {
      instance: string;
      status: "available";
      coverage: "accessible_databases_only";
      databasesOnPage: number;
      partial: boolean;
      outOfSyncOnPageCount: number;
      outOfSyncDatabases: InstanceSyncDatabase[];
      outOfSyncDatabasesPartial: boolean;
    };

function isCanonicalInstanceName(name: string): boolean {
  const parts = name.split("/");
  return (
    (parts.length === 2 &&
      parts[0] === "instances" &&
      parts[1] !== "-" &&
      parts[1].toLowerCase() !== "create") ||
    (parts.length === 4 &&
      parts[0] === "projects" &&
      parts[1] !== "-" &&
      parts[2] === "instances" &&
      parts[3] !== "-" &&
      parts[3].toLowerCase() !== "create")
  );
}

export function resolveInstanceNameFromPagePath(
  pagePath: string
): string | undefined {
  try {
    const url = new URL(pagePath, "http://bytebase.local");
    if (url.origin !== "http://bytebase.local") return undefined;
    const path = url.pathname;
    const parts = path
      .split("/")
      .filter(Boolean)
      .map((part) => decodeURIComponent(part));
    const name =
      parts.length === 2 &&
      parts[0] === "instances" &&
      parts[1].toLowerCase() !== "create"
        ? `${parts[0]}/${parts[1]}`
        : parts.length === 4 &&
            parts[0] === "projects" &&
            parts[2] === "instances" &&
            parts[3].toLowerCase() !== "create"
          ? `${parts[0]}/${parts[1]}/${parts[2]}/${parts[3]}`
          : undefined;
    return name && isCanonicalInstanceName(name) ? name : undefined;
  } catch {
    return undefined;
  }
}

const enumName = (value: number): string => {
  const name = SyncStatus[value];
  return typeof name === "string" ? name : "UNSPECIFIED";
};

function isDatabaseOnInstance(databaseName: string, instanceName: string) {
  const prefix = `${instanceName}/databases/`;
  return (
    databaseName.startsWith(prefix) &&
    databaseName.split("/").length === instanceName.split("/").length + 2
  );
}

export async function collectInstanceOverview(
  instanceName: string
): Promise<InstanceOverview> {
  if (!isCanonicalInstanceName(instanceName)) {
    return { instance: instanceName, status: "unavailable" };
  }

  try {
    const response = await databaseServiceClientConnect.searchDatabases(
      create(SearchDatabasesRequestSchema, {
        pageSize: getDefaultPagination(),
        filter: `instance == ${celString(instanceName)}`,
      }),
      { contextValues: createContextValues().set(silentContextKey, true) }
    );
    const databases = response.databases.filter((database) =>
      isDatabaseOnInstance(database.name, instanceName)
    );
    const outOfSyncDatabases = databases
      .filter((database) => database.syncStatus !== SyncStatus.OK)
      .map((database) => ({
        name: database.name,
        syncStatus: enumName(database.syncStatus),
      }));

    return {
      instance: instanceName,
      status: "available",
      coverage: "accessible_databases_only",
      databasesOnPage: databases.length,
      partial: Boolean(response.nextPageToken),
      outOfSyncOnPageCount: outOfSyncDatabases.length,
      outOfSyncDatabases: outOfSyncDatabases.slice(
        0,
        MAX_OUT_OF_SYNC_DATABASES
      ),
      outOfSyncDatabasesPartial:
        outOfSyncDatabases.length > MAX_OUT_OF_SYNC_DATABASES,
    };
  } catch {
    return { instance: instanceName, status: "unavailable" };
  }
}

export function createInstanceOverviewTool(instanceName?: string) {
  return async (): Promise<string> => {
    if (!instanceName) {
      return JSON.stringify({ status: "unavailable" });
    }
    return JSON.stringify(await collectInstanceOverview(instanceName));
  };
}
