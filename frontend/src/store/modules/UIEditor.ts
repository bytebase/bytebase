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
    droppedTableList: [],
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
    // * tableName is used to position those new tables with UNKNOWN_ID.
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
    closeTab(tabId: string) {
      const tabList = Array.from(this.tabState.tabMap.values());
      const tabIndex = tabList.findIndex((tab) => tab.id === tabId);
      // Find next tab for showing.
      if (this.tabState.currentTabId === tabId) {
        let nextTabIndex = -1;
        if (tabIndex === 0) {
          nextTabIndex = 1;
        } else {
          nextTabIndex = tabIndex - 1;
        }
        const nextTab = tabList[nextTabIndex];
        if (nextTab) {
          this.setCurrentTab(nextTab.id);
        } else {
          this.setCurrentTab("");
        }
      }
      this.tabState.tabMap.delete(tabId);
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
      return databaseList;
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
    // findTable tries to find the table from the table list including existing tables and created tables.
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
      const databaseTableList = this.tableList.filter(
        (table) => table.database.id === databaseId
      );
      const databaseTableNameList = databaseTableList.map(
        (table) => table.name
      );
      let tableNameUniqueIndex =
        databaseTableList.filter((table) => table.id === UNKNOWN_ID).length + 1;
      let tableName = `untitled_table_${tableNameUniqueIndex}`;
      while (databaseTableNameList.includes(tableName)) {
        tableNameUniqueIndex++;
        tableName = `untitled_table_${tableNameUniqueIndex}`;
      }

      const database = useDatabaseStore().getDatabaseById(databaseId);
      table.id = UNKNOWN_ID;
      table.name = tableName;
      table.database = database;
      this.tableList.push(table);
      return table;
    },
    dropTable(databaseId: DatabaseId, tableId: TableId, tableName: string) {
      const index = this.tableList.findIndex(
        (table) =>
          table.database.id === databaseId &&
          table.id === tableId &&
          table.name === tableName
      );
      const table = this.tableList[index];
      if (table.id === UNKNOWN_ID) {
        this.tableList.splice(index, 1);
      } else {
        this.droppedTableList.push(table);
      }

      const tab = this.getTabByDatabaseIdAndTableName(databaseId, tableName);
      if (tab) {
        this.closeTab(tab.id);
      }
    },
    restoreTable(databaseId: DatabaseId, tableId: TableId, tableName: string) {
      const index = this.droppedTableList.findIndex(
        (table) =>
          table.database.id === databaseId &&
          table.id === tableId &&
          table.name === tableName
      );
      this.droppedTableList.splice(index, 1);
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
