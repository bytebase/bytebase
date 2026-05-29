import { create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { uniq } from "lodash-es";
import { databaseServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { UNKNOWN_ID } from "@/types/const";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  BatchGetDatabasesRequestSchema,
  type Database,
  DatabaseSchema$,
  GetDatabaseRequestSchema,
  ListDatabasesRequestSchema,
  SyncDatabaseRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { isValidDatabaseName } from "@/types/v1/database";
import {
  formatEnvironmentName,
  unknownEnvironment,
} from "@/types/v1/environment";
import { unknownInstanceResource } from "@/types/v1/instance";
import type { AppSliceCreator, DatabaseSlice } from "./types";
import { buildDatabaseFilter, toError } from "./utils";

const UNKNOWN_PROJECT_NAME = `projects/${UNKNOWN_ID}`;

// Inlined to keep the app store's load graph free of the Pinia `@/store`
// barrel that `@/utils/v1/database` pulls in.
function instanceResourceNameFromDatabase(databaseName: string): string {
  const match = databaseName.match(/(?:^|\/)instances\/([^/]+)\/databases\//);
  return match ? `instances/${match[1]}` : "";
}

// Mirrors the legacy Pinia `unknownDatabase`, inlined so this module does not
// import `@/types/v1/project` (which pulls the Pinia actuator store).
function createUnknownDatabase(): Database {
  const instanceResource = unknownInstanceResource();
  return createProto(DatabaseSchema$, {
    name: `${instanceResource.name}/databases/${UNKNOWN_ID}`,
    state: State.ACTIVE,
    project: UNKNOWN_PROJECT_NAME,
    effectiveEnvironment: formatEnvironmentName(unknownEnvironment().id),
    instanceResource,
  });
}

export const createDatabaseSlice: AppSliceCreator<DatabaseSlice> = (
  set,
  get
) => {
  // Mirrors the legacy Pinia `batchComposeDatabase`: pre-caches the owning
  // projects and guarantees `instanceResource` is populated (with a fallback)
  // so consumers can read engine / instance off any database.
  const composeDatabases = async (
    databases: Database[]
  ): Promise<Database[]> => {
    await get().batchFetchProjects(databases.map((db) => db.project));
    for (const database of databases) {
      if (!database.instanceResource) {
        database.instanceResource = {
          ...unknownInstanceResource(),
          name: instanceResourceNameFromDatabase(database.name),
        };
      }
    }
    return databases;
  };

  const fetchByName = async (
    name: string,
    silent: boolean
  ): Promise<Database | undefined> => {
    if (!isValidDatabaseName(name)) return undefined;
    const existing = get().databasesByName[name];
    if (existing) return existing;
    const pending = get().databaseRequests[name];
    if (pending) return pending;

    const request = databaseServiceClientConnect
      .getDatabase(createProto(GetDatabaseRequestSchema, { name }), {
        contextValues: createContextValues().set(silentContextKey, silent),
      })
      .then(async (database: Database) => {
        const [composed] = await composeDatabases([database]);
        set((state) => {
          const { [name]: _, ...databaseRequests } = state.databaseRequests;
          return {
            databasesByName: {
              ...state.databasesByName,
              [composed.name]: composed,
            },
            databaseErrorsByName: {
              ...state.databaseErrorsByName,
              [name]: undefined,
            },
            databaseRequests,
          };
        });
        return composed;
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
  };

  return {
    databasesByName: {},
    databaseRequests: {},
    databaseErrorsByName: {},

    getDatabaseByName: (name) =>
      get().databasesByName[name] ?? createUnknownDatabase(),

    fetchDatabase: (name) => fetchByName(name, false),

    getOrFetchDatabaseByName: async (name, silent = true) =>
      (await fetchByName(name, silent)) ?? createUnknownDatabase(),

    batchFetchDatabases: async (names) => {
      const validNames = uniq(names).filter(isValidDatabaseName);
      if (!validNames.length) return [];
      const response = await databaseServiceClientConnect.batchGetDatabases(
        createProto(BatchGetDatabasesRequestSchema, {
          parent: "-",
          names: validNames,
        })
      );
      const composed = await composeDatabases(response.databases);
      set((state) => {
        const next = { ...state.databasesByName };
        for (const db of composed) {
          next[db.name] = db;
        }
        return { databasesByName: next };
      });
      return composed;
    },

    batchGetOrFetchDatabases: async (names) => {
      const validNames = uniq(names).filter(isValidDatabaseName);
      const pending = validNames.filter((name) => {
        const cached = get().databasesByName[name];
        return !(cached && isValidDatabaseName(cached.name));
      });
      await get().batchFetchDatabases(pending);
      return validNames.map((name) => get().getDatabaseByName(name));
    },

    fetchDatabases: async ({
      parent,
      pageSize,
      pageToken,
      filter,
      orderBy,
      silent,
    }) => {
      const filterString =
        typeof filter === "string"
          ? filter
          : filter
            ? buildDatabaseFilter(filter)
            : "";
      const showDeleted =
        typeof filter === "object" ? filter?.showDeleted : undefined;
      const response = await databaseServiceClientConnect.listDatabases(
        createProto(ListDatabasesRequestSchema, {
          parent,
          pageSize,
          pageToken: pageToken ?? "",
          filter: filterString,
          orderBy: orderBy ?? "",
          showDeleted,
        }),
        {
          contextValues: createContextValues().set(
            silentContextKey,
            silent ?? false
          ),
        }
      );
      const composed = await composeDatabases(response.databases);
      set((state) => {
        const next = { ...state.databasesByName };
        for (const db of composed) {
          next[db.name] = db;
        }
        return { databasesByName: next };
      });
      return {
        databases: composed,
        nextPageToken: response.nextPageToken,
      };
    },

    syncDatabase: async (name, refresh = false) => {
      await databaseServiceClientConnect.syncDatabase(
        createProto(SyncDatabaseRequestSchema, { name })
      );
      if (refresh) {
        const database = await databaseServiceClientConnect.getDatabase(
          createProto(GetDatabaseRequestSchema, { name })
        );
        const [composed] = await composeDatabases([database]);
        set((state) => ({
          databasesByName: {
            ...state.databasesByName,
            [composed.name]: composed,
          },
        }));
      }
    },
  };
};
