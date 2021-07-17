import axios from "axios";
import {
  Database,
  DatabaseId,
  ResourceIdentifier,
  ResourceObject,
  Table,
  TableState,
  unknown,
} from "../../types";

function convert(
  table: ResourceObject,
  includedList: ResourceObject[],
  rootGetters: any
): Table {
  const databaseId = (table.relationships!.database.data as ResourceIdentifier)
    .id;

  let database: Database = unknown("DATABASE") as Database;
  for (const item of includedList || []) {
    if (item.type == "database" && item.id == databaseId) {
      database = rootGetters["database/convert"](item);
      break;
    }
  }
  return {
    ...(table.attributes as Omit<Table, "id" | "database">),
    id: parseInt(table.id),
    database,
  };
}

const state: () => TableState = () => ({
  tableListByDatabaseId: new Map(),
});

const getters = {
  tableListByDatabaseId:
    (state: TableState) =>
    (databaseId: DatabaseId): Table[] => {
      return state.tableListByDatabaseId.get(databaseId) || [];
    },
};

const actions = {
  async fetchTableListByDatabaseId(
    { commit, rootGetters }: any,
    databaseId: DatabaseId
  ) {
    const data = (await axios.get(`/api/database/${databaseId}/table`)).data;
    const tableList = data.data.map((table: ResourceObject) => {
      return convert(table, data.included, rootGetters);
    });

    commit("setTableListByDatabaseId", { databaseId, tableList });
    return tableList;
  },
};

const mutations = {
  setTableListByDatabaseId(
    state: TableState,
    {
      databaseId,
      tableList,
    }: {
      databaseId: DatabaseId;
      tableList: Table[];
    }
  ) {
    state.tableListByDatabaseId.set(databaseId, tableList);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
