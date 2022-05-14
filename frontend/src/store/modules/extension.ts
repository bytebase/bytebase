import { defineStore } from "pinia";
import axios from "axios";
import {
  Database,
  DatabaseId,
  ResourceIdentifier,
  ResourceObject,
  unknown,
  Extension,
  ExtensionState,
} from "@/types";
import { getPrincipalFromIncludedList } from "./principal";
import { useDatabaseStore } from "./database";

function convert(
  extension: ResourceObject,
  includedList: ResourceObject[]
): Extension {
  const databaseId = (
    extension.relationships!.database.data as ResourceIdentifier
  ).id;

  let database: Database = unknown("DATABASE") as Database;
  const databaseStore = useDatabaseStore();
  for (const item of includedList || []) {
    if (item.type == "database" && item.id == databaseId) {
      database = databaseStore.convert(item, includedList);
      break;
    }
  }

  return {
    ...(extension.attributes as Omit<
      Extension,
      "id" | "database" | "creator" | "updater"
    >),
    id: parseInt(extension.id),
    creator: getPrincipalFromIncludedList(
      extension.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      extension.relationships!.updater.data,
      includedList
    ),
    database,
  };
}

export const useExtensionStore = defineStore("extension", {
  state: (): ExtensionState => ({
    extensionListByDatabaseId: new Map(),
  }),

  actions: {
    getExtensionListByDatabaseId(databaseId: DatabaseId): Extension[] {
      return this.extensionListByDatabaseId.get(databaseId) || [];
    },

    async fetchExtensionListByDatabaseId(databaseId: DatabaseId) {
      const data = (await axios.get(`/api/database/${databaseId}/extension`))
        .data;
      const extensionList = data.data.map((extension: ResourceObject) => {
        return convert(extension, data.included);
      });

      this.extensionListByDatabaseId.set(databaseId, extensionList);
      return extensionList;
    },
  },
});
