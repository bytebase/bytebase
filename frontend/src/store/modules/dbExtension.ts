import { defineStore } from "pinia";
import axios from "axios";
import {
  Database,
  DatabaseId,
  ResourceIdentifier,
  ResourceObject,
  unknown,
  DBExtension,
  DBExtensionState,
} from "@/types";
import { getPrincipalFromIncludedList } from "./principal";
import { useDatabaseStore } from "./database";

function convert(
  dbExtension: ResourceObject,
  includedList: ResourceObject[]
): DBExtension {
  const databaseId = (
    dbExtension.relationships!.database.data as ResourceIdentifier
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
    ...(dbExtension.attributes as Omit<
      DBExtension,
      "id" | "database" | "creator" | "updater"
    >),
    id: parseInt(dbExtension.id),
    creator: getPrincipalFromIncludedList(
      dbExtension.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      dbExtension.relationships!.updater.data,
      includedList
    ),
    database,
  };
}

export const useDBExtensionStore = defineStore("dbExtension", {
  state: (): DBExtensionState => ({
    dbExtensionListByDatabaseId: new Map(),
  }),

  actions: {
    getDBExtensionListByDatabaseId(databaseId: DatabaseId): DBExtension[] {
      return this.dbExtensionListByDatabaseId.get(databaseId) || [];
    },

    async fetchdbExtensionListByDatabaseId(databaseId: DatabaseId) {
      const data = (await axios.get(`/api/database/${databaseId}/extension`))
        .data;
      const dbExtensionList = data.data.map((dbExtension: ResourceObject) => {
        return convert(dbExtension, data.included);
      });

      this.dbExtensionListByDatabaseId.set(databaseId, dbExtensionList);
      return dbExtensionList;
    },
  },
});
