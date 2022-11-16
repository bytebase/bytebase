import axios from "axios";
import { defineStore } from "pinia";
import { cloneDeep, isUndefined, uniqueId } from "lodash-es";
import {
  DatabaseId,
  TableId,
  Table,
  Database,
  unknown,
  UNKNOWN_ID,
  TabContext,
  UIEditorState,
  UIEditorTabType,
  TableTabContext,
  DatabaseEdit,
} from "@/types";
import { useDatabaseStore, useTableStore } from "./";

export const generateUniqueTabId = () => {
  return uniqueId();
};

const getDefaultUIEditorState = (): UIEditorState => {
  return {
    tabState: {
      tabMap: new Map<string, TabContext>(),
      currentTabId: "",
    },
    databaseList: [],
    tableList: [],
  };
};

export const useUIEditorStore = defineStore("UIEditor", {
  state: (): UIEditorState => {
    return getDefaultUIEditorState();
  },
  getters: {
    currentTab(state) {
      return state.tabState.tabMap.get(state.tabState.currentTabId);
    },
  },
  actions: {
    getTabById(tabId: string) {
      return this.tabState.tabMap.get(tabId);
    },
    // getTabByDatabaseIdAndTableName gets tab by database id and table name.
    // * tableName is using to position those new tables with UNKNOWN_ID.
    getTabByDatabaseIdAndTableName(databaseId: DatabaseId, tableName?: string) {
      const wantedTabType = isUndefined(tableName)
        ? UIEditorTabType.TabForDatabase
        : UIEditorTabType.TabForTable;
      for (const [_, tab] of this.tabState.tabMap) {
        if (tab.type === wantedTabType && tab.databaseId === databaseId) {
          if (wantedTabType === UIEditorTabType.TabForTable) {
            if ((tab as TableTabContext).tableCache.name === tableName) {
              return tab;
            }
          } else {
            return tab;
          }
        }
      }
      return undefined;
    },
    addTab(tab: TabContext, setAsCurrentTab = true) {
      const tabTemp = this.getTabByDatabaseIdAndTableName(
        tab.databaseId,
        tab.type === UIEditorTabType.TabForTable
          ? tab.tableCache.name
          : undefined
      );
      if (!isUndefined(tabTemp)) {
        tab = tabTemp;
      } else {
        this.tabState.tabMap.set(tab.id, tab);
      }
      if (setAsCurrentTab) {
        this.setCurrentTab(tab.id);
      }
    },
    setCurrentTab(tabId: string) {
      if (isUndefined(this.getTabById(tabId))) {
        return;
      }
      this.tabState.currentTabId = tabId;
    },
    async fetchDatabaseList(databaseIdList: DatabaseId[]) {
      const databaseList: Database[] = [];
      for (const id of databaseIdList) {
        const database = cloneDeep(
          await useDatabaseStore().getOrFetchDatabaseById(id)
        );
        databaseList.push(database);
      }
      this.databaseList = databaseList;
    },
    async getOrFetchTableListByDatabaseId(databaseId: DatabaseId) {
      const tableList: Table[] = [];
      for (const table of this.tableList) {
        if (table.database.id === databaseId) {
          tableList.push(table);
        }
      }

      if (tableList.length === 0) {
        const tableListData = cloneDeep(
          await useTableStore().fetchTableListByDatabaseId(databaseId)
        );
        tableList.push(...tableListData);
        this.tableList.push(...tableListData);
      }
      return tableList;
    },
    // findTable try to find the table from table list including existed tables and created tables.
    findTable(tableId: TableId, tableName: string, databaseId: DatabaseId) {
      return this.tableList.find(
        (table) =>
          table.id === tableId &&
          table.name === tableName &&
          table.database.id === databaseId
      );
    },
    // createNewTable creates a temp table with databaseId.
    // Its ID is UNKNOWN_ID and name should be an unique temp name.
    createNewTable(databaseId: DatabaseId) {
      const table = unknown("TABLE");
      const index =
        this.tableList.filter((table) => table.id === UNKNOWN_ID).length + 1;
      table.name = `New Table-${index}`;
      table.database.id = databaseId;
      this.tableList.push(table);
      return table;
    },
    async postDatabaseEdit(databaseEdit: DatabaseEdit) {
      const stmt = (
        await axios.post<string>(
          `/api/database/${databaseEdit.databaseId}/edit`,
          databaseEdit
        )
      ).data;
      return stmt;
    },
  },
});
