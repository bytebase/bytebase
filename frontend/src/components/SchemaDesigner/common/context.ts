import { inject, InjectionKey, provide } from "vue";
import {
  SchemaDesignerContext,
  SchemaDesignerTabType,
  TabContext,
} from "./type";
import { isUndefined } from "lodash-es";

export const KEY = Symbol(
  "bb.schema-designer"
) as InjectionKey<SchemaDesignerContext>;

export const useSchemaDesignerContext = () => {
  return inject(KEY)!;
};

export const provideSchemaDesignerContext = (
  context: Pick<
    SchemaDesignerContext,
    | "engine"
    | "baselineMetadata"
    | "metadata"
    | "tabState"
    | "originalSchemas"
    | "editableSchemas"
  >
) => {
  const { editableSchemas, tabState } = context;

  provide(KEY, {
    ...context,

    // Tab related functions.
    getCurrentTab: (): TabContext | undefined => {
      if (isUndefined(tabState.value.currentTabId)) {
        return undefined;
      }
      return tabState.value.tabMap.get(tabState.value.currentTabId);
    },
    addTab(tab: TabContext, setAsCurrentTab = true) {
      const tabCache = Array.from(tabState.value.tabMap.values()).find(
        (item) => {
          if (item.type !== tab.type) {
            return false;
          }

          if (
            item.type === SchemaDesignerTabType.TabForTable &&
            item.schemaId === tab.schemaId &&
            item.tableId === tab.tableId
          ) {
            return true;
          }
          return false;
        }
      );

      if (tabCache !== undefined) {
        tab = {
          ...tabCache,
          ...tab,
          id: tabCache.id,
        };
      }
      tabState.value.tabMap.set(tab.id, tab);

      if (setAsCurrentTab) {
        tabState.value.currentTabId = tab.id;
      }
    },

    // Table related functions.
    getTable: (schemaId: string, tableId: string) => {
      const schema = editableSchemas.value.find((item) => item.id === schemaId);
      if (schema === undefined) {
        throw new Error(`Schema ${schemaId} not found.`);
      }

      const tableItem = schema.tableList.find((item) => item.id === tableId);
      if (tableItem === undefined) {
        throw new Error(`Table ${tableId} not found.`);
      }

      return tableItem;
    },
    dropTable: (schemaId: string, tableId: string) => {
      const tabList = Array.from(tabState.value.tabMap.values());
      for (const tab of tabList) {
        if (
          tab.type === SchemaDesignerTabType.TabForTable &&
          tab.schemaId === schemaId &&
          tab.tableId === tableId
        ) {
          tabState.value.tabMap.delete(tab.id);
          if (tabState.value.currentTabId === tab.id) {
            tabState.value.currentTabId = undefined;
          }
        }
      }

      const schemaItem = editableSchemas.value.find(
        (item) => item.id === schemaId
      );
      if (schemaItem === undefined) {
        return;
      }

      schemaItem.tableList = schemaItem.tableList.filter(
        (item) => item.id !== tableId
      );
    },
  });
};
