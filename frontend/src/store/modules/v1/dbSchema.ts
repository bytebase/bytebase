import { defineStore } from "pinia";
import { computed, unref, watchEffect } from "vue";
import { databaseServiceClient } from "@/grpcweb";
import { useCache } from "@/store/cache";
import type { MaybeRef } from "@/types";
import { UNKNOWN_ID, EMPTY_ID, UNKNOWN_INSTANCE_NAME } from "@/types";
import {
  TableMetadata,
  DatabaseMetadata,
  DatabaseMetadataView,
} from "@/types/proto/v1/database_service";
import { extractDatabaseResourceName } from "@/utils";

const VIEW_FULL = DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL;
const VIEW_BASIC = DatabaseMetadataView.DATABASE_METADATA_VIEW_BASIC;

type DatabaseMetadataCacheKey = [
  string /* database metadata resource name */,
  view: DatabaseMetadataView,
];
type TableMetadataCacheKey = [
  string /* database metadata resource name */,
  string /* schema */,
  string /* table */,
];

export const useDBSchemaV1Store = defineStore("dbSchema_v1", () => {
  const cacheByName = useCache<DatabaseMetadataCacheKey, DatabaseMetadata>(
    "bb.db-schema.by-name"
  );
  const tableRequestCacheByName = useCache<
    TableMetadataCacheKey,
    TableMetadata | undefined
  >("bb.db-schema.table-metadata-by-name");

  // getCache try use cache (full -> basic). VIEW_FULL is always the source of truth.
  const getCache = (
    name: string,
    view?: DatabaseMetadataView
  ): [DatabaseMetadata | undefined, DatabaseMetadataView | undefined] => {
    const metadataResourceName = ensureDatabaseMetadataResourceName(name);

    const full = cacheByName.getEntity([metadataResourceName, VIEW_FULL]);
    if (full) {
      return [full, VIEW_FULL];
    }

    if (view && view !== VIEW_BASIC) {
      // want FULL but only found BASIC
      return [undefined, view];
    }

    const entity = cacheByName.getEntity([metadataResourceName, VIEW_BASIC]);
    return [entity, VIEW_BASIC];
  };
  const getRequestCache = (
    name: string,
    view: DatabaseMetadataView
  ): [
    Promise<DatabaseMetadata> | undefined,
    DatabaseMetadataView | undefined,
  ] => {
    const metadataResourceName = ensureDatabaseMetadataResourceName(name);
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
    view: DatabaseMetadataView = VIEW_BASIC,
    dropIfNotExist: boolean = false
  ) => {
    const [existed, existedView] = getCache(metadata.name, view);
    if (!existed || view === VIEW_FULL) {
      return setCache(metadata, view);
    }
    if (!existedView) {
      throw new Error(
        `should never reach this line. metadata=${metadata}, view=${view}, existed=${existed}, existedView=${existedView}`
      );
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

      dropCacheIfNotExist(
        dropIfNotExist,
        schema.tables,
        existed.schemas[schemaIndex].tables
      );
    }

    dropCacheIfNotExist(dropIfNotExist, metadata.schemas, existed.schemas);

    return setCache(existed, existedView);
  };

  // drop old data if it not exists in the new data list.
  const dropCacheIfNotExist = (
    dropIfNotExist: boolean,
    newList: { name: string }[],
    oldList: { name: string }[]
  ) => {
    if (!dropIfNotExist) {
      return;
    }
    let i = 0;
    while (i < oldList.length) {
      const index = newList.findIndex((s) => s.name === oldList[i].name);
      if (index < 0) {
        oldList.splice(i, 1);
      } else {
        i++;
      }
    }
  };

  const getDatabaseMetadataWithoutDefault = (
    database: string,
    view?: DatabaseMetadataView
  ) => getCache(database, view)[0];

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
      getDatabaseMetadataWithoutDefault(database, view) ??
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
    skipCache?: boolean;
    silent?: boolean;
  }) => {
    const {
      database,
      view = VIEW_BASIC,
      skipCache = false,
      silent = false,
    } = params;
    const { databaseName } = extractDatabaseResourceName(database);
    if (
      databaseName === String(UNKNOWN_ID) ||
      databaseName === String(EMPTY_ID)
    ) {
      return DatabaseMetadata.fromPartial({
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
      view,
    });
    const promise = databaseServiceClient.getDatabaseMetadata(
      {
        name: metadataResourceName,
        view,
      },
      {
        silent,
      }
    );
    setRequestCache(metadataResourceName, view, promise);
    promise.then((res) => {
      mergeCache(res, view, true);
    });

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
  const getTableByName = (
    database: string,
    table: string,
    schema?: string,
    view?: DatabaseMetadataView
  ) => {
    const [databaseMetadata] = getCache(database, view);
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
    const { databaseName } = extractDatabaseResourceName(database);
    if (
      databaseName === String(UNKNOWN_ID) ||
      databaseName === String(EMPTY_ID)
    ) {
      return TableMetadata.fromPartial({
        name: table,
      });
    }

    const metadataResourceName = ensureDatabaseMetadataResourceName(database);

    if (!skipCache) {
      const existedTable = getTableByName(database, table, schema, VIEW_FULL);
      if (existedTable && existedTable.columns.length > 0) {
        return existedTable;
      }

      const existedRequest = tableRequestCacheByName.getRequest([
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
      filter: `schema == "${schema}" && table == "${table}"`,
      view: VIEW_FULL,
    });
    const promise = databaseServiceClient
      .getDatabaseMetadata(
        {
          name: metadataResourceName,
          filter: `schema == "${schema}" && table == "${table}"`,
          // TODO(ed): remove the legacy view
          view: VIEW_FULL,
        },
        {
          silent,
        }
      )
      .then((res) => mergeCache(res, VIEW_BASIC))
      .then((res) => {
        return res.schemas
          .find((s) => s.name === schema)
          ?.tables.find((t) => t.name === table);
      });
    tableRequestCacheByName.setRequest(
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
  const getExtensionList = (database: string) => {
    const [databaseMetadata] = getCache(database);
    if (!databaseMetadata) {
      return [];
    }

    return databaseMetadata.extensions;
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

  const removeCache = (name: string) => {
    const metadataResourceName = ensureDatabaseMetadataResourceName(name);
    cacheByName.invalidateEntity([metadataResourceName, VIEW_FULL]);
    cacheByName.invalidateEntity([metadataResourceName, VIEW_BASIC]);
    Array.from(tableRequestCacheByName.requestCacheMap.values()).forEach(
      (cache) => {
        if (cache.keys[0] === metadataResourceName) {
          tableRequestCacheByName.invalidateRequest(cache.keys);
        }
      }
    );
  };

  return {
    getDatabaseMetadataWithoutDefault,
    getDatabaseMetadata,
    getOrFetchDatabaseMetadata,
    getSchemaList,
    getTableList,
    getOrFetchTableMetadata,
    getExternalTableList,
    getOrFetchExternalTableList,
    getViewList,
    getExtensionList,
    getFunctionList,
    removeCache,
  };
});

const ensureDatabaseResourceName = (name: string) => {
  return extractDatabaseResourceName(name).database;
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
    const { databaseName } = extractDatabaseResourceName(unref(database));
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
