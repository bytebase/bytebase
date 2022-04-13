import { defineStore } from "pinia";
import axios from "axios";
import {
  Database,
  DatabaseId,
  ResourceIdentifier,
  ResourceObject,
  unknown,
  View,
  ViewState,
} from "../../types";
import { getPrincipalFromIncludedList } from "../pinia";
import { useDatabaseStore } from "../pinia-modules";

function convert(view: ResourceObject, includedList: ResourceObject[]): View {
  const databaseId = (view.relationships!.database.data as ResourceIdentifier)
    .id;

  let database: Database = unknown("DATABASE") as Database;
  const databaseStore = useDatabaseStore();
  for (const item of includedList || []) {
    if (item.type == "database" && item.id == databaseId) {
      database = databaseStore.convert(item, includedList);
      break;
    }
  }

  return {
    ...(view.attributes as Omit<
      View,
      "id" | "database" | "creator" | "updater"
    >),
    id: parseInt(view.id),
    creator: getPrincipalFromIncludedList(
      view.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      view.relationships!.updater.data,
      includedList
    ),
    database,
  };
}

export const useViewStore = defineStore("view", {
  state: (): ViewState => ({
    viewListByDatabaseId: new Map(),
  }),

  actions: {
    getViewListByDatabaseId(databaseId: DatabaseId): View[] {
      return this.viewListByDatabaseId.get(databaseId) || [];
    },

    async fetchViewListByDatabaseId(databaseId: DatabaseId) {
      const data = (await axios.get(`/api/database/${databaseId}/view`)).data;
      const viewList = data.data.map((view: ResourceObject) => {
        return convert(view, data.included);
      });

      this.viewListByDatabaseId.set(databaseId, viewList);
      return viewList;
    },
  },
});
