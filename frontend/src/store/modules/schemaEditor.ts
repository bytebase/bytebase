import axios from "axios";
import { defineStore } from "pinia";
import { cloneDeep, isUndefined, uniqueId } from "lodash-es";
import {
  DatabaseId,
  Database,
  TabContext,
  SchemaEditorState,
  SchemaEditorTabType,
  TableTabContext,
  DatabaseEdit,
  ResourceObject,
} from "@/types";
import { DatabaseEditResult } from "@/types/schemaEditor";
import {
  convertSchemaMetadataList,
  convertSchemaMetadataToSchema,
  Table,
} from "@/types/schemaEditor/atomType";
import { SchemaMetadata } from "@/types/proto/store/database";
import { useDatabaseStore, useDBSchemaStore } from ".";

export const generateUniqueTabId = () => {
  return uniqueId();
};

const getDefaultSchemaEditorState = (): SchemaEditorState => {
  return {
    tabState: {
      tabMap: new Map<string, TabContext>(),
      currentTabId: "",
    },
    databaseSchemaById: new Map(),
  };
};

function convertDatabaseEditResult(
  databaseEditResult: ResourceObject
): DatabaseEditResult {
  return {
    ...databaseEditResult.attributes,
  } as any as DatabaseEditResult;
}

export const useSchemaEditorStore = defineStore("SchemaEditor", {
  state: (): SchemaEditorState => {
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
    databaseList(): Database[] {
      return Array.from(this.databaseSchemaById.values()).map(
        (databaseSchema) => databaseSchema.database
      );
    },
  },
  actions: {
    addTab(tab: TabContext, setAsCurrentTab = true) {
      const tabCache = this.tabList.find((item) => {
        if (item.type !== tab.type) {
          return false;
        }

        if (
          item.type === SchemaEditorTabType.TabForDatabase &&
          item.databaseId === tab.databaseId
        ) {
          return true;
        }
        if (
          item.type === SchemaEditorTabType.TabForTable &&
          item.databaseId === tab.databaseId &&
          item.tableId === (tab as TableTabContext).tableId
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
    findTab(databaseId: DatabaseId, tableId?: string) {
      let tabType = SchemaEditorTabType.TabForDatabase;
      if (tableId !== undefined) {
        tabType = SchemaEditorTabType.TabForTable;
      }

      const tab = this.tabList.find((tab) => {
        if (tab.type !== tabType || tab.databaseId !== databaseId) {
          return false;
        }

        if (tab.type === SchemaEditorTabType.TabForDatabase) {
          return true;
        } else if (
          tab.type === SchemaEditorTabType.TabForTable &&
          tab.tableId === tableId
        ) {
          return true;
        }

        return false;
      });

      return tab;
    },
    async fetchDatabaseList(databaseIdList: DatabaseId[]) {
      const databaseList: Database[] = [];
      for (const id of databaseIdList) {
        const database = cloneDeep(
          await useDatabaseStore().getOrFetchDatabaseById(id)
        );
        databaseList.push(database);
        this.databaseSchemaById.set(database.id, {
          database: database,
          schemaList: [],
          originSchemaList: [],
        });
      }
      return databaseList;
    },
    async fetchSchemaListByDatabaseId(
      databaseId: DatabaseId,
      skipCache = false
    ) {
      const database = useDatabaseStore().getDatabaseById(databaseId);
      const schemaMetadataList =
        await useDBSchemaStore().getOrFetchSchemaListByDatabaseId(
          databaseId,
          skipCache
        );
      const schemaList = convertSchemaMetadataList(schemaMetadataList);
      if (schemaList.length === 0 && database.instance.engine === "MYSQL") {
        schemaList.push(
          convertSchemaMetadataToSchema(SchemaMetadata.fromPartial({}))
        );
      }

      this.databaseSchemaById.set(databaseId, {
        database: database,
        schemaList: schemaList,
        originSchemaList: cloneDeep(schemaList),
      });

      return this.databaseSchemaById.get(databaseId)!.schemaList;
    },
    getSchema(databaseId: DatabaseId, schemaId: string) {
      return this.databaseSchemaById
        .get(databaseId)
        ?.schemaList.find((schema) => schema.id === schemaId);
    },
    getOriginSchema(databaseId: DatabaseId, schemaId: string) {
      return this.databaseSchemaById
        .get(databaseId)
        ?.originSchemaList.find((schema) => schema.id === schemaId);
    },
    dropSchema(databaseId: DatabaseId, schemaId: string) {
      const schema = this.getSchema(databaseId, schemaId);
      if (!schema) {
        return;
      }

      if (schema.status === "created") {
        const databaseSchema = this.databaseSchemaById.get(databaseId);
        if (databaseSchema) {
          databaseSchema.schemaList =
            this.databaseSchemaById
              .get(databaseId)
              ?.schemaList.filter((schema) => schema.id !== schemaId) || [];

          // Close related tabs.
          for (const tab of this.tabList) {
            if (tab.databaseId !== databaseId) {
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
    restoreSchema(databaseId: DatabaseId, schemaId: string) {
      const schema = this.getSchema(databaseId, schemaId);
      if (!schema) {
        return;
      }

      schema.status = "normal";
    },
    getTable(databaseId: DatabaseId, schemaId: string, tableId: string) {
      return this.getSchema(databaseId, schemaId)?.tableList.find(
        (table) => table.id === tableId
      );
    },
    getOriginTable(databaseId: DatabaseId, schemaId: string, tableId: string) {
      return this.getOriginSchema(databaseId, schemaId)?.tableList.find(
        (table) => table.id === tableId
      );
    },
    getTableWithTableTab(tab: TableTabContext) {
      return this.databaseSchemaById
        .get(tab.databaseId)
        ?.schemaList.find((schema) => schema.id === tab.schemaId)
        ?.tableList?.find((table) => table.id === tab.tableId);
    },
    dropTable(databaseId: DatabaseId, schemaId: string, tableId: string) {
      const table = this.getTable(databaseId, schemaId, tableId);
      if (!table) {
        return;
      }

      // Remove table record and close tab for created table.
      if (table.status === "created") {
        const tableList = this.databaseSchemaById
          .get(databaseId)
          ?.schemaList.find((schema) => schema.id === schemaId)
          ?.tableList as Table[];
        const index = tableList.findIndex((item) => item.id === table.id);
        tableList.splice(index, 1);
        const tab = this.findTab(databaseId, table.id);
        if (tab) {
          this.closeTab(tab.id);
        }
      } else {
        table.status = "dropped";
      }
    },
    restoreTable(databaseId: DatabaseId, schemaId: string, tableId: string) {
      const table = this.getTable(databaseId, schemaId, tableId);
      if (!table) {
        return;
      }

      table.status = "normal";
    },
    async postDatabaseEdit(databaseEdit: DatabaseEdit) {
      const resData = (
        await axios.post(
          `/api/database/${databaseEdit.databaseId}/edit`,
          databaseEdit
        )
      ).data;
      const databaseEditResult = convertDatabaseEditResult(resData.data);
      return databaseEditResult;
    },
  },
});
