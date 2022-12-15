import axios from "axios";
import { defineStore } from "pinia";
import { DatabaseId, DBSchemaState } from "@/types";
import { DatabaseMetadata } from "@/types/proto/database";

export const useDBSchemaStore = defineStore("dbSchema", {
  state: (): DBSchemaState => ({
    databaseMetadataById: new Map(),
  }),
  actions: {
    async getOrFetchDatabaseMetadata(
      databaseId: DatabaseId
    ): Promise<DatabaseMetadata> {
      if (this.databaseMetadataById.has(databaseId)) {
        return this.databaseMetadataById.get(databaseId) as DatabaseMetadata;
      }

      const res = await axios.get<DatabaseMetadata>(
        `/api/database/${databaseId}/schema?metadata=true`
      );
      const databaseMetadata = DatabaseMetadata.fromJSON(res.data);
      this.databaseMetadataById.set(databaseId, databaseMetadata);
      return databaseMetadata;
    },
  },
});
