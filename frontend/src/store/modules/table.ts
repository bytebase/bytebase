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
import { getPrincipalFromIncludedList } from "../pinia";

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
      database = rootGetters["database/convert"](item, includedList);
      break;
    }
  }

  return {
    ...(table.attributes as Omit<
      Table,
      "id" | "database" | "creator" | "updater"
    >),
    id: parseInt(table.id),
    creator: getPrincipalFromIncludedList(
      table.relationships!.creator.data,
      includedList
    ),
    updater: getPrincipalFromIncludedList(
      table.relationships!.updater.data,
      includedList
    ),
    database,
  };
}

const state: () => TableState = () => ({
  tableListByDatabaseId: new Map(),
});

const getters = {
  tableListByDatabaseIdAndTableName:
    (state: TableState) =>
    (databaseId: DatabaseId, tableName: string): Table | undefined => {
      const list = state.tableListByDatabaseId.get(databaseId);
      if (list) {
        return list.find((item: Table) => item.name == tableName);
      }
      return undefined;
    },

  tableListByDatabaseId:
    (state: TableState) =>
    (databaseId: DatabaseId): Table[] => {
      return state.tableListByDatabaseId.get(databaseId) || [];
    },
};

const actions = {
  async fetchTableByDatabaseIdAndTableName(
    { commit, rootGetters }: any,
    { databaseId, tableName }: { databaseId: DatabaseId; tableName: string }
  ) {
    const data = (
      await axios.get(`/api/database/${databaseId}/table/${tableName}`)
    ).data;
    const table = convert(data.data, data.included, rootGetters);

    commit("setTableByDatabaseIdAndTableName", {
      databaseId,
      tableName,
      table,
    });
    return table;
  },

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
  setTableByDatabaseIdAndTableName(
    state: TableState,
    {
      databaseId,
      tableName,
      table,
    }: {
      databaseId: DatabaseId;
      tableName: string;
      table: Table;
    }
  ) {
    const list = state.tableListByDatabaseId.get(databaseId);
    if (list) {
      const i = list.findIndex((item: Table) => item.name == tableName);
      if (i != -1) {
        list[i] = table;
      } else {
        list.push(table);
      }
    } else {
      state.tableListByDatabaseId.set(databaseId, [table]);
    }
  },

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
