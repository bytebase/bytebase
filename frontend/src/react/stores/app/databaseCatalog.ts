import { create as createProto } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { databaseCatalogServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
import { UNKNOWN_ID, UNKNOWN_INSTANCE_NAME } from "@/types";
import {
  ColumnCatalogSchema,
  type DatabaseCatalog,
  DatabaseCatalogSchema,
  GetDatabaseCatalogRequestSchema,
  TableCatalog_ColumnsSchema,
  TableCatalogSchema,
  UpdateDatabaseCatalogRequestSchema,
} from "@/types/proto-es/v1/database_catalog_service_pb";
import { extractDatabaseResourceName } from "@/utils";
import type { AppSliceCreator, DatabaseCatalogSlice } from "./types";

// Catalog resource name is `{database}/catalog`.
export const catalogResourceName = (database: string): string =>
  `${extractDatabaseResourceName(database).database}/catalog`;

export const emptyDatabaseCatalog = (database: string): DatabaseCatalog =>
  createProto(DatabaseCatalogSchema, {
    name: catalogResourceName(database),
    schemas: [],
  });

const isUnknownDatabase = (database: string): boolean =>
  extractDatabaseResourceName(database).databaseName === String(UNKNOWN_ID);

const UNKNOWN_DATABASE = `${UNKNOWN_INSTANCE_NAME}/databases/${UNKNOWN_ID}`;

export const createDatabaseCatalogSlice: AppSliceCreator<
  DatabaseCatalogSlice
> = (set, get) => ({
  catalogsByName: {},
  catalogRequests: {},

  getDatabaseCatalog: (database) => {
    if (isUnknownDatabase(database)) {
      return emptyDatabaseCatalog(UNKNOWN_DATABASE);
    }
    return (
      get().catalogsByName[catalogResourceName(database)] ??
      emptyDatabaseCatalog(database)
    );
  },

  getOrFetchDatabaseCatalog: async ({
    database,
    skipCache = false,
    silent = false,
  }) => {
    if (isUnknownDatabase(database)) {
      return emptyDatabaseCatalog(UNKNOWN_DATABASE);
    }
    const key = catalogResourceName(database);
    if (!skipCache) {
      const cached = get().catalogsByName[key];
      if (cached) return cached;
      const pending = get().catalogRequests[key];
      if (pending) return pending;
    }
    const request = databaseCatalogServiceClientConnect
      .getDatabaseCatalog(
        createProto(GetDatabaseCatalogRequestSchema, { name: key }),
        { contextValues: createContextValues().set(silentContextKey, silent) }
      )
      .then((res) => {
        set((state) => {
          const { [key]: _, ...catalogRequests } = state.catalogRequests;
          return {
            catalogsByName: { ...state.catalogsByName, [res.name]: res },
            catalogRequests,
          };
        });
        return res;
      })
      .catch((err) => {
        set((state) => {
          const { [key]: _, ...catalogRequests } = state.catalogRequests;
          return { catalogRequests };
        });
        throw err;
      });
    set((state) => ({
      catalogRequests: { ...state.catalogRequests, [key]: request },
    }));
    return request;
  },

  updateDatabaseCatalog: async (catalog) => {
    const database = extractDatabaseResourceName(catalog.name).database;
    const validCatalog = createProto(DatabaseCatalogSchema, {
      name: catalog.name,
      schemas: [],
    });
    // Drop schemas that no longer exist in the database metadata, but keep
    // them all if metadata can't be loaded (e.g. missing permission) —
    // mirrors the legacy Pinia behavior.
    let metadata;
    try {
      metadata = await get().getOrFetchDatabaseMetadata({ database });
    } catch {
      metadata = undefined;
    }
    for (const schema of catalog.schemas) {
      if (!metadata || metadata.schemas.some((s) => s.name === schema.name)) {
        validCatalog.schemas.push(schema);
      }
    }
    const response =
      await databaseCatalogServiceClientConnect.updateDatabaseCatalog(
        createProto(UpdateDatabaseCatalogRequestSchema, {
          catalog: validCatalog,
        })
      );
    set((state) => ({
      catalogsByName: { ...state.catalogsByName, [response.name]: response },
    }));
    return response;
  },
});

export const getTableCatalog = (
  catalog: DatabaseCatalog,
  schema: string,
  table: string
) => {
  const schemaCatalog = catalog.schemas.find((s) => s.name === schema);
  return (
    schemaCatalog?.tables.find((t) => t.name === table) ??
    createProto(TableCatalogSchema, {
      name: table,
      kind: {
        case: "columns",
        value: createProto(TableCatalog_ColumnsSchema, {}),
      },
    })
  );
};

export const getColumnCatalog = (
  catalog: DatabaseCatalog,
  schema: string,
  table: string,
  column: string
) => {
  const tableCatalog = getTableCatalog(catalog, schema, table);
  const columns =
    tableCatalog.kind?.case === "columns"
      ? tableCatalog.kind.value.columns
      : [];
  return (
    columns.find((c) => c.name === column) ??
    createProto(ColumnCatalogSchema, { name: column })
  );
};
