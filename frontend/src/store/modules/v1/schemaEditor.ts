import { isUndefined, uniqueId, cloneDeep } from "lodash-es";
import { defineStore } from "pinia";
import { ComposedDatabase, emptyProject } from "@/types";
import { Engine } from "@/types/proto/v1/common";
import { ColumnConfig, TableConfig } from "@/types/proto/v1/database_service";
import {
  BranchSchema,
  DatabaseSchema,
  SchemaEditorV1State,
  Table,
  TabContext,
  SchemaEditorTabType,
  TableTabContext,
} from "@/types/v1/schemaEditor";
import { Schema } from "@/types/v1/schemaEditor/atomType";
import { useDatabaseV1Store } from "./database";

export const generateUniqueTabId = () => {
  return uniqueId();
};

const getDefaultSchemaEditorState = (): SchemaEditorV1State => {
  return {
    project: emptyProject(),
    readonly: false,
    resourceType: "database",
    resourceMap: {
      database: new Map([]),
      branch: new Map([]),
    },
    tabState: {
      tabMap: new Map<string, TabContext>([]),
      currentTabId: "",
    },
  };
};

export const useSchemaEditorV1Store = defineStore("SchemaEditorV1", {
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
    currentDatabase(): ComposedDatabase | undefined {
      if (!this.currentTab) {
        return;
      }
      if (this.resourceType === "branch") {
        const baselineDatabase = this.resourceMap.branch.get(
          this.currentTab.parentName
        )?.branch.baselineDatabase;
        if (!baselineDatabase) {
          return;
        }
        return useDatabaseV1Store().getDatabaseByName(baselineDatabase);
      } else {
        return this.resourceMap.database.get(this.currentTab.parentName)
          ?.database;
      }
    },
    currentSchemaList(): Schema[] {
      if (!this.currentTab) {
        return [] as Schema[];
      }
      return (
        this.resourceMap[this.resourceType].get(this.currentTab.parentName)
          ?.schemaList ?? []
      );
    },
  },
  actions: {
    getCurrentEngine(parentName: string) {
      const parentResouce = this.resourceMap[this.resourceType].get(parentName);
      if (!parentResouce) {
        return Engine.MYSQL;
      }
      if (this.resourceType === "database") {
        return (parentResouce as DatabaseSchema).database.instanceEntity.engine;
      } else if (this.resourceType === "branch") {
        return (parentResouce as BranchSchema).branch.engine;
      } else {
        return Engine.MYSQL;
      }
    },
    setState(state: Partial<SchemaEditorV1State>) {
      Object.assign(this, state);
    },
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
    isEmptyColumnConfig(config: ColumnConfig): boolean {
      return Object.keys(config.labels).length === 0 && !config.semanticTypeId;
    },
    removeColumnConfig(schema: Schema, table: string, column: string) {
      const tableConfig = this.getTableConfig(schema, table);
      const index = tableConfig.columnConfigs.findIndex(
        (config) => config.name === column
      );
      if (index < 0) {
        return;
      }

      const pendingUpdateTableConfig = cloneDeep(tableConfig);
      pendingUpdateTableConfig.columnConfigs.splice(index, 1);
      const tableIndex = schema.config.tableConfigs.findIndex(
        (config) => config.name === table
      );
      if (tableIndex < 0) {
        return;
      }
      if (pendingUpdateTableConfig.columnConfigs.length === 0) {
        schema.config.tableConfigs.splice(tableIndex, 1);
      } else {
        schema.config.tableConfigs[tableIndex] = pendingUpdateTableConfig;
      }
    },
    updateColumnConfig(
      schema: Schema,
      table: string,
      column: string,
      config: Partial<ColumnConfig>
    ) {
      const tableConfig = this.getTableConfig(schema, table);
      const index = tableConfig.columnConfigs.findIndex(
        (config) => config.name === column
      );
      let pendingUpdateColumnConfig = ColumnConfig.fromPartial({
        ...config,
        name: column,
      });
      if (index >= 0) {
        pendingUpdateColumnConfig = ColumnConfig.fromPartial({
          ...tableConfig.columnConfigs[index],
          ...config,
        });
      }

      if (this.isEmptyColumnConfig(pendingUpdateColumnConfig)) {
        return this.removeColumnConfig(schema, table, column);
      }

      const pendingUpdateTableConfig = cloneDeep(tableConfig);
      if (index < 0) {
        pendingUpdateTableConfig.columnConfigs.push(pendingUpdateColumnConfig);
      } else {
        pendingUpdateTableConfig.columnConfigs[index] =
          pendingUpdateColumnConfig;
      }

      const tableIndex = schema.config.tableConfigs.findIndex(
        (config) => config.name === table
      );
      if (tableIndex < 0) {
        schema.config.tableConfigs.push(pendingUpdateTableConfig);
      } else {
        schema.config.tableConfigs[tableIndex] = pendingUpdateTableConfig;
      }
    },
    getTableConfig(schame: Schema, table: string): TableConfig {
      return (
        schame.config.tableConfigs.find((config) => config.name === table) ??
        TableConfig.fromPartial({
          name: table,
        })
      );
    },
    getColumnConfig(
      schame: Schema,
      table: string,
      column: string
    ): ColumnConfig {
      const tableConfig = this.getTableConfig(schame, table);
      return (
        tableConfig.columnConfigs.find((config) => config.name === column) ??
        ColumnConfig.fromPartial({
          name: column,
        })
      );
    },
    getSchema(parentName: string, schemaId: string) {
      return this.resourceMap[this.resourceType]
        .get(parentName)
        ?.schemaList.find((schema) => schema.id === schemaId);
    },
    getOriginSchema(parentName: string, schemaId: string) {
      return this.resourceMap[this.resourceType]
        .get(parentName)
        ?.originSchemaList.find((schema) => schema.id === schemaId);
    },
    dropSchema(parentName: string, schemaId: string) {
      const schema = this.getSchema(parentName, schemaId);
      if (!schema) {
        return;
      }

      if (schema.status === "created") {
        const resource = this.resourceMap[this.resourceType].get(parentName);
        if (resource) {
          resource.schemaList =
            this.resourceMap[this.resourceType]
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
    restoreSchema(parentName: string, schemaId: string) {
      const schema = this.getSchema(parentName, schemaId);
      if (!schema) {
        return;
      }
      schema.status = "normal";
    },
    getTable(parentName: string, schemaId: string, tableId: string) {
      return this.getSchema(parentName, schemaId)?.tableList.find(
        (table) => table.id === tableId
      );
    },
    getOriginTable(parentName: string, schemaId: string, tableId: string) {
      return this.getOriginSchema(parentName, schemaId)?.tableList.find(
        (table) => table.id === tableId
      );
    },
    getTableWithTableTab(tab: TableTabContext) {
      return this.resourceMap[this.resourceType]
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
        const tableList = this.resourceMap[this.resourceType]
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
