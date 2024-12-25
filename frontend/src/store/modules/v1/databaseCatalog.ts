import { defineStore } from "pinia";
import { computed, unref, watchEffect } from "vue";
import { databaseCatalogServiceClient } from "@/grpcweb";
import { useCache } from "@/store/cache";
import type { MaybeRef } from "@/types";
import { UNKNOWN_ID, EMPTY_ID, UNKNOWN_INSTANCE_NAME } from "@/types";
import {
  ColumnCatalog,
  DatabaseCatalog,
  TableCatalog,
} from "@/types/proto/v1/database_catalog_service";
import { extractDatabaseResourceName } from "@/utils";

type DatabaseCatalogCacheKey = [string /* database catalog resource name */];

export const useDatabaseCatalogV1Store = defineStore("databaseCatalog_v1", () => {
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
    const promise = databaseCatalogServiceClient.getDatabaseCatalog(
      {
        name: catalogResourceName,
      },
      {
        silent,
      }
    );
    promise.then((res) => {
      setCache(res);
    });

    return promise;
  };

  const updateDatabaseCatalog = async (catalog: DatabaseCatalog) => {
    const updated = await databaseCatalogServiceClient.updateDatabaseCatalog({
      catalog,
    });
    setCache(updated);
    return updated;
  };

  const getDatabaseCatalog = (database: string) => {
    const { databaseName } = extractDatabaseResourceName(database);
    if (databaseName === String(UNKNOWN_ID) || databaseName === String(EMPTY_ID)) {
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
});

const ensureDatabaseResourceName = (name: string) => {
  return extractDatabaseResourceName(name).database;
};
const ensureDatabaseCatalogResourceName = (name: string) => {
  const database = ensureDatabaseResourceName(name);
  return `${database}/catalog`;
};

export const getTableCatalog = (catalog: DatabaseCatalog, schema: string, table: string) => {
  const schemaCatalog = catalog.schemas.find((s) => s.name === schema)
  return schemaCatalog?.tables.find((t) => t.name === table) ?? TableCatalog.fromPartial({
    name: table,
  });
};

export const getColumnCatalog = (catalog: DatabaseCatalog, schema: string, table: string, column: string) => {
  const schemaConfig = catalog.schemas.find((s) => s.name === schema)
  const tableCatalog = schemaConfig?.tables.find((t) => t.name === table) ?? TableCatalog.fromPartial({});
  return tableCatalog.columns?.columns.find((c) => c.name === column) ?? ColumnCatalog.fromPartial({
    name: column,
  });
};

export const useDatabaseCatalog = (
  database: MaybeRef<string>,
  skipCache: MaybeRef<boolean>
) => {
  const store = useDatabaseCatalogV1Store();
  watchEffect(() => {
    const { databaseName } = extractDatabaseResourceName(unref(database));
    if (
      databaseName === String(UNKNOWN_ID) ||
      databaseName === String(EMPTY_ID)
    ) {
      return;
    }
    store.getOrFetchDatabaseCatalog({
      database: unref(database),
      skipCache: unref(skipCache),
    });
  });
  return computed(() =>
    store.getDatabaseCatalog(unref(database))
  );
}