import { create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { databaseServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { UNKNOWN_ID } from "@/types/const";
import {
  type DatabaseMetadata,
  DatabaseMetadataSchema,
  type ExtensionMetadata,
  type ExternalTableMetadata,
  ExternalTableMetadataSchema,
  type FunctionMetadata,
  GetDatabaseMetadataRequestSchema,
  type SchemaMetadata,
  type TableMetadata,
  TableMetadataSchema,
  type ViewMetadata,
  ViewMetadataSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { UNKNOWN_INSTANCE_NAME } from "@/types/v1/instance";
import { extractDatabaseResourceName } from "@/utils";
import type { AppSliceCreator, DBSchemaSlice } from "./types";

const ensureDatabaseResourceName = (name: string) =>
  extractDatabaseResourceName(name).database;

const ensureDatabaseMetadataResourceName = (name: string) =>
  `${ensureDatabaseResourceName(name)}/metadata`;

const cacheKey = (metadataName: string, filter: string, limit: number) =>
  `${metadataName}::${filter}::${limit}`;

// Module-level singleton placeholders for not-found lookups. Reusing the
// same reference keeps `useAppStore((s) => s.getXxxMetadata(...))`
// selectors stable across renders so React's `Object.is` check doesn't
// fire on every unrelated store update. The legacy Pinia store created a
// fresh proto each call, but its Vue-reactivity bridge papered over the
// resulting instability.
const EMPTY_TABLE_METADATA: TableMetadata = createProto(
  TableMetadataSchema,
  {}
);
const EMPTY_VIEW_METADATA: ViewMetadata = createProto(ViewMetadataSchema, {});
const EMPTY_EXTERNAL_TABLE_METADATA: ExternalTableMetadata = createProto(
  ExternalTableMetadataSchema,
  {}
);

// Stable empty-list singletons so `useAppStore((s) => s.getXxxList(...))`
// selectors don't return a fresh array on every call (which would trip
// Zustand's `Object.is` snapshot check and trigger React re-renders on
// every unrelated store update). Mutation in place is contractually
// forbidden — consumers must `.concat` / spread to derive new arrays.
const EMPTY_SCHEMA_LIST: SchemaMetadata[] = [];
const EMPTY_TABLE_LIST: TableMetadata[] = [];
const EMPTY_VIEW_LIST: ViewMetadata[] = [];
const EMPTY_EXTERNAL_TABLE_LIST: ExternalTableMetadata[] = [];
const EMPTY_FUNCTION_LIST: FunctionMetadata[] = [];
const EMPTY_EXTENSION_LIST: ExtensionMetadata[] = [];

// When no schema is selected the list getters flatten across all schemas.
// `flatMap` allocates a fresh array on every call, which would trip Zustand's
// `Object.is` snapshot check and loop `useSyncExternalStore` forever (the bug
// only reproduces on non-schema engines like MySQL, whose single schema is
// named "" so the flatten branch runs; Postgres selects "public" and hits the
// stable `.tables` ref). Cache the flattened result keyed by the `schemas`
// array reference — that reference is stable while the database metadata stays
// cached, so repeated calls return the same array. The cache entry is dropped
// automatically once the metadata (and its `schemas` array) is replaced or GC'd.
const flattenCache = <T>(
  selector: (schema: SchemaMetadata) => T[]
): ((schemas: SchemaMetadata[]) => T[]) => {
  const cache = new WeakMap<SchemaMetadata[], T[]>();
  return (schemas) => {
    const cached = cache.get(schemas);
    if (cached) return cached;
    const flattened = schemas.flatMap(selector);
    cache.set(schemas, flattened);
    return flattened;
  };
};

const flattenTables = flattenCache((s) => s.tables);
const flattenViews = flattenCache((s) => s.views);
const flattenExternalTables = flattenCache((s) => s.externalTables);
const flattenFunctions = flattenCache((s) => s.functions);

const isUnknownDatabase = (database: string): boolean => {
  const { databaseName, instanceName } = extractDatabaseResourceName(database);
  return (
    databaseName === String(UNKNOWN_ID) || instanceName === String(UNKNOWN_ID)
  );
};

export const createDBSchemaSlice: AppSliceCreator<DBSchemaSlice> = (
  set,
  get
) => {
  const emptyDatabaseMetadataByName = new Map<string, DatabaseMetadata>();

  const getEmptyDatabaseMetadata = (database: string) => {
    const metadataName = ensureDatabaseMetadataResourceName(database);
    const cached = emptyDatabaseMetadataByName.get(metadataName);
    if (cached) return cached;
    const metadata = createProto(DatabaseMetadataSchema, {
      name: metadataName,
    });
    emptyDatabaseMetadataByName.set(metadataName, metadata);
    return metadata;
  };

  return {
    metadataByName: {},
    metadataRequests: {},

    getDatabaseMetadata: (database) => {
      const metadataName = ensureDatabaseMetadataResourceName(database);
      const cached = get().metadataByName[cacheKey(metadataName, "", 0)];
      return cached ?? getEmptyDatabaseMetadata(database);
    },

    getCachedDatabaseMetadata: (database) => {
      const metadataName = ensureDatabaseMetadataResourceName(database);
      return get().metadataByName[cacheKey(metadataName, "", 0)];
    },

    getSchemaList: (database) => {
      return (
        get().getCachedDatabaseMetadata(database)?.schemas ?? EMPTY_SCHEMA_LIST
      );
    },

    getSchemaMetadata: ({ database, schema }) => {
      const metadata = get().getDatabaseMetadata(database);
      return metadata.schemas.find((s) => s.name === schema);
    },

    getTableList: ({ database, schema }) => {
      const schemas = get().getCachedDatabaseMetadata(database)?.schemas;
      if (!schemas) return EMPTY_TABLE_LIST;
      if (schema) {
        return (
          schemas.find((s) => s.name === schema)?.tables ?? EMPTY_TABLE_LIST
        );
      }
      return flattenTables(schemas);
    },

    getViewList: ({ database, schema }) => {
      const schemas = get().getCachedDatabaseMetadata(database)?.schemas;
      if (!schemas) return EMPTY_VIEW_LIST;
      if (schema) {
        return schemas.find((s) => s.name === schema)?.views ?? EMPTY_VIEW_LIST;
      }
      return flattenViews(schemas);
    },

    getExternalTableList: ({ database, schema }) => {
      const schemas = get().getCachedDatabaseMetadata(database)?.schemas;
      if (!schemas) return EMPTY_EXTERNAL_TABLE_LIST;
      if (schema) {
        return (
          schemas.find((s) => s.name === schema)?.externalTables ??
          EMPTY_EXTERNAL_TABLE_LIST
        );
      }
      return flattenExternalTables(schemas);
    },

    getFunctionList: ({ database, schema }) => {
      const schemas = get().getCachedDatabaseMetadata(database)?.schemas;
      if (!schemas) return EMPTY_FUNCTION_LIST;
      if (schema) {
        return (
          schemas.find((s) => s.name === schema)?.functions ??
          EMPTY_FUNCTION_LIST
        );
      }
      return flattenFunctions(schemas);
    },

    getExtensionList: (database) => {
      return (
        get().getCachedDatabaseMetadata(database)?.extensions ??
        EMPTY_EXTENSION_LIST
      );
    },

    removeDatabaseMetadataCache: (database) => {
      const metadataName = ensureDatabaseMetadataResourceName(database);
      const prefix = `${metadataName}::`;
      set((state) => {
        const nextEntries: Record<string, DatabaseMetadata> = {};
        for (const [key, value] of Object.entries(state.metadataByName)) {
          if (!key.startsWith(prefix)) nextEntries[key] = value;
        }
        const nextRequests: Record<string, Promise<DatabaseMetadata>> = {};
        for (const [key, value] of Object.entries(state.metadataRequests)) {
          if (!key.startsWith(prefix)) nextRequests[key] = value;
        }
        return {
          metadataByName: nextEntries,
          metadataRequests: nextRequests,
        };
      });
    },

    getTableMetadata: ({ database, schema, table }) => {
      const metadata = get().getDatabaseMetadata(database);
      const tables = schema
        ? (metadata.schemas.find((s) => s.name === schema)?.tables ?? [])
        : metadata.schemas.flatMap((s) => s.tables);
      return tables.find((t) => t.name === table) ?? EMPTY_TABLE_METADATA;
    },

    getExternalTableMetadata: ({ database, schema, externalTable }) => {
      const metadata = get().getDatabaseMetadata(database);
      const externalTables = schema
        ? (metadata.schemas.find((s) => s.name === schema)?.externalTables ??
          [])
        : metadata.schemas.flatMap((s) => s.externalTables);
      return (
        externalTables.find((t) => t.name === externalTable) ??
        EMPTY_EXTERNAL_TABLE_METADATA
      );
    },

    getViewMetadata: ({ database, schema, view }) => {
      const metadata = get().getDatabaseMetadata(database);
      const views = schema
        ? (metadata.schemas.find((s) => s.name === schema)?.views ?? [])
        : metadata.schemas.flatMap((s) => s.views);
      return views.find((v) => v.name === view) ?? EMPTY_VIEW_METADATA;
    },

    getOrFetchDatabaseMetadata: async (params) => {
      const {
        database,
        skipCache = false,
        silent = false,
        limit = 0,
        filter = "",
      } = params;
      if (isUnknownDatabase(database)) {
        return getEmptyDatabaseMetadata(
          `${UNKNOWN_INSTANCE_NAME}/databases/${UNKNOWN_ID}`
        );
      }

      const metadataName = ensureDatabaseMetadataResourceName(database);
      const key = cacheKey(metadataName, filter, limit);

      if (!skipCache) {
        const existing = get().metadataByName[key];
        if (existing) return existing;
        const pending = get().metadataRequests[key];
        if (pending) return pending;
      }

      const request = databaseServiceClientConnect
        .getDatabaseMetadata(
          createProto(GetDatabaseMetadataRequestSchema, {
            name: metadataName,
            limit,
            filter,
          }),
          {
            contextValues: createContextValues().set(silentContextKey, silent),
          }
        )
        .then((metadata: DatabaseMetadata) => {
          set((state) => {
            const { [key]: _, ...metadataRequests } = state.metadataRequests;
            return {
              metadataByName: {
                ...state.metadataByName,
                [key]: metadata,
              },
              metadataRequests,
            };
          });
          return metadata;
        })
        .catch((error) => {
          set((state) => {
            const { [key]: _, ...metadataRequests } = state.metadataRequests;
            return { metadataRequests };
          });
          throw error;
        });
      set((state) => ({
        metadataRequests: {
          ...state.metadataRequests,
          [key]: request,
        },
      }));
      return request;
    },
  };
};

// Re-export types so consumers don't reach into `types.ts` for narrow
// per-slice types.
export type {
  DatabaseMetadata,
  ExternalTableMetadata,
  SchemaMetadata,
  TableMetadata,
  ViewMetadata,
};
