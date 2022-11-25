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
  Column,
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
      if (isUndefined(state.tabState.currentTabId)) {
        return undefined;
      }
      return state.tabState.tabMap.get(state.tabState.currentTabId);
    },
    tabList(state) {
      return Array.from(state.tabState.tabMap.values());
    },
  },
  actions: {
    addTab(tab: TabContext, setAsCurrentTab = true) {
      const tabCache = this.tabList.find((item) => {
        if (item.type !== tab.type) {
          return false;
        }

        if (
          item.type === UIEditorTabType.TabForDatabase &&
          item.databaseId === tab.databaseId
        ) {
          return true;
        }
        if (
          item.type === UIEditorTabType.TabForTable &&
          item.table === (tab as TableTabContext).table
        ) {
          return true;
        }
        return false;
      });

      if (tabCache !== undefined) {
        tab = tabCache;
      } else {
        this.tabState.tabMap.set(tab.id, tab);
      }

      if (setAsCurrentTab) {
        this.setCurrentTab(tab.id);
      }
    },
    setCurrentTab(tabId: string) {
      if (isUndefined(this.tabState.tabMap.get(tabId))) {
        this.tabState.currentTabId = undefined;
      } else {
        this.tabState.currentTabId = tabId;
      }
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
      table.columnList = [
        {
          id: UNKNOWN_ID,
          name: "id",
          type: "int",
          nullable: false,
          comment: "ID",
        } as Column,
      ];
      this.tableList.push(table);
      return table;
    },
    dropTable(table: Table) {
      const index = this.tableList.findIndex((item) => item === table);
      if (table.id === UNKNOWN_ID) {
        this.tableList.splice(index, 1);
      } else {
        this.droppedTableList.push(table);
      }

      const tab = this.tabList.find(
        (tab) => tab.type === UIEditorTabType.TabForTable && tab.table === table
      );
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
