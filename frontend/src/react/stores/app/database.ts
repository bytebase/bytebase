import { create as createProto } from "@bufbuild/protobuf";
import { databaseServiceClientConnect } from "@/connect";
import {
  BatchGetDatabasesRequestSchema,
  type Database,
  GetDatabaseRequestSchema,
  ListDatabasesRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { isValidDatabaseName } from "@/types/v1/database";
import type { AppSliceCreator, DatabaseSlice } from "./types";

function toError(error: unknown): Error {
  if (error instanceof Error) return error;
  return new Error(String(error));
}

export const createDatabaseSlice: AppSliceCreator<DatabaseSlice> = (
  set,
  get
) => ({
  databasesByName: {},
  databaseRequests: {},
  databaseErrorsByName: {},

  fetchDatabase: async (name) => {
    if (!isValidDatabaseName(name)) return undefined;
    const existing = get().databasesByName[name];
    if (existing) return existing;
    const pending = get().databaseRequests[name];
    if (pending) return pending;

    const request = databaseServiceClientConnect
      .getDatabase(createProto(GetDatabaseRequestSchema, { name }))
      .then((database: Database) => {
        set((state) => {
          const { [name]: _, ...databaseRequests } = state.databaseRequests;
          return {
            databasesByName: {
              ...state.databasesByName,
              [database.name]: database,
            },
            databaseErrorsByName: {
              ...state.databaseErrorsByName,
              [name]: undefined,
            },
            databaseRequests,
          };
        });
        return database;
      })
      .catch((error) => {
        set((state) => {
          const { [name]: _, ...databaseRequests } = state.databaseRequests;
          return {
            databaseErrorsByName: {
              ...state.databaseErrorsByName,
              [name]: toError(error),
            },
            databaseRequests,
          };
        });
        return undefined;
      });
    set((state) => ({
      databaseRequests: {
        ...state.databaseRequests,
        [name]: request,
      },
    }));
    return request;
  },

  batchFetchDatabases: async (names) => {
    if (!names.length) return [];
    const response = await databaseServiceClientConnect.batchGetDatabases(
      createProto(BatchGetDatabasesRequestSchema, { parent: "-", names })
    );
    set((state) => {
      const next = { ...state.databasesByName };
      for (const db of response.databases) {
        next[db.name] = db;
      }
      return { databasesByName: next };
    });
    return response.databases;
  },

  fetchDatabases: async ({ parent, pageSize, pageToken, filter, orderBy }) => {
    const response = await databaseServiceClientConnect.listDatabases(
      createProto(ListDatabasesRequestSchema, {
        parent,
        pageSize,
        pageToken: pageToken ?? "",
        filter: filter ?? "",
        orderBy: orderBy ?? "",
      })
    );
    set((state) => {
      const next = { ...state.databasesByName };
      for (const db of response.databases) {
        next[db.name] = db;
      }
      return { databasesByName: next };
    });
    return {
      databases: response.databases,
      nextPageToken: response.nextPageToken,
    };
  },
});
