import { isUndefined, uniqueId } from "lodash-es";
import { defineStore } from "pinia";
import { ComposedDatabase } from "@/types";
import {
  SchemaEditorV1State,
  Table,
  TabContext,
  SchemaEditorTabType,
  TableTabContext,
} from "@/types/schemaEditorV1";

export const generateUniqueTabId = () => {
  return uniqueId();
};

const getDefaultSchemaEditorState = (): SchemaEditorV1State => {
  return {
    mode: "database",
    resourceMap: {
      database: new Map(),
      branch: new Map(),
    },
    tabState: {
      tabMap: new Map<string, TabContext>(),
      currentTabId: "",
    },
  };
};

export const useSchemaEditorStore = defineStore("SchemaEditorV1", {
  state: (): SchemaEditorV1State => {
    return getDefaultSchemaEditorState();
  },
  getters: {
    currentTab(): TabContext | undefined {
      if (isUndefined(this.tabState.currentTabId)) {
        return undefined;
      }
      return this.tabState.tabMap.get(this.tabState.currentTabId);
    },
    tabList(): TabContext[] {
      return Array.from(this.tabState.tabMap.values());
    },
    databaseList(): ComposedDatabase[] {
      return Array.from(this.resourceMap["database"].values()).map(
        (databaseSchema) => databaseSchema.database
      );
    },
  },
  actions: {
    addTab(tab: TabContext, setAsCurrentTab = true) {
      const tabCache = this.tabList.find((item) => {
        if (
          item.type === tab.type &&
          item.parentName === tab.parentName &&
          (item.type === SchemaEditorTabType.TabForDatabase ||
            (item.type === SchemaEditorTabType.TabForTable &&
              item.tableId === (tab as TableTabContext).tableId))
        ) {
          return true;
        }
        return false;
      });

      if (tabCache !== undefined) {
        tab = {
          ...tabCache,
          ...tab,
          id: tabCache.id,
        };
      }
      this.tabState.tabMap.set(tab.id, tab);

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
    findTab(parentName: string, tableId?: string) {
      let tabType = SchemaEditorTabType.TabForDatabase;
      if (tableId !== undefined) {
        tabType = SchemaEditorTabType.TabForTable;
      }

      const tab = this.tabList.find((tab) => {
        if (
          tab.type === tabType &&
          tab.parentName === parentName &&
          (tab.type === SchemaEditorTabType.TabForDatabase ||
            (tab.type === SchemaEditorTabType.TabForTable &&
              tab.tableId === tableId))
        ) {
          return true;
        }
        return false;
      });

      return tab;
    },
    getSchema(parentName: string, schemaId: string) {
      return this.resourceMap[this.mode]
        .get(parentName)
        ?.schemaList.find((schema) => schema.id === schemaId);
    },
    getOriginSchema(parentName: string, schemaId: string) {
      return this.resourceMap[this.mode]
        .get(parentName)
        ?.originSchemaList.find((schema) => schema.id === schemaId);
    },
    dropSchema(parentName: string, schemaId: string) {
      const schema = this.getSchema(parentName, schemaId);
      if (!schema) {
        return;
      }

      if (schema.status === "created") {
        const resource = this.resourceMap[this.mode].get(parentName);
        if (resource) {
          resource.schemaList =
            this.resourceMap[this.mode]
              .get(parentName)
              ?.schemaList.filter((schema) => schema.id !== schemaId) || [];

          // Close related tabs.
          for (const tab of this.tabList) {
            if (tab.parentName !== parentName) {
              continue;
            }

            if (
              tab.type === SchemaEditorTabType.TabForTable &&
              tab.schemaId === schemaId
            ) {
              this.closeTab(tab.id);
            }
          }
        }
      } else {
        schema.status = "dropped";
      }
    },
    restoreSchema(databaseId: string, schemaId: string) {
      const schema = this.getSchema(databaseId, schemaId);
      if (!schema) {
        return;
      }
      schema.status = "normal";
    },
    getTable(databaseId: string, schemaId: string, tableId: string) {
      return this.getSchema(databaseId, schemaId)?.tableList.find(
        (table) => table.id === tableId
      );
    },
    getOriginTable(databaseId: string, schemaId: string, tableId: string) {
      return this.getOriginSchema(databaseId, schemaId)?.tableList.find(
        (table) => table.id === tableId
      );
    },
    getTableWithTableTab(tab: TableTabContext) {
      return this.resourceMap[this.mode]
        .get(tab.parentName)
        ?.schemaList.find((schema) => schema.id === tab.schemaId)
        ?.tableList?.find((table) => table.id === tab.tableId);
    },
    dropTable(parentName: string, schemaId: string, tableId: string) {
      const table = this.getTable(parentName, schemaId, tableId);
      if (!table) {
        return;
      }

      // Remove table record and close tab for created table.
      if (table.status === "created") {
        const tableList = this.resourceMap[this.mode]
          .get(parentName)
          ?.schemaList.find((schema) => schema.id === schemaId)
          ?.tableList as Table[];
        const index = tableList.findIndex((item) => item.id === table.id);
        tableList.splice(index, 1);
        const tab = this.findTab(parentName, table.id);
        if (tab) {
          this.closeTab(tab.id);
        }
      } else {
        table.status = "dropped";
      }
    },
    restoreTable(parentName: string, schemaId: string, tableId: string) {
      const table = this.getTable(parentName, schemaId, tableId);
      if (!table) {
        return;
      }
      table.status = "normal";
    },
  },
});
