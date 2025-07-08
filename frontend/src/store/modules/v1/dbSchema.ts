import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { defineStore } from "pinia";
import { computed, unref, watchEffect } from "vue";
import { databaseServiceClientConnect } from "@/grpcweb";
import { silentContextKey } from "@/grpcweb/context-key";
// Removed conversion imports as part of Bold Migration Strategy
import { useCache } from "@/store/cache";
import type { MaybeRef } from "@/types";
import { UNKNOWN_ID, EMPTY_ID, UNKNOWN_INSTANCE_NAME } from "@/types";
import { GetDatabaseMetadataRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import type {
  TableMetadata,
  DatabaseMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import {
  TableMetadataSchema,
  DatabaseMetadataSchema,
  ExternalTableMetadataSchema,
  ViewMetadataSchema,
  SchemaMetadataSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { extractDatabaseResourceName } from "@/utils";

type DatabaseMetadataCacheKey = [string /* database metadata resource name */];
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

  // getCache try use cache.
  const getCache = (name: string): DatabaseMetadata | undefined => {
    const metadataResourceName = ensureDatabaseMetadataResourceName(name);

    const full = cacheByName.getEntity([metadataResourceName]);
    if (full) {
      return full;
    }

    const entity = cacheByName.getEntity([metadataResourceName]);
    return entity;
  };
  const getRequestCache = (
    name: string
  ): Promise<DatabaseMetadata> | undefined => {
    const metadataResourceName = ensureDatabaseMetadataResourceName(name);
    const entity = cacheByName.getRequest([metadataResourceName]);
    return entity;
  };
  const setCache = (metadata: DatabaseMetadata) => {
    cacheByName.setEntity([metadata.name], metadata);
    return metadata;
  };
  const setRequestCache = (
    name: string,
    promise: Promise<DatabaseMetadata>
  ) => {
    const metadataResourceName = ensureDatabaseMetadataResourceName(name);
    cacheByName.setRequest([metadataResourceName], promise);
  };
  const mergeCache = (
    metadata: DatabaseMetadata,
    dropIfNotExist: boolean = false
  ) => {
    const existed = getCache(metadata.name);
    if (!existed) {
      return setCache(metadata);
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

    return setCache(existed);
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

  const getDatabaseMetadataWithoutDefault = (database: string) =>
    getCache(database);

  /**
   *
   * @param database
   * @param view if not specified, try find full -> basic
   * @returns
   */
  const getDatabaseMetadata = (database: string) => {
    return (
      getDatabaseMetadataWithoutDefault(database) ??
      create(DatabaseMetadataSchema, {
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
    skipCache?: boolean;
    silent?: boolean;
  }) => {
    const { database, skipCache = false, silent = false } = params;
    const { databaseName } = extractDatabaseResourceName(database);
    if (
      databaseName === String(UNKNOWN_ID) ||
      databaseName === String(EMPTY_ID)
    ) {
      return create(DatabaseMetadataSchema, {
        name: ensureDatabaseMetadataResourceName(
          `${UNKNOWN_INSTANCE_NAME}/databases/${UNKNOWN_ID}`
        ),
      });
    }

    const metadataResourceName = ensureDatabaseMetadataResourceName(database);

    if (!skipCache) {
      const existed = getCache(database);
      if (existed) {
        // The metadata entity is stored in local dictionary.
        return existed;
      }

      const cachedRequest = getRequestCache(metadataResourceName);
      if (cachedRequest) {
        // The request was sent but still not returned.
        // We won't create a duplicated request.
        return cachedRequest;
      }
    }

    // Send a request and cache it.
    console.debug("[getOrFetchDatabaseMetadata]", {
      name: metadataResourceName,
    });
    const request = create(GetDatabaseMetadataRequestSchema, {
      name: metadataResourceName,
    });
    const promise = databaseServiceClientConnect.getDatabaseMetadata(request, {
      contextValues: createContextValues().set(silentContextKey, silent),
    }); // Work directly with proto-es types
    setRequestCache(metadataResourceName, promise);
    promise.then((res) => {
      mergeCache(res, true);
    });

    return promise;
  };

  const getSchemaList = (database: string) => {
    return getCache(database)?.schemas ?? [];
  };

  const getSchemaMetadata = ({
    database,
    schema,
  }: {
    database: string;
    schema: string;
  }) => {
    return (
      getSchemaList(database).find((s) => s.name === schema) ??
      create(SchemaMetadataSchema, {})
    );
  };

  const getTableList = ({
    database,
    schema,
  }: {
    database: string;
    schema?: string;
  }) => {
    const databaseMetadata = getCache(database);
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

  const getTableMetadata = ({
    database,
    table,
    schema,
  }: {
    database: string;
    table: string;
    schema?: string;
  }) => {
    const tableList = getTableList({ database, schema });
    return (
      tableList.find((t) => t.name === table) ?? create(TableMetadataSchema, {})
    );
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
      return create(TableMetadataSchema, {
        name: table,
      });
    }

    const metadataResourceName = ensureDatabaseMetadataResourceName(database);

    if (!skipCache) {
      const existedTable = getTableMetadata({ database, table, schema });
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
    });
    const request = create(GetDatabaseMetadataRequestSchema, {
      name: metadataResourceName,
      filter: `schema == "${schema}" && table == "${table}"`,
    });
    const promise = databaseServiceClientConnect
      .getDatabaseMetadata(request, {
        contextValues: createContextValues().set(silentContextKey, silent),
      })
      .then((res) => mergeCache(res))
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

  const getExternalTableList = ({
    database,
    schema,
  }: {
    database: string;
    schema?: string;
  }) => {
    const databaseMetadata = getCache(database);
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

  const getExternalTableMetadata = ({
    database,
    schema,
    externalTable,
  }: {
    database: string;
    schema?: string;
    externalTable: string;
  }) => {
    return (
      getExternalTableList({ database, schema }).find(
        (metadata) => metadata.name === externalTable
      ) ?? create(ExternalTableMetadataSchema, {})
    );
  };

  const getOrFetchExternalTableList = async (
    database: string,
    schema?: string
  ) => {
    if (!getCache(database)) {
      await getOrFetchDatabaseMetadata({ database });
    }
    return getExternalTableList({ database, schema });
  };

  const getViewList = ({
    database,
    schema,
  }: {
    database: string;
    schema?: string;
  }) => {
    const databaseMetadata = getCache(database);
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

  const getViewMetadata = ({
    database,
    schema,
    view,
  }: {
    database: string;
    schema?: string;
    view: string;
  }) => {
    return (
      getViewList({ database, schema }).find((v) => v.name === view) ??
      create(ViewMetadataSchema, {})
    );
  };

  const getExtensionList = (database: string) => {
    const databaseMetadata = getCache(database);
    if (!databaseMetadata) {
      return [];
    }

    return databaseMetadata.extensions;
  };

  const getFunctionList = ({
    database,
    schema,
  }: {
    database: string;
    schema?: string;
  }) => {
    const databaseMetadata = getCache(database);
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
    cacheByName.invalidateEntity([metadataResourceName]);
    cacheByName.invalidateEntity([metadataResourceName]);
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
    getSchemaMetadata,
    getTableList,
    getTableMetadata,
    getOrFetchTableMetadata,
    getExternalTableList,
    getExternalTableMetadata,
    getOrFetchExternalTableList,
    getViewList,
    getViewMetadata,
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
  skipCache: MaybeRef<boolean>
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
    });
  });
  return computed(() => store.getDatabaseMetadata(unref(database)));
};
