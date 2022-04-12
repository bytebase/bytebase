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

function convert(
  view: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): View {
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

const state: () => ViewState = () => ({
  viewListByDatabaseId: new Map(),
});

const getters = {
  viewListByDatabaseId:
    (state: ViewState) =>
    (databaseId: DatabaseId): View[] => {
      return state.viewListByDatabaseId.get(databaseId) || [];
    },
};

const actions = {
  async fetchViewListByDatabaseId(
    { commit, rootGetters }: any,
    databaseId: DatabaseId
  ) {
    const data = (await axios.get(`/api/database/${databaseId}/view`)).data;
    const viewList = data.data.map((view: ResourceObject) => {
      return convert(view, data.included, rootGetters);
    });

    commit("setViewListByDatabaseId", { databaseId, viewList });
    return viewList;
  },
};

const mutations = {
  setViewListByDatabaseId(
    state: ViewState,
    {
      databaseId,
      viewList,
    }: {
      databaseId: DatabaseId;
      viewList: View[];
    }
  ) {
    state.viewListByDatabaseId.set(databaseId, viewList);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
