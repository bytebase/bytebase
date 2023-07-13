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
    "engine" | "baselineMetadata" | "metadata" | "tabState"
  >
) => {
  const { metadata, tabState } = context;

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
            item.schema === tab.schema &&
            item.table === tab.table
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
    // Schema related functions.
    dropSchema: (schema: string) => {
      const tabList = Array.from(tabState.value.tabMap.values());
      for (const tab of tabList) {
        if (tab.schema === schema) {
          tabState.value.tabMap.delete(tab.id);
          if (tabState.value.currentTabId === tab.id) {
            tabState.value.currentTabId = undefined;
          }
        }
      }

      metadata.value.schemas = metadata.value.schemas.filter(
        (item) => item.name !== schema
      );
    },

    // Table related functions.
    dropTable: (schema: string, table: string) => {
      const tabList = Array.from(tabState.value.tabMap.values());
      for (const tab of tabList) {
        if (
          tab.type === SchemaDesignerTabType.TabForTable &&
          tab.schema === schema &&
          tab.table === table
        ) {
          tabState.value.tabMap.delete(tab.id);
          if (tabState.value.currentTabId === tab.id) {
            tabState.value.currentTabId = undefined;
          }
        }
      }

      const schemaItem = metadata.value.schemas.find(
        (item) => item.name === schema
      );
      if (schemaItem === undefined) {
        return;
      }

      schemaItem.tables = schemaItem.tables.filter(
        (item) => item.name !== table
      );
    },
  });
};
