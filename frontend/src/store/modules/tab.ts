import { watchThrottled } from "@vueuse/core";
import { pick } from "lodash-es";
import { defineStore } from "pinia";
import { computed, reactive, ref, toRef, watch } from "vue";
import { TabInfo, CoreTabInfo, AnyTabInfo, TabMode } from "@/types";
import {
  getDefaultTab,
  INITIAL_TAB,
  isTempTab,
  isSimilarTab,
  WebStorageHelper,
  isDisconnectedTab,
} from "@/utils";
import { useFilterStore } from "./filter";
import { useWebTerminalV1Store } from "./v1";
import { useWorkSheetStore } from "./v1/worksheet";

const LOCAL_STORAGE_KEY_PREFIX = "bb.sql-editor.tab-list";
const KEYS = {
  tabIdList: "tab-id-list",
  currentTabId: "current-tab-id",
  tab: (id: string) => `tab.${id}`,
};

// Only store the core fields of a tab.
// Don't store anything which might be too large.
const PERSISTENT_TAB_FIELDS = [
  "id",
  "name",
  "connection",
  "isSaved",
  "savedAt",
  "statement",
  "sheetName",
  "mode",
  "batchQueryContext",
] as const;
type PersistentTabInfo = Pick<TabInfo, typeof PERSISTENT_TAB_FIELDS[number]>;

export const useTabStore = defineStore("tab", () => {
  const storage = new WebStorageHelper("bb.sql-editor.tab-list", localStorage);
  const { filter } = useFilterStore();

  // states
  // We store the tabIdList and the tabs separately.
  // This index-entity modeling enables us to update one tab entity at a time,
  // and reduce the performance costing while writing localStorage.
  const tabs = ref(new Map<string, TabInfo>());
  const tabIdList = ref<string[]>([]);
  const currentTabId = ref<string>();

  // getters
  const getTabById = (id: string) => {
    return tabs.value.get(id) ?? getDefaultTab();
  };
  const tabList = computed((): TabInfo[] => tabIdList.value.map(getTabById));
  const currentTab = computed((): TabInfo => {
    return getTabById(currentTabId.value ?? "");
  });
  const isDisconnected = computed((): boolean => {
    return isDisconnectedTab(currentTab.value);
  });

  // actions
  const addTab = (payload?: AnyTabInfo, beside = false) => {
    const newTab = reactive<TabInfo>({
      ...getDefaultTab(),
      ...payload,
    });

    const { id } = newTab;
    const index = tabIdList.value.indexOf(currentTabId.value ?? "");
    if (beside && index >= 0) {
      tabIdList.value.splice(index + 1, 0, id);
    } else {
      tabIdList.value.push(id);
    }
    currentTabId.value = id;
    tabs.value.set(id, newTab);

    watchTab(newTab, false);
  };
  const removeTab = (tab: TabInfo) => {
    const id = tab.id;
    const index = tabIdList.value.indexOf(id);
    if (index >= 0) {
      tabIdList.value.splice(index, 1);
      tabs.value.delete(id);
      storage.remove(KEYS.tab(id));

      if (tab.mode === TabMode.Admin) {
        useWebTerminalV1Store().clearQueryStateByTab(tab);
      }
    }
  };
  const updateCurrentTab = (payload: AnyTabInfo) => {
    updateTab(currentTabId.value ?? "", payload);
  };
  const updateTab = (tabId: string, payload: AnyTabInfo) => {
    const tab = getTabById(tabId);
    Object.assign(tab, payload);
  };
  const setCurrentTabId = (id: string) => {
    currentTabId.value = id;
  };
  const selectOrAddSimilarTab = (
    tab: CoreTabInfo,
    beside = false,
    defaultName?: string
  ) => {
    if (isDisconnected.value) {
      if (defaultName) {
        currentTab.value.name = defaultName;
      }
      return;
    }
    if (isSimilarTab(tab, currentTab.value)) {
      return;
    }
    const similarTab = tabList.value.find((tmp) => isSimilarTab(tmp, tab));
    if (similarTab) {
      setCurrentTabId(similarTab.id);
    } else {
      addTab(tab, beside);
      if (defaultName) {
        currentTab.value.name = defaultName;
      }
    }
  };
  const selectOrAddTempTab = (newTab?: AnyTabInfo) => {
    if (isDisconnected.value) {
      return;
    }
    if (isTempTab(currentTab.value)) {
      return;
    }
    const tempTab = tabList.value.find(isTempTab);
    if (tempTab) {
      setCurrentTabId(tempTab.id);
    } else {
      addTab(newTab);
    }
  };
  const _cleanup = (tabIdList: string[]) => {
    const prefix = `${LOCAL_STORAGE_KEY_PREFIX}.tab.`;
    const keys = storage.keys().filter((key) => key.startsWith(prefix));
    keys.forEach((key) => {
      const id = key.substring(prefix.length);
      if (tabIdList.indexOf(id) < 0) {
        storage.remove(KEYS.tab(id));
      }
    });
  };

  // watchers
  const watchTab = (tab: TabInfo, immediate: boolean) => {
    const dirtyFields = [
      () => tab.name,
      () => tab.sheetName,
      () => tab.statement,
      () => tab.connection,
      () => tab.batchQueryContext,
    ];
    watch(dirtyFields, () => {
      tab.isFreshNew = false;
    });

    // Use a throttled watcher to reduce the performance overhead when writing.
    watchThrottled(
      () => pick(tab, ...PERSISTENT_TAB_FIELDS),
      (tabPartial: PersistentTabInfo) => {
        storage.save(KEYS.tab(tabPartial.id), tabPartial);
      },
      { deep: true, immediate, throttle: 100, trailing: true }
    );
  };

  // Load session from local storage.
  // Reset if failed.
  const init = () => {
    if (filter.project || filter.database) {
      // Don't load tabs from local storage if the filter is specified.
    } else {
      // Load tabIdList and currentTabId
      tabIdList.value = storage.load(KEYS.tabIdList, []);
      currentTabId.value = storage.load(KEYS.currentTabId, INITIAL_TAB.id);
      if (tabIdList.value.indexOf(currentTabId.value) < 0) {
        // currentTabId is not in tabIdList accidentally
        // fallback to the first tab
        currentTabId.value = tabIdList.value[0];
      }
    }

    // Fallback to the initial tab if tabIdList is empty.
    if (tabIdList.value.length === 0) {
      // tabIdList is empty accidentally
      tabIdList.value = [INITIAL_TAB.id];
    }

    // Load tab details
    tabIdList.value.forEach((id) => {
      const tabPartial = storage.load<PersistentTabInfo | undefined>(
        KEYS.tab(id),
        undefined
      );
      maybeMigrateLegacyTab(storage, tabPartial);
      // Use a stored tab info if possible.
      // Fallback to getDefaultTab() otherwise.
      const tab = reactive<TabInfo>({
        ...getDefaultTab(),
        ...tabPartial,
        id,
      });
      // Legacy id support
      tab.connection.databaseId = String(tab.connection.databaseId);
      tab.connection.instanceId = String(tab.connection.instanceId);
      watchTab(tab, false);
      tabs.value.set(id, tab);
    });

    // Fetch opening sheets if needed
    const sheetV1Store = useWorkSheetStore();
    tabList.value.forEach((tab) => {
      if (tab.sheetName) {
        sheetV1Store.getOrFetchSheetByName(tab.sheetName);
      }
    });

    // Clean up unused stored tabs
    _cleanup(tabIdList.value);

    watch(
      currentTabId,
      (id) => {
        storage.save(KEYS.currentTabId, id);
      },
      { immediate: true }
    );
    watch(
      tabIdList,
      (idList) => {
        storage.save(KEYS.tabIdList, idList);
      },
      { deep: true, immediate: true }
    );
  };
  init();

  const reset = () => {
    storage.clear();
    init();
  };

  // exposure
  return {
    tabIdList,
    tabList,
    currentTabId,
    currentTab,
    isDisconnected,
    getTabById,
    addTab,
    removeTab,
    updateTab,
    updateCurrentTab,
    setCurrentTabId,
    selectOrAddTempTab,
    selectOrAddSimilarTab,
    reset,
  };
});

export const useCurrentTab = () => {
  const store = useTabStore();
  return toRef(store, "currentTab");
};

const maybeMigrateLegacyTab = (
  storage: WebStorageHelper,
  tab: PersistentTabInfo | undefined
) => {
  if (!tab) return;
  const { connection } = tab;
  if (
    typeof connection.databaseId === "number" ||
    typeof connection.instanceId === "number"
  ) {
    connection.databaseId = String(connection.databaseId);
    connection.instanceId = String(connection.instanceId);
    storage.save(KEYS.tab(tab.id), tab);
  }
  return tab;
};

export const isTabClosable = (tab: TabInfo) => {
  const { tabList } = useTabStore();

  if (tabList.length > 1) {
    // Not the only one tab
    return true;
  }
  if (tabList.length === 1) {
    // It's the only one tab, and it's closable if it's a sheet tab
    return !!tab.sheetName;
  }
  return false;
};
