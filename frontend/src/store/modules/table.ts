import { defineStore } from "pinia";
import axios from "axios";
import {
  Column,
  Database,
  DatabaseId,
  ResourceIdentifier,
  ResourceObject,
  Table,
  TableIndex,
  TableState,
  unknown,
} from "@/types";
import { getPrincipalFromIncludedList } from "./principal";
import { useDatabaseStore } from "./database";

function convert(table: ResourceObject, includedList: ResourceObject[]): Table {
  const databaseId = (table.relationships!.database.data as ResourceIdentifier)
    .id;

  let database: Database = unknown("DATABASE") as Database;
  const databaseStore = useDatabaseStore();
  for (const item of includedList || []) {
    if (item.type == "database" && item.id == databaseId) {
      database = databaseStore.convert(item, includedList);
      break;
    }
  }

  const columnList = (table.attributes.columnList as Column[]) || [];
  const indexList = (table.attributes.indexList as TableIndex[]) || [];

  return {
    ...(table.attributes as Omit<
      Table,
      "id" | "database" | "creator" | "updater" | "columnList" | "indexList"
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
    columnList,
    indexList,
    database,
  };
}

export const useTableStore = defineStore("table", {
  state: (): TableState => ({
    tableListByDatabaseId: new Map(),
  }),

  actions: {
    getTableListByDatabaseIdAndTableName(
      databaseId: DatabaseId,
      tableName: string
    ): Table | undefined {
      const list = this.tableListByDatabaseId.get(databaseId);
      if (list) {
        return list.find((item: Table) => item.name == tableName);
      }
      return undefined;
    },

    getTableListByDatabaseId(databaseId: DatabaseId): Table[] {
      return this.tableListByDatabaseId.get(databaseId) || [];
    },

    setTableByDatabaseIdAndTableName({
      databaseId,
      tableName,
      table,
    }: {
      databaseId: DatabaseId;
      tableName: string;
      table: Table;
    }) {
      const list = this.tableListByDatabaseId.get(databaseId);
      if (list) {
        const i = list.findIndex((item: Table) => item.name == tableName);
        if (i != -1) {
          list[i] = table;
        } else {
          list.push(table);
        }
      } else {
        this.tableListByDatabaseId.set(databaseId, [table]);
      }
    },

    setTableListByDatabaseId({
      databaseId,
      tableList,
    }: {
      databaseId: DatabaseId;
      tableList: Table[];
    }) {
      this.tableListByDatabaseId.set(databaseId, tableList);
    },

    async fetchTableByDatabaseIdAndTableName({
      databaseId,
      tableName,
    }: {
      databaseId: DatabaseId;
      tableName: string;
    }) {
      const data = (
        await axios.get(`/api/database/${databaseId}/table/${tableName}`)
      ).data;
      const table = convert(data.data, data.included);

      this.setTableByDatabaseIdAndTableName({
        databaseId,
        tableName,
        table,
      });
      return table;
    },

    async fetchTableListByDatabaseId(databaseId: DatabaseId) {
      const data = (await axios.get(`/api/database/${databaseId}/table`)).data;
      const tableList = data.data.map((table: ResourceObject) => {
        return convert(table, data.included);
      });

      this.setTableListByDatabaseId({ databaseId, tableList });
      return tableList;
    },
  },
});
