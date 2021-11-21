import axios from "axios";
import {
  Database,
  DatabaseID,
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
  const databaseID = (table.relationships!.database.data as ResourceIdentifier)
    .id;

  let database: Database = unknown("DATABASE") as Database;
  for (const item of includedList || []) {
    if (item.type == "database" && item.id == databaseID) {
      database = rootGetters["database/convert"](item, includedList);
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
  tableListByDatabaseID: new Map(),
});

const getters = {
  tableListByDatabaseIDAndTableName:
    (state: TableState) =>
    (databaseID: DatabaseID, tableName: string): Table | undefined => {
      const list = state.tableListByDatabaseID.get(databaseID);
      if (list) {
        return list.find((item: Table) => item.name == tableName);
      }
      return undefined;
    },

  tableListByDatabaseID:
    (state: TableState) =>
    (databaseID: DatabaseID): Table[] => {
      return state.tableListByDatabaseID.get(databaseID) || [];
    },
};

const actions = {
  async fetchTableByDatabaseIDAndTableName(
    { commit, rootGetters }: any,
    { databaseID, tableName }: { databaseID: DatabaseID; tableName: string }
  ) {
    const data = (
      await axios.get(`/api/database/${databaseID}/table/${tableName}`)
    ).data;
    const table = convert(data.data, data.included, rootGetters);

    commit("setTableByDatabaseIDAndTableName", {
      databaseID,
      tableName,
      table,
    });
    return table;
  },

  async fetchTableListByDatabaseID(
    { commit, rootGetters }: any,
    databaseID: DatabaseID
  ) {
    const data = (await axios.get(`/api/database/${databaseID}/table`)).data;
    const tableList = data.data.map((table: ResourceObject) => {
      return convert(table, data.included, rootGetters);
    });

    commit("setTableListByDatabaseID", { databaseID, tableList });
    return tableList;
  },
};

const mutations = {
  setTableByDatabaseIDAndTableName(
    state: TableState,
    {
      databaseID,
      tableName,
      table,
    }: {
      databaseID: DatabaseID;
      tableName: string;
      table: Table;
    }
  ) {
    const list = state.tableListByDatabaseID.get(databaseID);
    if (list) {
      const i = list.findIndex((item: Table) => item.name == tableName);
      if (i != -1) {
        list[i] = table;
      } else {
        list.push(table);
      }
    } else {
      state.tableListByDatabaseID.set(databaseID, [table]);
    }
  },

  setTableListByDatabaseID(
    state: TableState,
    {
      databaseID,
      tableList,
    }: {
      databaseID: DatabaseID;
      tableList: Table[];
    }
  ) {
    state.tableListByDatabaseID.set(databaseID, tableList);
  },
};

export default {
  namespaced: true,
  state,
  getters,
  actions,
  mutations,
};
