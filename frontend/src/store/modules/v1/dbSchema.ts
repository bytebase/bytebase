import { create } from "@bufbuild/protobuf";
import { createContextValues } from "@connectrpc/connect";
import { defineStore } from "pinia";
import { computed, unref, watchEffect } from "vue";
import { databaseServiceClientConnect } from "@/connect";
import { silentContextKey } from "@/connect/context-key";
// Removed conversion imports as part of Bold Migration Strategy
import { useCache } from "@/store/cache";
import type { MaybeRef } from "@/types";
import { EMPTY_ID, UNKNOWN_ID, UNKNOWN_INSTANCE_NAME } from "@/types";
import type { DatabaseMetadata } from "@/types/proto-es/v1/database_service_pb";
import {
  DatabaseMetadataSchema,
  ExternalTableMetadataSchema,
  GetDatabaseMetadataRequestSchema,
  SchemaMetadataSchema,
  TableMetadataSchema,
  ViewMetadataSchema,
} from "@/types/proto-es/v1/database_service_pb";
import { extractDatabaseResourceName } from "@/utils";

type DatabaseMetadataCacheKey = [
  string /* database metadata resource name */,
  string /* filter */,
  number /* limit */,
];

export const useDBSchemaV1Store = defineStore("dbSchema_v1", () => {
  const cacheByName = useCache<DatabaseMetadataCacheKey, DatabaseMetadata>(
    "bb.db-schema.by-name"
  );

  // getCache try use cache.
  const getCache = ({
    name,
    filter,
    limit,
  }: {
    name: string;
    filter: string;
    limit: number;
  }): DatabaseMetadata | undefined => {
    const metadataResourceName = ensureDatabaseMetadataResourceName(name);
    return cacheByName.getEntity([metadataResourceName, filter, limit]);
  };

  const getRequestCache = ({
    name,
    filter,
    limit,
  }: {
    name: string;
    filter: string;
    limit: number;
  }): Promise<DatabaseMetadata> | undefined => {
    const metadataResourceName = ensureDatabaseMetadataResourceName(name);
    return cacheByName.getRequest([metadataResourceName, filter, limit]);
  };

  const setCache = ({
    metadata,
    filter,
    limit,
  }: {
    metadata: DatabaseMetadata;
    filter: string;
    limit: number;
  }) => {
    cacheByName.setEntity([metadata.name, filter, limit], metadata);
    return metadata;
  };

  const setRequestCache = ({
    name,
    filter,
    limit,
    promise,
  }: {
    name: string;
    filter: string;
    limit: number;
    promise: Promise<DatabaseMetadata>;
  }) => {
    const metadataResourceName = ensureDatabaseMetadataResourceName(name);
    cacheByName.setRequest([metadataResourceName, filter, limit], promise);
    return promise;
  };

  const getDatabaseMetadataWithoutDefault = (database: string) =>
    getCache({
      name: database,
      filter: "",
      limit: 0,
    });

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
    limit?: number; // limit the number of returned tables per schema
    filter?: string; // used to filter schema and table, e.g. schema == "public" && table.matches("user*")
  }) => {
    const {
      limit = 0,
      filter = "",
      database,
      silent = false,
      skipCache = false,
    } = params;
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

    if (!skipCache) {
      const existed = getCache({ name: database, filter, limit });
      if (existed) {
        // The metadata entity is stored in local dictionary.
        return existed;
      }
      const cachedRequest = getRequestCache({
        name: database,
        filter,
        limit,
      });
      if (cachedRequest) {
        // The request was sent but still not returned.
        // We won't create a duplicated request.
        return cachedRequest;
      }
    }

    // Send a request and cache it.
    console.debug("[getOrFetchDatabaseMetadata]", {
      database,
      limit,
      filter,
    });
    const request = create(GetDatabaseMetadataRequestSchema, {
      name: ensureDatabaseMetadataResourceName(database),
      limit,
      filter,
    });
    const promise = databaseServiceClientConnect
      .getDatabaseMetadata(request, {
        contextValues: createContextValues().set(silentContextKey, silent),
      })
      .then((res) => {
        return setCache({ metadata: res, filter, limit });
      });
    return setRequestCache({ name: database, filter, limit, promise });
  };

  const getSchemaList = (database: string) => {
    return getDatabaseMetadataWithoutDefault(database)?.schemas ?? [];
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
    const databaseMetadata = getDatabaseMetadataWithoutDefault(database);
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
    silent = false,
  }: {
    database: string;
    schema: string;
    table: string;
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

    const tableMetadata = getTableMetadata({ database, schema, table });
    if (tableMetadata.name === table) {
      return tableMetadata;
    }

    return getOrFetchDatabaseMetadata({
      database,
      silent,
      filter: `schema == "${schema}" && table == "${table}"`,
    }).then((res) => {
      return res.schemas
        .find((s) => s.name === schema)
        ?.tables.find((t) => t.name === table);
    });
  };

  const getExternalTableList = ({
    database,
    schema,
  }: {
    database: string;
    schema?: string;
  }) => {
    const databaseMetadata = getDatabaseMetadataWithoutDefault(database);
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
    if (!getDatabaseMetadataWithoutDefault(database)) {
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
    const databaseMetadata = getDatabaseMetadataWithoutDefault(database);
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
    const databaseMetadata = getDatabaseMetadataWithoutDefault(database);
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
    const databaseMetadata = getDatabaseMetadataWithoutDefault(database);
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
    Array.from(cacheByName.requestCacheMap.values()).forEach((cache) => {
      if (cache.keys[0] === metadataResourceName) {
        cacheByName.invalidateRequest(cache.keys);
      }
    });
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
