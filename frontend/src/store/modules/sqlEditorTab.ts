import { useLocalStorage, watchThrottled } from "@vueuse/core";
import { pick } from "lodash-es";
import { defineStore } from "pinia";
import { computed, reactive, ref, toRef, watch } from "vue";
import "@/types";
import { SQLEditorTab } from "@/types/sqlEditorTab";
import {
  getDefaultTab,
  INITIAL_TAB,
  isTempTab,
  isSimilarTab,
  WebStorageHelper,
  isDisconnectedTab,
  defaultSQLEditorTab,
} from "@/utils";
import { useWebTerminalV1Store } from "./v1";
import { useWorkSheetStore } from "./v1/worksheet";

const LOCAL_STORAGE_KEY_PREFIX = "bb.sql-editor-tab";
const KEYS = {
  tabIdList: "tab-id-list",
  currentTabId: "current-tab-id",
  tab: (id: string) => `tab.${id}`,
};

// Only store the core fields of a tab.
// Don't store anything which might be too large.
const PERSISTENT_TAB_FIELDS = [
  "id",
  "title",
  "connection",
  "statement",
  "mode",
  "sheet",
  "status",
  "batchContext",
] as const;
type PersistentTab = Pick<SQLEditorTab, typeof PERSISTENT_TAB_FIELDS[number]>;

const keyPrefixWithProject = (project: string) => {
  return `${LOCAL_STORAGE_KEY_PREFIX}.projects/${project || "ALL"}`;
};
const getStorage = (project: string) => {
  return new WebStorageHelper(keyPrefixWithProject(project), localStorage);
};

export const useSQLEditorTabsByProject = (project: string) => {
  const storage = getStorage(project);
  // We store the tabIdList and the tabs separately.
  // This index-entity modeling enables us to update one tab entity at a time,
  // and reduce the performance costing while writing localStorage.
  // `tabIdList` stores all `tab.id`s in the project
  const tabIdList = computed({
    get() {
      return storage.load<string[]>(KEYS.tabIdList, []);
    },
    set(idList) {
      storage.save<string[]>(KEYS.tabIdList, idList);
    },
  });
  // `currentTabId` stores current `tab.id` in the project
  // default to empty string, which means no tab is selected
  const currentTabId = computed({
    get() {
      return storage.load<string>(KEYS.currentTabId, "");
    },
    set(id) {
      storage.save<string>(KEYS.currentTabId, id);
    },
  });
  // `tabs` stores all PersistentTabs in the project
  const tabs = ref(new Map<string, SQLEditorTab>());

  const tabById = (id: string) => {
    return tabs.value.get(id);
  };
  const tabList = computed(() => {
    return tabIdList.value.map(tabById);
  });
  const currentTab = computed(() => {
    return tabs.value.get(currentTabId.value);
  });

  // actions
  /**
   *
   * @param payload
   * @param beside `true` to add the tab beside currentTab, `false` to add the tab to the last, default to `false`
   * @returns
   */
  const add = (payload?: Partial<SQLEditorTab>, beside = false) => {
    const newTab = reactive<SQLEditorTab>({
      ...defaultSQLEditorTab(),
      ...payload,
    });

    const { id } = newTab;
    const position = tabIdList.value.indexOf(currentTabId.value ?? "");
    if (beside && position >= 0) {
      tabIdList.value.splice(position + 1, 0, id);
    } else {
      tabIdList.value.push(id);
    }
    currentTabId.value = id;
    tabs.value.set(id, newTab);

    watchTab(newTab, true /* immediate */);
    throw new Error("not implemented");
  };
  const remove = (tab: SQLEditorTab) => {
    const { id } = tab;
    const position = tabIdList.value.indexOf(id);
    if (position < 0) return;
    tabIdList.value.splice(position, 1);
    tabs.value.delete(id);
    storage.remove(KEYS.tab(id));

    if (tab.mode === "ADMIN") {
      useWebTerminalV1Store().clearQueryStateByTab(id);
    }
  };
  const update = (id: string, payload: Partial<SQLEditorTab>) => {
    const tab = tabById(id);
    if (!tab) return;
    Object.assign(tab, payload);
  };
  const updateCurrent = (payload: Partial<SQLEditorTab>) => {
    const id = currentTabId.value;
    if (!id) return;
    update(id, payload);
  };
  const setCurrent = (id: string) => {
    currentTabId.value = id;
  };
  // clean persistent tabs that are not in the `tabIdList` anymore
  const _cleanup = (tabIdList: string[]) => {
    const prefix = `${keyPrefixWithProject(project)}.tab.`;
    const keys = storage.keys().filter((key) => key.startsWith(prefix));
    keys.forEach((key) => {
      const id = key.substring(prefix.length);
      if (tabIdList.indexOf(id) < 0) {
        storage.remove(KEYS.tab(id));
      }
    });
  };
  // watch the field changes of a tab, store it to localStorage
  // when needed, but not to frequently (for performance consideration)
  const watchTab = (tab: SQLEditorTab, immediate: boolean) => {
    const dirtyFields = [
      () => tab.title,
      () => tab.sheet,
      () => tab.statement,
      () => tab.connection,
      () => tab.batchContext,
    ];
    // set `tab.status` to "DIRTY" when it's changed
    watch(dirtyFields, () => {
      tab.status = "DIRTY";
    });

    // Use a throttled watcher to reduce the performance overhead when writing.
    watchThrottled(
      () => pick(tab, ...PERSISTENT_TAB_FIELDS) as PersistentTab,
      (persistentTab) => {
        storage.save<PersistentTab>(KEYS.tab(persistentTab.id), persistentTab);
      },
      { deep: true, immediate, throttle: 100, trailing: true }
    );
  };
};

export const useSQLEditorTabStore = defineStore("sql-editor-tab", () => {
  // states
  const project = ref<string>(""); // empty to "ALL" projects for high-privileged users

  // const isDisconnected = computed((): boolean => {
  //   return isDisconnectedTab(currentTab.value);
  // });

  // // actions
  // const addTab = (payload?: AnyTabInfo, beside = false) => {
  //   const newTab = reactive<TabInfo>({
  //     ...getDefaultTab(),
  //     ...payload,
  //   });

  //   const { id } = newTab;
  //   const index = tabIdList.value.indexOf(currentTabId.value ?? "");
  //   if (beside && index >= 0) {
  //     tabIdList.value.splice(index + 1, 0, id);
  //   } else {
  //     tabIdList.value.push(id);
  //   }
  //   currentTabId.value = id;
  //   tabs.value.set(id, newTab);

  //   watchTab(newTab, false);
  // };
  // const removeTab = (tab: TabInfo) => {
  //   const id = tab.id;
  //   const index = tabIdList.value.indexOf(id);
  //   if (index >= 0) {
  //     tabIdList.value.splice(index, 1);
  //     tabs.value.delete(id);
  //     storage.remove(KEYS.tab(id));

  //     if (tab.mode === TabMode.Admin) {
  //       useWebTerminalV1Store().clearQueryStateByTab(tab);
  //     }
  //   }
  // };
  // const updateCurrentTab = (payload: AnyTabInfo) => {
  //   updateTab(currentTabId.value ?? "", payload);
  // };
  // const updateTab = (tabId: string, payload: AnyTabInfo) => {
  //   const tab = getTabById(tabId);
  //   Object.assign(tab, payload);
  // };
  // const setCurrentTabId = (id: string) => {
  //   currentTabId.value = id;
  // };
  // const selectOrAddSimilarTab = (
  //   tab: CoreTabInfo,
  //   beside = false,
  //   defaultName?: string
  // ) => {
  //   if (isDisconnected.value) {
  //     if (defaultName) {
  //       currentTab.value.name = defaultName;
  //     }
  //     return;
  //   }
  //   if (isSimilarTab(tab, currentTab.value)) {
  //     return;
  //   }
  //   const similarTab = tabList.value.find((tmp) => isSimilarTab(tmp, tab));
  //   if (similarTab) {
  //     setCurrentTabId(similarTab.id);
  //   } else {
  //     addTab(tab, beside);
  //     if (defaultName) {
  //       currentTab.value.name = defaultName;
  //     }
  //   }
  // };
  // const selectOrAddTempTab = (newTab?: AnyTabInfo) => {
  //   if (isDisconnected.value) {
  //     return;
  //   }
  //   if (isTempTab(currentTab.value)) {
  //     return;
  //   }
  //   const tempTab = tabList.value.find(isTempTab);
  //   if (tempTab) {
  //     setCurrentTabId(tempTab.id);
  //   } else {
  //     addTab(newTab);
  //   }
  // };
  // const _cleanup = (tabIdList: string[]) => {
  //   const prefix = `${LOCAL_STORAGE_KEY_PREFIX}.tab.`;
  //   const keys = storage.keys().filter((key) => key.startsWith(prefix));
  //   keys.forEach((key) => {
  //     const id = key.substring(prefix.length);
  //     if (tabIdList.indexOf(id) < 0) {
  //       storage.remove(KEYS.tab(id));
  //     }
  //   });
  // };

  // // watchers
  // const watchTab = (tab: TabInfo, immediate: boolean) => {
  //   const dirtyFields = [
  //     () => tab.name,
  //     () => tab.sheetName,
  //     () => tab.statement,
  //     () => tab.connection,
  //     () => tab.batchQueryContext,
  //   ];
  //   watch(dirtyFields, () => {
  //     tab.isFreshNew = false;
  //   });

  //   // Use a throttled watcher to reduce the performance overhead when writing.
  //   watchThrottled(
  //     () => pick(tab, ...PERSISTENT_TAB_FIELDS),
  //     (tabPartial: PersistentTabInfo) => {
  //       storage.save(KEYS.tab(tabPartial.id), tabPartial);
  //     },
  //     { deep: true, immediate, throttle: 100, trailing: true }
  //   );
  // };

  // // Load session from local storage.
  // // Reset if failed.
  // const init = () => {
  //   // Load tabIdList and currentTabId
  //   tabIdList.value = storage.load(KEYS.tabIdList, []);
  //   currentTabId.value = storage.load(KEYS.currentTabId, INITIAL_TAB.id);
  //   if (tabIdList.value.indexOf(currentTabId.value) < 0) {
  //     // currentTabId is not in tabIdList accidentally
  //     // fallback to the first tab
  //     currentTabId.value = tabIdList.value[0];
  //   }

  //   // Fallback to the initial tab if tabIdList is empty.
  //   if (tabIdList.value.length === 0) {
  //     // tabIdList is empty accidentally
  //     tabIdList.value = [INITIAL_TAB.id];
  //   }

  //   // Load tab details
  //   tabIdList.value.forEach((id) => {
  //     const tabPartial = storage.load<PersistentTabInfo | undefined>(
  //       KEYS.tab(id),
  //       undefined
  //     );
  //     maybeMigrateLegacyTab(storage, tabPartial);
  //     // Use a stored tab info if possible.
  //     // Fallback to getDefaultTab() otherwise.
  //     const tab = reactive<TabInfo>({
  //       ...getDefaultTab(),
  //       ...tabPartial,
  //       id,
  //     });
  //     // Legacy id support
  //     tab.connection.databaseId = String(tab.connection.databaseId);
  //     tab.connection.instanceId = String(tab.connection.instanceId);
  //     watchTab(tab, false);
  //     tabs.value.set(id, tab);
  //   });

  //   // Fetch opening sheets if needed
  //   const sheetV1Store = useWorkSheetStore();
  //   tabList.value.forEach((tab) => {
  //     if (tab.sheetName) {
  //       sheetV1Store.getOrFetchSheetByName(tab.sheetName);
  //     }
  //   });

  //   // Clean up unused stored tabs
  //   _cleanup(tabIdList.value);

  //   watch(
  //     currentTabId,
  //     (id) => {
  //       storage.save(KEYS.currentTabId, id);
  //     },
  //     { immediate: true }
  //   );
  //   watch(
  //     tabIdList,
  //     (idList) => {
  //       storage.save(KEYS.tabIdList, idList);
  //     },
  //     { deep: true, immediate: true }
  //   );
  // };
  // init();

  // const reset = () => {
  //   storage.clear();
  //   init();
  // };

  // // exposure
  // return {
  //   tabIdList,
  //   tabList,
  //   currentTabId,
  //   currentTab,
  //   isDisconnected,
  //   getTabById,
  //   addTab,
  //   removeTab,
  //   updateTab,
  //   updateCurrentTab,
  //   setCurrentTabId,
  //   selectOrAddTempTab,
  //   selectOrAddSimilarTab,
  //   reset,
  // };
  return {};
});

// export const useCurrentTab = () => {
//   const store = useSQLEditorTabStore();
//   return toRef(store, "currentTab");
// };

// const maybeMigrateLegacyTab = (
//   storage: WebStorageHelper,
//   tab: PersistentTabInfo | undefined
// ) => {
//   if (!tab) return;
//   const { connection } = tab;
//   if (
//     typeof connection.databaseId === "number" ||
//     typeof connection.instanceId === "number"
//   ) {
//     connection.databaseId = String(connection.databaseId);
//     connection.instanceId = String(connection.instanceId);
//     storage.save(KEYS.tab(tab.id), tab);
//   }
//   return tab;
// };

// export const isTabClosable = (tab: TabInfo) => {
//   const { tabList } = useSQLEditorTabStore();

//   if (tabList.length > 1) {
//     // Not the only one tab
//     return true;
//   }
//   if (tabList.length === 1) {
//     // It's the only one tab, and it's closable if it's a sheet tab
//     return !!tab.sheetName;
//   }
//   return false;
// };
