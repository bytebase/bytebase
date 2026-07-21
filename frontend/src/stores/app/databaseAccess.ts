import { create as createProto } from "@bufbuild/protobuf";
import { UNKNOWN_ID } from "@/types/const";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  type Database,
  DatabaseSchema$,
} from "@/types/proto-es/v1/database_service_pb";
import {
  formatEnvironmentName,
  unknownEnvironment,
} from "@/types/v1/environment";
import { unknownInstanceResource } from "@/types/v1/instance";
import type { DatabaseListParams } from "./types";

const UNKNOWN_PROJECT_NAME = `projects/${UNKNOWN_ID}`;

export function createUnknownDatabase(): Database {
  const instanceResource = unknownInstanceResource();
  return createProto(DatabaseSchema$, {
    name: `${instanceResource.name}/databases/${UNKNOWN_ID}`,
    state: State.ACTIVE,
    project: UNKNOWN_PROJECT_NAME,
    effectiveEnvironment: formatEnvironmentName(unknownEnvironment().id),
    instanceResource,
  });
}

type DatabaseAccess = {
  resetDatabases: () => void;
  getDatabaseList: () => Database[];
  getDatabaseByName: (name: string) => Database;
  getOrFetchDatabaseByName: (
    name: string,
    silent?: boolean
  ) => Promise<Database>;
  batchGetOrFetchDatabases: (names: string[]) => Promise<Database[]>;
  fetchDatabases: (params: DatabaseListParams) => Promise<{
    databases: Database[];
    nextPageToken: string;
  }>;
};

let databaseAccess: DatabaseAccess = {
  resetDatabases: () => {},
  getDatabaseList: () => [],
  getDatabaseByName: () => createUnknownDatabase(),
  getOrFetchDatabaseByName: async () => createUnknownDatabase(),
  batchGetOrFetchDatabases: async () => [],
  fetchDatabases: async () => ({ databases: [], nextPageToken: "" }),
};

export const setDatabaseAccess = (access: DatabaseAccess) => {
  databaseAccess = access;
};

export const resetDatabases = () => databaseAccess.resetDatabases();

export const getDatabaseList = () => databaseAccess.getDatabaseList();

export const getDatabaseByName = (name: string) =>
  databaseAccess.getDatabaseByName(name);

export const getOrFetchDatabaseByName = (name: string, silent?: boolean) =>
  databaseAccess.getOrFetchDatabaseByName(name, silent);

export const batchGetOrFetchDatabases = (names: string[]) =>
  databaseAccess.batchGetOrFetchDatabases(names);

export const fetchDatabases = (params: DatabaseListParams) =>
  databaseAccess.fetchDatabases(params);
