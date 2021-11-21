import axios from "axios";
import {
  Database,
  DatabaseID,
  ResourceIdentifier,
  ResourceObject,
  unknown,
  View,
  ViewState,
} from "../../types";

function convert(
  view: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): View {
  const databaseID = (view.relationships!.database.data as ResourceIdentifier)
    .id;

  let database: Database = unknown("DATABASE") as Database;
  for (const item of includedList || []) {
    if (item.type == "database" && item.id == databaseID) {
      database = rootGetters["database/convert"](item, includedList);
      break;
    }
  }
  return {
    ...(view.attributes as Omit<View, "id" | "database">),
    id: parseInt(view.id),
    database,
  };
}

const state: () => ViewState = () => ({
  viewListByDatabaseID: new Map(),
});

const getters = {
  viewListByDatabaseID:
    (state: ViewState) =>
    (databaseID: DatabaseID): View[] => {
      return state.viewListByDatabaseID.get(databaseID) || [];
    },
};

const actions = {
  async fetchViewListByDatabaseID(
    { commit, rootGetters }: any,
    databaseID: DatabaseID
  ) {
    const data = (await axios.get(`/api/database/${databaseID}/view`)).data;
    const viewList = data.data.map((view: ResourceObject) => {
      return convert(view, data.included, rootGetters);
    });

    commit("setViewListByDatabaseID", { databaseID, viewList });
    return viewList;
  },
};

const mutations = {
  setViewListByDatabaseID(
    state: ViewState,
    {
      databaseID,
      viewList,
    }: {
      databaseID: DatabaseID;
      viewList: View[];
    }
  ) {
    state.viewListByDatabaseID.set(databaseID, viewList);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
