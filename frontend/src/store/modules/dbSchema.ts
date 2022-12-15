import axios from "axios";
import { defineStore } from "pinia";
import { DatabaseId, DBSchemaState } from "@/types";
import { DatabaseMetadata, TableMetadata } from "@/types/proto/database";

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

      const res = await axios.get(
        `/api/database/${databaseId}/schema?metadata=true`
      );
      const databaseMetadata = DatabaseMetadata.fromJSON(res.data);
      this.databaseMetadataById.set(databaseId, databaseMetadata);
      return databaseMetadata;
    },
    getTableListByDatabaseId(databaseId: DatabaseId): TableMetadata[] {
      const databaseMetadata = this.databaseMetadataById.get(databaseId);
      if (!databaseMetadata) {
        return [];
      }

      const tableList: TableMetadata[] = [];
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
  },
});
