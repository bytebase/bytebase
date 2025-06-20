import { defineStore } from "pinia";
import { computed, unref, watchEffect } from "vue";
import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { databaseCatalogServiceClientConnect } from "@/grpcweb";
import { silentContextKey } from "@/grpcweb/context-key";
import { useCache } from "@/store/cache";
import type { MaybeRef } from "@/types";
import { UNKNOWN_ID, EMPTY_ID, UNKNOWN_INSTANCE_NAME } from "@/types";
import {
  ColumnCatalog,
  DatabaseCatalog,
  TableCatalog,
  TableCatalog_Columns,
} from "@/types/proto/v1/database_catalog_service";
import { 
  GetDatabaseCatalogRequestSchema,
  UpdateDatabaseCatalogRequestSchema
} from "@/types/proto-es/v1/database_catalog_service_pb";
import { 
  convertNewDatabaseCatalogToOld,
  convertOldDatabaseCatalogToNew 
} from "@/utils/v1/database-catalog-conversions";
import { extractDatabaseResourceName, hasProjectPermissionV2 } from "@/utils";
import { useDatabaseV1Store } from "./database";

type DatabaseCatalogCacheKey = [string /* database catalog resource name */];

export const useDatabaseCatalogV1Store = defineStore(
  "databaseCatalog_v1",
  () => {
    const cacheByName = useCache<DatabaseCatalogCacheKey, DatabaseCatalog>(
      "bb.database-catalog.by-name"
    );

    const getCache = (name: string): DatabaseCatalog | undefined => {
      const catalogResourceName = ensureDatabaseCatalogResourceName(name);
      return cacheByName.getEntity([catalogResourceName]);
    };

    const setCache = (catalog: DatabaseCatalog) => {
      cacheByName.setEntity([catalog.name], catalog);
      return catalog;
    };

    const getOrFetchDatabaseCatalog = async (params: {
      database: string;
      skipCache?: boolean;
      silent?: boolean;
    }) => {
      const { database, skipCache = false, silent = false } = params;
      const { databaseName } = extractDatabaseResourceName(database);
      if (
        databaseName === String(UNKNOWN_ID) ||
        databaseName === String(EMPTY_ID)
      ) {
        return DatabaseCatalog.fromPartial({
          name: ensureDatabaseCatalogResourceName(
            `${UNKNOWN_INSTANCE_NAME}/databases/${UNKNOWN_ID}`
          ),
        });
      }

      const catalogResourceName = ensureDatabaseCatalogResourceName(database);

      if (!skipCache) {
        const existed = getCache(database);
        if (existed) {
          return existed;
        }
      }

      console.debug("[getOrFetchDatabaseCatalog]", {
        name: catalogResourceName,
      });
      const request = create(GetDatabaseCatalogRequestSchema, {
        name: catalogResourceName,
      });
      const promise = databaseCatalogServiceClientConnect.getDatabaseCatalog(request, {
        contextValues: createContextValues().set(silentContextKey, silent),
      }).then((res) => {
        const oldCatalog = convertNewDatabaseCatalogToOld(res);
        setCache(oldCatalog);
        return oldCatalog;
      });

      return promise;
    };

    const updateDatabaseCatalog = async (catalog: DatabaseCatalog) => {
      const newCatalog = convertOldDatabaseCatalogToNew(catalog);
      const request = create(UpdateDatabaseCatalogRequestSchema, {
        catalog: newCatalog,
      });
      const response = await databaseCatalogServiceClientConnect.updateDatabaseCatalog(request);
      const updated = convertNewDatabaseCatalogToOld(response);
      setCache(updated);
      return updated;
    };

    const getDatabaseCatalog = (database: string) => {
      const { databaseName } = extractDatabaseResourceName(database);
      if (
        databaseName === String(UNKNOWN_ID) ||
        databaseName === String(EMPTY_ID)
      ) {
        return DatabaseCatalog.fromPartial({
          name: ensureDatabaseCatalogResourceName(
            `${UNKNOWN_INSTANCE_NAME}/databases/${UNKNOWN_ID}`
          ),
        });
      }

      return (
        getCache(database) ??
        DatabaseCatalog.fromPartial({
          name: ensureDatabaseCatalogResourceName(database),
          schemas: [],
        })
      );
    };

    const removeCache = (name: string) => {
      const catalogResourceName = ensureDatabaseCatalogResourceName(name);
      cacheByName.invalidateEntity([catalogResourceName]);
    };

    return {
      getOrFetchDatabaseCatalog,
      updateDatabaseCatalog,
      getDatabaseCatalog,
      removeCache,
    };
  }
);

const ensureDatabaseResourceName = (name: string) => {
  return extractDatabaseResourceName(name).database;
};
const ensureDatabaseCatalogResourceName = (name: string) => {
  const database = ensureDatabaseResourceName(name);
  return `${database}/catalog`;
};

export const getTableCatalog = (
  catalog: DatabaseCatalog,
  schema: string,
  table: string
) => {
  const schemaCatalog = catalog.schemas.find((s) => s.name === schema);
  return (
    schemaCatalog?.tables.find((t) => t.name === table) ??
    TableCatalog.fromPartial({
      name: table,
      columns: TableCatalog_Columns.fromPartial({}),
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
  return (
    tableCatalog.columns?.columns.find((c) => c.name === column) ??
    ColumnCatalog.fromPartial({
      name: column,
    })
  );
};

export const useDatabaseCatalog = (
  database: MaybeRef<string>,
  skipCache: MaybeRef<boolean>
) => {
  const store = useDatabaseCatalogV1Store();
  const databaseStore = useDatabaseV1Store();

  watchEffect(async () => {
    const { databaseName } = extractDatabaseResourceName(unref(database));
    if (
      databaseName === String(UNKNOWN_ID) ||
      databaseName === String(EMPTY_ID)
    ) {
      return;
    }
    const db = await databaseStore.getOrFetchDatabaseByName(unref(database));
    if (hasProjectPermissionV2(db.projectEntity, "bb.databaseCatalogs.get")) {
      await store.getOrFetchDatabaseCatalog({
        database: unref(database),
        skipCache: unref(skipCache),
      });
    }
  });
  return computed(() => store.getDatabaseCatalog(unref(database)));
};
