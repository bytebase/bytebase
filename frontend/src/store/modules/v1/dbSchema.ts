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
} from "@/types/proto/v1/database_service";
import { getInstanceAndDatabaseId } from "./common";

interface DBSchemaState {
  requestCache: Map<string, Promise<DatabaseMetadata>>;
  databaseMetadataByName: Map<string, DatabaseMetadata>;
}

export const useDBSchemaV1Store = defineStore("dbSchema_v1", {
  state: (): DBSchemaState => ({
    requestCache: new Map(),
    databaseMetadataByName: new Map(),
  }),
  actions: {
    async getOrFetchDatabaseMetadata(
      name: string,
      skipCache = false
    ): Promise<DatabaseMetadata> {
      const databaseId = getInstanceAndDatabaseId(name)[1];
      if (
        Number(databaseId) === UNKNOWN_ID ||
        Number(databaseId) === EMPTY_ID
      ) {
        return DatabaseMetadata.fromJSON({
          name: "<<Unknown database>>",
        });
      }

      if (!skipCache) {
        if (this.databaseMetadataByName.has(name)) {
          // The metadata entity is stored in local dictionary.
          return this.databaseMetadataByName.get(name) as DatabaseMetadata;
        }

        const cachedRequest = this.requestCache.get(name);
        if (cachedRequest) {
          // The request was sent but still not returned.
          // We won't create a duplicated request.
          return cachedRequest;
        }
      }

      // Send a request and cache it.
      const promise = databaseServiceClient
        .getDatabaseMetadata({
          name: `${name}/metadata`,
        })
        .then((res) => {
          this.databaseMetadataByName.set(name, res);
          return res;
        });
      this.requestCache.set(name, promise);

      return promise;
    },
    async getOrFetchSchemaList(
      name: string,
      skipCache = false
    ): Promise<SchemaMetadata[]> {
      if (skipCache || !this.databaseMetadataByName.has(name)) {
        await this.getOrFetchDatabaseMetadata(name, skipCache);
      }
      return this.getSchemaList(name);
    },
    getDatabaseMetadata(name: string): DatabaseMetadata {
      return (
        this.databaseMetadataByName.get(name) ??
        DatabaseMetadata.fromPartial({})
      );
    },
    getSchemaList(name: string): SchemaMetadata[] {
      const databaseMetadata = this.databaseMetadataByName.get(name);
      if (!databaseMetadata) {
        return [];
      }

      return databaseMetadata.schemas;
    },
    async getOrFetchTableList(name: string): Promise<TableMetadata[]> {
      if (!this.databaseMetadataByName.has(name)) {
        await this.getOrFetchDatabaseMetadata(name);
      }
      return this.getTableList(name);
    },
    getTableList(name: string): TableMetadata[] {
      const databaseMetadata = this.databaseMetadataByName.get(name);
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
    getTableByName(name: string, tableName: string): TableMetadata | undefined {
      const databaseMetadata = this.databaseMetadataByName.get(name);
      if (!databaseMetadata) {
        return undefined;
      }

      const tableList = this.getTableList(name);
      return tableList.find((table) => table.name === tableName);
    },
    async getOrFetchViewList(name: string): Promise<ViewMetadata[]> {
      if (!this.databaseMetadataByName.has(name)) {
        await this.getOrFetchDatabaseMetadata(name);
      }
      return this.getViewList(name);
    },
    getViewList(name: string): ViewMetadata[] {
      const databaseMetadata = this.databaseMetadataByName.get(name);
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
      if (!this.databaseMetadataByName.has(name)) {
        await this.getOrFetchDatabaseMetadata(name);
      }
      return this.getExtensionList(name);
    },
    getExtensionList(name: string): ExtensionMetadata[] {
      const databaseMetadata = this.databaseMetadataByName.get(name);
      if (!databaseMetadata) {
        return [];
      }

      return databaseMetadata.extensions;
    },
    async getOrFetchFunctionList(name: string): Promise<FunctionMetadata[]> {
      if (!this.databaseMetadataByName.has(name)) {
        await this.getOrFetchDatabaseMetadata(name);
      }
      return this.getFunctionList(name);
    },
    getFunctionList(name: string): FunctionMetadata[] {
      const databaseMetadata = this.databaseMetadataByName.get(name);
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
      this.requestCache.delete(name);
      this.databaseMetadataByName.delete(name);
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
      store.getOrFetchDatabaseMetadata(id, unref(skipCache));
    }
  });
  return computed(() => store.databaseMetadataByName.get(unref(name)));
};
