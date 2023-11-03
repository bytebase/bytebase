import { defineStore } from "pinia";
import { computed, unref, watchEffect } from "vue";
import { databaseServiceClient } from "@/grpcweb";
import { MaybeRef, UNKNOWN_ID, EMPTY_ID } from "@/types";
import {
  DatabaseMetadata,
  ExtensionMetadata,
  SchemaMetadata,
  TableMetadata,
  ViewMetadata,
  FunctionMetadata,
  DatabaseMetadataView,
} from "@/types/proto/v1/database_service";
import { getInstanceAndDatabaseId } from "./common";

interface DBSchemaState {
  requestCache: Map<string, Promise<DatabaseMetadata>>;
  _databaseMetadataByName: Map<string, DatabaseMetadata>;
}

export const useDBSchemaV1Store = defineStore("dbSchema_v1", {
  state: (): DBSchemaState => ({
    requestCache: new Map(),
    _databaseMetadataByName: new Map(),
  }),
  actions: {
    getFromCache(databaseName: string) {
      return this._databaseMetadataByName.get(this.getMedataName(databaseName));
    },
    setCache(metadata: DatabaseMetadata) {
      this._databaseMetadataByName.set(metadata.name, metadata);
      return metadata;
    },
    mergeToCache(metadata: DatabaseMetadata) {
      const existed = this._databaseMetadataByName.get(metadata.name);
      if (!existed) {
        return this.setCache(metadata);
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

      return this.setCache(existed);
    },
    getMedataName(databaseName: string) {
      return `${databaseName}/metadata`;
    },
    async updateDatabaseSchemaConfigs(metadata: DatabaseMetadata) {
      const updated = await databaseServiceClient.updateDatabaseMetadata({
        databaseMetadata: metadata,
        updateMask: ["schema_configs"],
      });
      this.setCache(updated);
      return updated;
    },
    async getOrFetchDatabaseMetadata({
      database,
      skipCache = false,
      silent = false,
      view = DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL,
      filter = undefined,
    }: {
      database: string;
      skipCache?: boolean;
      silent?: boolean;
      view?: DatabaseMetadataView;
      filter?: string | undefined;
    }): Promise<DatabaseMetadata> {
      const databaseId = getInstanceAndDatabaseId(database)[1];
      if (
        Number(databaseId) === UNKNOWN_ID ||
        Number(databaseId) === EMPTY_ID
      ) {
        return DatabaseMetadata.fromJSON({
          name: "<<Unknown database>>",
        });
      }

      const metadataName = this.getMedataName(database);

      if (!skipCache) {
        const existed = this.getFromCache(database);
        if (existed) {
          // The metadata entity is stored in local dictionary.
          return existed;
        }

        const cachedRequest = this.requestCache.get(metadataName);
        if (cachedRequest) {
          // The request was sent but still not returned.
          // We won't create a duplicated request.
          return cachedRequest;
        }
      }

      // Send a request and cache it.
      const promise = databaseServiceClient
        .getDatabaseMetadata(
          {
            name: metadataName,
            filter,
            view,
          },
          {
            silent,
          }
        )
        .then((res) => {
          this.setCache(res);
          return res;
        });
      this.requestCache.set(metadataName, promise);

      return promise;
    },
    async getOrFetchSchemaList(
      name: string,
      skipCache = false
    ): Promise<SchemaMetadata[]> {
      if (skipCache || !this.getFromCache(name)) {
        await this.getOrFetchDatabaseMetadata({
          database: name,
          skipCache,
        });
      }
      return this.getSchemaList(name);
    },
    getDatabaseMetadata(name: string): DatabaseMetadata {
      return (
        this.getFromCache(name) ??
        DatabaseMetadata.fromPartial({
          name: this.getMedataName(name),
        })
      );
    },
    getSchemaList(name: string): SchemaMetadata[] {
      return this.getFromCache(name)?.schemas ?? [];
    },
    async getOrFetchTableList(name: string): Promise<TableMetadata[]> {
      if (!this.getFromCache(name)) {
        await this.getOrFetchDatabaseMetadata({ database: name });
      }
      return this.getTableList(name);
    },
    getTableList(name: string): TableMetadata[] {
      const databaseMetadata = this.getFromCache(name);
      if (!databaseMetadata) {
        return [];
      }

      const tableList: TableMetadata[] = [];
      // TODO(steven): get table list with schema name for PG.
      for (const schema of databaseMetadata.schemas) {
        tableList.push(...schema.tables);
      }
      return tableList;
    },
    async getOrFetchTableMetadata({
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
    }) {
      if (!skipCache) {
        const existedTable = this.getTableByName(database, table);
        if (existedTable && existedTable.columns.length > 0) {
          return existedTable;
        }
      }

      const metadataName = this.getMedataName(database);
      return databaseServiceClient
        .getDatabaseMetadata(
          {
            name: metadataName,
            filter: `schemas/${schema || "-"}/tables/${table}`,
            view: DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL,
          },
          {
            silent,
          }
        )
        .then((res) => {
          let tableMetadata = TableMetadata.fromPartial({});
          for (const s of res.schemas) {
            if (s.name !== schema) {
              continue;
            }
            for (const t of s.tables) {
              if (t.name === table) {
                tableMetadata = t;
                break;
              }
            }
          }
          this.mergeToCache(res);
          return tableMetadata;
        });
    },
    getTableByName(name: string, tableName: string): TableMetadata | undefined {
      const databaseMetadata = this.getFromCache(name);
      if (!databaseMetadata) {
        return undefined;
      }

      const tableList = this.getTableList(name);
      return tableList.find((table) => table.name === tableName);
    },
    async getOrFetchViewList(name: string): Promise<ViewMetadata[]> {
      if (!this.getFromCache(name)) {
        await this.getOrFetchDatabaseMetadata({ database: name });
      }
      return this.getViewList(name);
    },
    getViewList(name: string): ViewMetadata[] {
      const databaseMetadata = this.getFromCache(name);
      if (!databaseMetadata) {
        return [];
      }

      const viewList: ViewMetadata[] = [];
      // TODO(steven): get view list with schema name for PG.
      for (const schema of databaseMetadata.schemas) {
        viewList.push(...schema.views);
      }
      return viewList;
    },
    async getOrFetchExtensionList(name: string): Promise<ExtensionMetadata[]> {
      if (!this.getFromCache(name)) {
        await this.getOrFetchDatabaseMetadata({ database: name });
      }
      return this.getExtensionList(name);
    },
    getExtensionList(name: string): ExtensionMetadata[] {
      const databaseMetadata = this.getFromCache(name);
      if (!databaseMetadata) {
        return [];
      }

      return databaseMetadata.extensions;
    },
    async getOrFetchFunctionList(name: string): Promise<FunctionMetadata[]> {
      if (!this.getFromCache(name)) {
        await this.getOrFetchDatabaseMetadata({ database: name });
      }
      return this.getFunctionList(name);
    },
    getFunctionList(name: string): FunctionMetadata[] {
      const databaseMetadata = this.getFromCache(name);
      if (!databaseMetadata) {
        return [];
      }

      const functionList: FunctionMetadata[] = [];
      for (const schema of databaseMetadata.schemas) {
        functionList.push(...schema.functions);
      }
      return functionList;
    },
    removeCache(name: string) {
      this.requestCache.delete(this.getMedataName(name));
      this._databaseMetadataByName.delete(this.getMedataName(name));
    },
  },
});

export const useMetadata = (
  name: MaybeRef<string>,
  skipCache: MaybeRef<boolean>
) => {
  const store = useDBSchemaV1Store();
  watchEffect(() => {
    const id = unref(name);
    const uid = getInstanceAndDatabaseId(id)[1];
    if (Number(uid) !== UNKNOWN_ID && Number(uid) !== EMPTY_ID) {
      store.getOrFetchDatabaseMetadata({
        database: id,
        skipCache: unref(skipCache),
      });
    }
  });
  return computed(() => store.getFromCache(unref(name)));
};
