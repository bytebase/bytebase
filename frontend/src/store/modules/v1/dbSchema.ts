import { defineStore } from "pinia";
import { computed, unref, watchEffect } from "vue";
import { databaseServiceClient } from "@/grpcweb";
import { useCache } from "@/store/cache";
import { MaybeRef, UNKNOWN_ID, EMPTY_ID, UNKNOWN_INSTANCE_NAME } from "@/types";
import {
  DatabaseMetadata,
  TableMetadata,
  DatabaseMetadataView,
} from "@/types/proto/v1/database_service";
import { extractDatabaseResourceName } from "@/utils";

const VIEW_FULL = DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL;
const VIEW_BASIC = DatabaseMetadataView.DATABASE_METADATA_VIEW_BASIC;

type DatabaseMetadataCacheKey = [
  string /* database metadata resource name */,
  view: DatabaseMetadataView
];
type TableMetadataCacheKey = [
  string /* database metadata resource name */,
  string /* schema */,
  string /* table */
];

export const useDBSchemaV1Store = defineStore("dbSchema_v1", () => {
  const cacheByName = useCache<DatabaseMetadataCacheKey, DatabaseMetadata>(
    "bb.db-schema.by-name"
  );
  const tableMetadataCacheByName = useCache<
    TableMetadataCacheKey,
    TableMetadata | undefined
  >("bb.db-schema.table-metadata-by-name");

  const getCache = (
    name: string,
    view?: DatabaseMetadataView
  ): [DatabaseMetadata | undefined, DatabaseMetadataView | undefined] => {
    const metadataResourceName = ensureDatabaseMetadataResourceName(name);
    if (view === undefined) {
      const full = cacheByName.getEntity([metadataResourceName, VIEW_FULL]);
      if (full) {
        return [full, VIEW_FULL];
      }
      const basic = cacheByName.getEntity([metadataResourceName, VIEW_BASIC]);
      if (basic) {
        return [basic, VIEW_BASIC];
      }
      return [undefined, undefined];
    }
    const entity = cacheByName.getEntity([metadataResourceName, view]);
    return [entity, view];
  };
  const getRequestCache = (
    name: string,
    view?: DatabaseMetadataView
  ): [
    Promise<DatabaseMetadata> | undefined,
    DatabaseMetadataView | undefined
  ] => {
    const metadataResourceName = ensureDatabaseMetadataResourceName(name);
    if (view === undefined) {
      const full = cacheByName.getRequest([metadataResourceName, VIEW_FULL]);
      if (full) {
        return [full, VIEW_FULL];
      }
      const basic = cacheByName.getRequest([metadataResourceName, VIEW_BASIC]);
      if (basic) {
        return [basic, VIEW_BASIC];
      }
      return [undefined, undefined];
    }
    const entity = cacheByName.getRequest([metadataResourceName, view]);
    return [entity, view];
  };
  const setCache = (metadata: DatabaseMetadata, view: DatabaseMetadataView) => {
    if (view === VIEW_FULL) {
      cacheByName.invalidateEntity([metadata.name, VIEW_BASIC]);
    }
    cacheByName.setEntity([metadata.name, view], metadata);
    return metadata;
  };
  const setRequestCache = (
    name: string,
    view: DatabaseMetadataView,
    promise: Promise<DatabaseMetadata>
  ) => {
    const metadataResourceName = ensureDatabaseMetadataResourceName(name);
    if (view === VIEW_FULL) {
      cacheByName.invalidateRequest([metadataResourceName, VIEW_BASIC]);
    }
    cacheByName.setRequest([metadataResourceName, view], promise);
  };
  const mergeCache = (
    metadata: DatabaseMetadata,
    view?: DatabaseMetadataView
  ) => {
    const [existed, existedView] = getCache(metadata.name);
    if (!existed || view === VIEW_FULL) {
      return setCache(metadata, view ?? VIEW_BASIC);
    }
    console.assert(existedView !== undefined);
    if (!existedView) {
      throw new Error(
        `should never reach this line. metadata=${metadata}, view=${view}, existed=${existed}, existedView=${existedView}`
      );
    }

    for (const schemaConfig of metadata.schemaConfigs) {
      const schemaConfigIndex = existed.schemaConfigs.findIndex(
        (s) => s.name === schemaConfig.name
      );
      if (schemaConfigIndex < 0) {
        existed.schemaConfigs.push(schemaConfig);
        continue;
      }
      for (const tableConfig of schemaConfig.tableConfigs) {
        const tableIndex = existed.schemaConfigs[
          schemaConfigIndex
        ].tableConfigs.findIndex((t) => t.name === tableConfig.name);
        if (tableIndex < 0) {
          existed.schemaConfigs[schemaConfigIndex].tableConfigs.push(
            tableConfig
          );
        } else {
          existed.schemaConfigs[schemaConfigIndex].tableConfigs[tableIndex] =
            tableConfig;
        }
      }
    }

    for (const schema of metadata.schemas) {
      const schemaIndex = existed.schemas.findIndex(
        (s) => s.name === schema.name
      );
      if (schemaIndex < 0) {
        existed.schemas.push(schema);
        continue;
      }
      for (const table of schema.tables) {
        const tableIndex = existed.schemas[schemaIndex].tables.findIndex(
          (t) => t.name === table.name
        );
        if (tableIndex < 0) {
          existed.schemas[schemaIndex].tables.push(table);
        } else {
          existed.schemas[schemaIndex].tables[tableIndex] = table;
        }
      }
    }
    return setCache(existed, view ?? existedView);
  };
  const updateDatabaseSchemaConfigs = async (metadata: DatabaseMetadata) => {
    await databaseServiceClient.updateDatabaseMetadata({
      databaseMetadata: metadata,
      updateMask: ["schema_configs"],
    });
    // updateDatabaseMetadata actually returns basic view
    // so we cannot setCache(updated) here
    mergeCache(metadata);
  };
  /**
   *
   * @param database
   * @param view if not specified, try find full -> basic
   * @returns
   */
  const getDatabaseMetadata = (
    database: string,
    view?: DatabaseMetadataView
  ) => {
    return (
      getCache(database, view)[0] ??
      DatabaseMetadata.fromPartial({
        name: ensureDatabaseMetadataResourceName(database),
      })
    );
  };
  /**
   * @param params if view is not specified, try use cache (full -> basic), use basic if cache missed
   * @returns
   */
  const getOrFetchDatabaseMetadata = async (params: {
    database: string;
    view?: DatabaseMetadataView;
    filter?: string;
    skipCache?: boolean;
    silent?: boolean;
  }) => {
    const {
      database,
      view,
      filter = "",
      skipCache = false,
      silent = false,
    } = params;
    const { database: databaseName } = extractDatabaseResourceName(database);
    if (
      databaseName === String(UNKNOWN_ID) ||
      databaseName === String(EMPTY_ID)
    ) {
      return DatabaseMetadata.fromJSON({
        name: ensureDatabaseMetadataResourceName(
          `${UNKNOWN_INSTANCE_NAME}/databases/${UNKNOWN_ID}`
        ),
      });
    }

    const metadataResourceName = ensureDatabaseMetadataResourceName(database);

    if (!skipCache) {
      const [existed] = getCache(database, view);
      if (existed) {
        // The metadata entity is stored in local dictionary.
        return existed;
      }

      const [cachedRequest] = getRequestCache(metadataResourceName, view);
      if (cachedRequest) {
        // The request was sent but still not returned.
        // We won't create a duplicated request.
        return cachedRequest;
      }
    }

    // Send a request and cache it.
    console.debug("[getOrFetchDatabaseMetadata]", {
      name: metadataResourceName,
      filter,
      view: view ?? VIEW_BASIC,
    });
    const promise = databaseServiceClient.getDatabaseMetadata(
      {
        name: metadataResourceName,
        filter,
        view: view ?? VIEW_BASIC,
      },
      {
        silent,
      }
    );
    setRequestCache(metadataResourceName, view ?? VIEW_BASIC, promise);
    promise.then((res) => mergeCache(res, view));

    return promise;
  };
  const getSchemaList = (database: string) => {
    return getCache(database)[0]?.schemas ?? [];
  };
  const getTableList = (database: string, schema?: string) => {
    const [databaseMetadata] = getCache(database);
    if (!databaseMetadata) {
      return [];
    }

    if (schema) {
      return (
        databaseMetadata.schemas.find((s) => s.name === schema)?.tables ?? []
      );
    }

    return databaseMetadata.schemas.flatMap((s) => s.tables);
  };
  const getOrFetchTableList = async (database: string, schema?: string) => {
    if (!getCache(database)[0]) {
      await getOrFetchDatabaseMetadata({ database, view: VIEW_BASIC });
    }
    return getTableList(database, schema);
  };
  const getTableByName = (database: string, table: string, schema?: string) => {
    const [databaseMetadata] = getCache(database);
    if (!databaseMetadata) {
      return undefined;
    }

    const tableList = getTableList(database, schema);
    return tableList.find((t) => t.name === table);
  };
  const getOrFetchTableMetadata = async ({
    database,
    schema,
    table,
    skipCache = false,
    silent = false,
  }: {
    database: string;
    schema: string;
    table: string;
    skipCache?: boolean;
    silent?: boolean;
  }) => {
    const metadataResourceName = ensureDatabaseMetadataResourceName(database);

    if (!skipCache) {
      const existedTable = getTableByName(database, table);
      if (existedTable && existedTable.columns.length > 0) {
        return existedTable;
      }

      const existedRequest = tableMetadataCacheByName.getRequest([
        metadataResourceName,
        schema,
        table,
      ]);
      if (existedRequest) {
        return existedRequest;
      }
    }

    console.debug("[getOrFetchDatabaseMetadata]", {
      name: metadataResourceName,
      filter: `schemas/${schema || "-"}/tables/${table}`,
      view: DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL,
    });
    const promise = databaseServiceClient
      .getDatabaseMetadata(
        {
          name: metadataResourceName,
          filter: `schemas/${schema || "-"}/tables/${table}`,
          view: DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL,
        },
        {
          silent,
        }
      )
      .then((res) => mergeCache(res))
      .then((res) => {
        return res.schemas
          .find((s) => s.name === schema)
          ?.tables.find((t) => t.name === table);
      });
    tableMetadataCacheByName.setRequest(
      [metadataResourceName, schema, table],
      promise
    );
    return promise;
  };
  const getExternalTableList = (database: string, schema?: string) => {
    const [databaseMetadata] = getCache(database);
    if (!databaseMetadata) {
      return [];
    }

    if (schema) {
      return (
        databaseMetadata.schemas.find((s) => s.name === schema)
          ?.externalTables ?? []
      );
    }

    return databaseMetadata.schemas.flatMap((s) => s.externalTables);
  };
  const getOrFetchExternalTableList = async (
    database: string,
    schema?: string
  ) => {
    if (!getCache(database)[0]) {
      await getOrFetchDatabaseMetadata({ database, view: VIEW_BASIC });
    }
    return getExternalTableList(database, schema);
  };
  const getViewList = (database: string, schema?: string) => {
    const [databaseMetadata] = getCache(database);
    if (!databaseMetadata) {
      return [];
    }

    if (schema) {
      return (
        databaseMetadata.schemas.find((s) => s.name === schema)?.views ?? []
      );
    }

    return databaseMetadata.schemas.flatMap((s) => s.views);
  };
  const getOrFetchViewList = async (database: string, schema?: string) => {
    if (!getCache(database)[0]) {
      await getOrFetchDatabaseMetadata({ database, view: VIEW_BASIC });
    }
    return getViewList(database, schema);
  };
  const getExtensionList = (database: string) => {
    const [databaseMetadata] = getCache(database);
    if (!databaseMetadata) {
      return [];
    }

    return databaseMetadata.extensions;
  };
  const getOrFetchExtensionList = async (database: string) => {
    if (!getCache(database)[0]) {
      await getOrFetchDatabaseMetadata({ database, view: VIEW_BASIC });
    }
    return getExtensionList(database);
  };
  const getFunctionList = (database: string, schema?: string) => {
    const [databaseMetadata] = getCache(database);
    if (!databaseMetadata) {
      return [];
    }
    if (schema) {
      return (
        databaseMetadata.schemas.find((s) => s.name === schema)?.functions ?? []
      );
    }
    return databaseMetadata.schemas.flatMap((s) => s.functions);
  };
  const getOrFetchFunctionList = async (database: string, schema?: string) => {
    if (!getCache(database)[0]) {
      await getOrFetchDatabaseMetadata({ database, view: VIEW_BASIC });
    }
    return getFunctionList(database, schema);
  };

  const removeCache = (name: string) => {
    const metadataResourceName = ensureDatabaseMetadataResourceName(name);
    cacheByName.invalidateEntity([metadataResourceName, VIEW_FULL]);
    cacheByName.invalidateEntity([metadataResourceName, VIEW_BASIC]);
    Array.from(tableMetadataCacheByName.entityCacheMap.values()).forEach(
      (cache) => {
        if (cache.keys[0] === metadataResourceName) {
          tableMetadataCacheByName.invalidateEntity(cache.keys);
        }
      }
    );
    Array.from(tableMetadataCacheByName.requestCacheMap.values()).forEach(
      (cache) => {
        if (cache.keys[0] === metadataResourceName) {
          tableMetadataCacheByName.invalidateRequest(cache.keys);
        }
      }
    );
  };

  return {
    updateDatabaseSchemaConfigs,
    getDatabaseMetadata,
    getOrFetchDatabaseMetadata,
    getSchemaList,
    getTableList,
    getOrFetchTableList,
    getTableByName,
    getOrFetchTableMetadata,
    getExternalTableList,
    getOrFetchExternalTableList,
    getViewList,
    getOrFetchViewList,
    getExtensionList,
    getOrFetchExtensionList,
    getFunctionList,
    getOrFetchFunctionList,
    getCache,
    removeCache,
  };
});

const ensureDatabaseResourceName = (name: string) => {
  const { instance, database } = extractDatabaseResourceName(name);
  return `instances/${instance}/databases/${database}`;
};
const ensureDatabaseMetadataResourceName = (name: string) => {
  const database = ensureDatabaseResourceName(name);
  return `${database}/metadata`;
};

export const useMetadata = (
  database: MaybeRef<string>,
  skipCache: MaybeRef<boolean>,
  view: MaybeRef<DatabaseMetadataView>
) => {
  const store = useDBSchemaV1Store();
  watchEffect(() => {
    const { database: databaseName } = extractDatabaseResourceName(
      unref(database)
    );
    if (
      databaseName === String(UNKNOWN_ID) ||
      databaseName === String(EMPTY_ID)
    ) {
      return;
    }
    store.getOrFetchDatabaseMetadata({
      database: unref(database),
      skipCache: unref(skipCache),
      view: unref(view),
    });
  });
  return computed(() =>
    store.getDatabaseMetadata(unref(database), unref(view))
  );
};
