import axios from "axios";
import { defineStore } from "pinia";
import { DatabaseId, DBSchemaState } from "@/types";
import {
  DatabaseMetadata,
  ExtensionMetadata,
  SchemaMetadata,
  TableMetadata,
  ViewMetadata,
} from "@/types/proto/database";

const requestCache = new Map<DatabaseId, Promise<DatabaseMetadata>>();

export const useDBSchemaStore = defineStore("dbSchema", {
  state: (): DBSchemaState => ({
    databaseMetadataById: new Map(),
  }),
  actions: {
    async getOrFetchDatabaseMetadataById(
      databaseId: DatabaseId
    ): Promise<DatabaseMetadata> {
      if (this.databaseMetadataById.has(databaseId)) {
        return this.databaseMetadataById.get(databaseId) as DatabaseMetadata;
      }

      const cachedRequest = requestCache.get(databaseId);
      if (cachedRequest) {
        return cachedRequest;
      }

      const promise = axios
        .get(`/api/database/${databaseId}/schema?metadata=true`)
        .then((res) => {
          const databaseMetadata = DatabaseMetadata.fromJSON(res.data);
          this.databaseMetadataById.set(databaseId, databaseMetadata);
          return databaseMetadata;
        });
      requestCache.set(databaseId, promise);

      return promise;
    },
    async getOrFetchSchemaListByDatabaseId(
      databaseId: DatabaseId
    ): Promise<SchemaMetadata[]> {
      if (!this.databaseMetadataById.has(databaseId)) {
        await this.getOrFetchDatabaseMetadataById(databaseId);
      }
      return this.getSchemaListByDatabaseId(databaseId);
    },
    getSchemaListByDatabaseId(databaseId: DatabaseId): SchemaMetadata[] {
      const databaseMetadata = this.databaseMetadataById.get(databaseId);
      if (!databaseMetadata) {
        return [];
      }

      return databaseMetadata.schemas;
    },
    async getOrFetchTableListByDatabaseId(
      databaseId: DatabaseId
    ): Promise<TableMetadata[]> {
      if (!this.databaseMetadataById.has(databaseId)) {
        await this.getOrFetchDatabaseMetadataById(databaseId);
      }
      return this.getTableListByDatabaseId(databaseId);
    },
    getTableListByDatabaseId(databaseId: DatabaseId): TableMetadata[] {
      const databaseMetadata = this.databaseMetadataById.get(databaseId);
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
    getTableByDatabaseIdAndTableName(
      databaseId: DatabaseId,
      tableName: string
    ): TableMetadata | undefined {
      const databaseMetadata = this.databaseMetadataById.get(databaseId);
      if (!databaseMetadata) {
        return undefined;
      }

      const tableList = this.getTableListByDatabaseId(databaseId);
      return tableList.find((table) => table.name === tableName);
    },
    async getOrFetchViewListByDatabaseId(
      databaseId: DatabaseId
    ): Promise<ViewMetadata[]> {
      if (!this.databaseMetadataById.has(databaseId)) {
        await this.getOrFetchDatabaseMetadataById(databaseId);
      }
      return this.getViewListByDatabaseId(databaseId);
    },
    getViewListByDatabaseId(databaseId: DatabaseId): ViewMetadata[] {
      const databaseMetadata = this.databaseMetadataById.get(databaseId);
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
    async getOrFetchExtensionListByDatabaseId(
      databaseId: DatabaseId
    ): Promise<ExtensionMetadata[]> {
      if (!this.databaseMetadataById.has(databaseId)) {
        await this.getOrFetchDatabaseMetadataById(databaseId);
      }
      return this.getExtensionListByDatabaseId(databaseId);
    },
    getExtensionListByDatabaseId(databaseId: DatabaseId): ExtensionMetadata[] {
      const databaseMetadata = this.databaseMetadataById.get(databaseId);
      if (!databaseMetadata) {
        return [];
      }

      return databaseMetadata.extensions;
    },
  },
});
