import { create as createProto } from "@bufbuild/protobuf";
import { Code, createContextValues } from "@connectrpc/connect";
import { uniq } from "lodash-es";
import { databaseServiceClientConnect } from "@/api";
import { ignoredCodesContextKey, silentContextKey } from "@/api/context-key";
import {
  BatchGetDatabasesRequestSchema,
  BatchSyncDatabasesRequestSchema,
  type BatchUpdateDatabasesRequest,
  BatchUpdateDatabasesRequestSchema,
  type Database,
  type DiffSchemaRequest,
  DiffSchemaRequestSchema,
  GetDatabaseRequestSchema,
  GetDatabaseSchemaRequestSchema,
  ListDatabasesRequestSchema,
  SyncDatabaseRequestSchema,
  type UpdateDatabaseRequest,
  UpdateDatabaseRequestSchema,
} from "@/types/proto-es/v1/database_service_pb";
import {
  type Instance,
  InstanceResourceSchema,
} from "@/types/proto-es/v1/instance_service_pb";
import { isValidDatabaseName } from "@/types/v1/database";
import { unknownInstanceResource } from "@/types/v1/instance";
import { createUnknownDatabase, setDatabaseAccess } from "./databaseAccess";
import type { AppSliceCreator, DatabaseSlice } from "./types";
import { buildDatabaseFilter, isMissingOrForbidden, toError } from "./utils";

// Inlined to keep the app store's load graph free of the Pinia `@/stores`
// barrel that `@/utils/v1/database` pulls in.
function instanceResourceNameFromDatabase(databaseName: string): string {
  const match = databaseName.match(
    /^(?:(projects\/[^/]+)\/)?instances\/([^/]+)\/databases\//
  );
  if (!match) return "";
  return match[1]
    ? `${match[1]}/instances/${match[2]}`
    : `instances/${match[2]}`;
}

export const createDatabaseSlice: AppSliceCreator<DatabaseSlice> = (
  set,
  get
) => {
  let unknownDatabase: Database | undefined;
  const getUnknownDatabase = () => {
    unknownDatabase ??= createUnknownDatabase();
    return unknownDatabase;
  };

  // Mirrors the legacy Pinia `batchComposeDatabase`: pre-caches the owning
  // projects and guarantees `instanceResource` is populated (with a fallback)
  // so consumers can read engine / instance off any database.
  const composeDatabases = async (
    databases: Database[],
    silent = false
  ): Promise<Database[]> => {
    await get().batchFetchProjects(
      databases.map((db) => db.project),
      silent
    );
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

  // Compose then immutably merge into the by-name cache; returns the composed
  // list so callers can hand it straight back to their consumers.
  const upsertDatabases = async (
    databases: Database[],
    silent = false
  ): Promise<Database[]> => {
    const composed = await composeDatabases(databases, silent);
    set((state) => {
      const next = { ...state.databasesByName };
      for (const db of composed) {
        next[db.name] = db;
      }
      return { databasesByName: next };
    });
    return composed;
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
        const [composed] = await composeDatabases([database], silent);
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

  const slice: DatabaseSlice = {
    databasesByName: {},
    databaseRequests: {},
    databaseErrorsByName: {},

    resetDatabases: () => {
      set({
        databasesByName: {},
        databaseRequests: {},
        databaseErrorsByName: {},
      });
    },

    getDatabaseList: () => Object.values(get().databasesByName),

    getDatabaseByName: (name) =>
      get().databasesByName[name] ?? getUnknownDatabase(),

    fetchDatabase: (name) => fetchByName(name, false),

    getOrFetchDatabaseByName: async (name, silent = true) =>
      (await fetchByName(name, silent)) ?? getUnknownDatabase(),

    batchFetchDatabases: async (names, silent = false) => {
      const validNames = uniq(names).filter(isValidDatabaseName);
      if (!validNames.length) return [];
      try {
        const response = await databaseServiceClientConnect.batchGetDatabases(
          createProto(BatchGetDatabasesRequestSchema, {
            parent: "-",
            names: validNames,
          }),
          {
            // A revoked name fails the whole batch; ignoring it here keeps the
            // interceptor from navigating to /403. The fallback renders the rest.
            contextValues: createContextValues()
              .set(ignoredCodesContextKey, [
                // Listing any code replaces the middleware defaults, so
                // NotFound is repeated here. Unauthenticated is left out: an
                // ignored code short-circuits before the token refresh.
                Code.NotFound,
                Code.PermissionDenied,
              ])
              .set(silentContextKey, silent),
          }
        );
        const composed = await composeDatabases(response.databases, silent);
        set((state) => {
          const next = { ...state.databasesByName };
          for (const db of composed) {
            next[db.name] = db;
          }
          return { databasesByName: next };
        });
        return composed;
      } catch (error) {
        if (!isMissingOrForbidden(error)) throw error;
        // Batch is all-or-nothing; refetch per name so one stale name isn't
        // fatal. Raw gets, composed once: composeDatabases fans out to
        // batchFetchProjects, so composing per database multiplies that call.
        const settled = await Promise.allSettled(
          validNames.map((name) =>
            databaseServiceClientConnect.getDatabase(
              createProto(GetDatabaseRequestSchema, { name }),
              {
                contextValues: createContextValues().set(
                  silentContextKey,
                  true
                ),
              }
            )
          )
        );
        // Record what did not come back, the way fetchByName does. A rejection
        // here is one database missing from an already degraded view, not a
        // reason to discard the ones that did resolve.
        set((state) => ({
          databaseErrorsByName: settled.reduce(
            (errors, result, index) =>
              result.status === "rejected"
                ? { ...errors, [validNames[index]]: toError(result.reason) }
                : errors,
            state.databaseErrorsByName
          ),
        }));
        const found = settled.flatMap((result) =>
          result.status === "fulfilled" ? [result.value] : []
        );
        return found.length > 0 ? await upsertDatabases(found, true) : [];
      }
    },

    batchGetOrFetchDatabases: async (names, silent = false) => {
      const validNames = uniq(names).filter(isValidDatabaseName);
      const pending = validNames.filter((name) => {
        const cached = get().databasesByName[name];
        return !(cached && isValidDatabaseName(cached.name));
      });
      await get().batchFetchDatabases(pending, silent);
      return validNames.map((name) => get().getDatabaseByName(name));
    },

    fetchDatabases: async ({
      parent,
      pageSize,
      pageToken,
      filter,
      orderBy,
      silent,
      skipCacheRemoval,
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
      if (
        /^(?:projects\/[^/]+\/)?instances\/[^/]+$/.test(parent) &&
        !skipCacheRemoval
      ) {
        get().removeCacheByInstance(parent);
      }
      const composed = await upsertDatabases(
        response.databases,
        silent ?? false
      );
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
        await upsertDatabases([database]);
      }
    },

    batchSyncDatabases: async (databases) => {
      await databaseServiceClientConnect.batchSyncDatabases(
        createProto(BatchSyncDatabasesRequestSchema, {
          parent: "-",
          names: databases,
        })
      );
    },

    batchUpdateDatabases: async (params: BatchUpdateDatabasesRequest) => {
      const response = await databaseServiceClientConnect.batchUpdateDatabases(
        createProto(BatchUpdateDatabasesRequestSchema, {
          parent: params.parent,
          requests: params.requests.map((req) => ({
            database: req.database,
            updateMask: req.updateMask,
          })),
        })
      );
      return upsertDatabases(response.databases);
    },

    updateDatabase: async (params: UpdateDatabaseRequest) => {
      if (!params.database) {
        throw new Error("Database is required for update");
      }
      const updated = await databaseServiceClientConnect.updateDatabase(
        createProto(UpdateDatabaseRequestSchema, {
          database: params.database,
          updateMask: params.updateMask,
        })
      );
      const [composed] = await upsertDatabases([updated]);
      return composed;
    },

    removeCacheByInstance: (instance) => {
      const names = Object.keys(get().databasesByName).filter(
        (name) => instanceResourceNameFromDatabase(name) === instance
      );
      if (!names.length) return;
      set((state) => {
        const next = { ...state.databasesByName };
        for (const name of names) {
          delete next[name];
        }
        return { databasesByName: next };
      });
      for (const name of names) {
        get().removeDatabaseMetadataCache(name);
      }
    },

    updateDatabaseInstance: (instance: Instance) => {
      set((state) => {
        const next = { ...state.databasesByName };
        for (const [name, database] of Object.entries(next)) {
          if (instanceResourceNameFromDatabase(name) !== instance.name) {
            continue;
          }
          next[name] = {
            ...database,
            instanceResource: createProto(InstanceResourceSchema, {
              name: instance.name,
              title: instance.title,
              engine: instance.engine,
              environment: instance.environment,
              activation: instance.activation,
              dataSources: [],
            }),
          };
        }
        return { databasesByName: next };
      });
    },

    fetchDatabaseSchema: (database) =>
      databaseServiceClientConnect.getDatabaseSchema(
        createProto(GetDatabaseSchemaRequestSchema, {
          name: `${database}/schema`,
        })
      ),

    diffSchema: (params: DiffSchemaRequest) =>
      databaseServiceClientConnect.diffSchema(
        createProto(DiffSchemaRequestSchema, params)
      ),
  };

  setDatabaseAccess({
    resetDatabases: slice.resetDatabases,
    getDatabaseList: slice.getDatabaseList,
    getDatabaseByName: slice.getDatabaseByName,
    getOrFetchDatabaseByName: slice.getOrFetchDatabaseByName,
    batchGetOrFetchDatabases: slice.batchGetOrFetchDatabases,
    fetchDatabases: slice.fetchDatabases,
  });

  return slice;
};
