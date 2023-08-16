import { watchThrottled } from "@vueuse/core";
import { pick } from "lodash-es";
import { defineStore } from "pinia";
import { computed, reactive, ref, toRef, watch } from "vue";
import { TabInfo, CoreTabInfo, AnyTabInfo, TabMode } from "@/types";
import { UNKNOWN_ID } from "@/types";
import {
  getDefaultTab,
  INITIAL_TAB,
  isTempTab,
  isSimilarTab,
  WebStorageHelper,
  instanceV1AllowsCrossDatabaseQuery,
} from "@/utils";
import { useWebTerminalV1Store } from "./v1";
import { useInstanceV1Store } from "./v1/instance";
import { useSheetV1Store } from "./v1/sheet";

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
] as const;
type PersistentTabInfo = Pick<TabInfo, typeof PERSISTENT_TAB_FIELDS[number]>;

export const useTabStore = defineStore("tab", () => {
  const storage = new WebStorageHelper("bb.sql-editor.tab-list", localStorage);
  const instanceStore = useInstanceV1Store();

  // states
  // We store the tabIdList and the tabs separately.
  // This index-entity modeling enables us to update one tab entity at a time,
  // and reduce the performance costing while writing localStorage.
  const tabs = ref(new Map<string, TabInfo>());
  const tabIdList = ref<string[]>([]);
  const currentTabId = ref<string>();
  const asidePanelTab = ref<"databases" | "sheets">("databases");

  // getters
  const getTabById = (id: string) => {
    return tabs.value.get(id) ?? getDefaultTab();
  };
  const tabList = computed((): TabInfo[] => tabIdList.value.map(getTabById));
  const currentTab = computed((): TabInfo => {
    const tab = tabList.value.find((tab) => tab.id === currentTabId.value);
    return tab ?? getDefaultTab();
  });
  const isDisconnected = computed((): boolean => {
    const { instanceId, databaseId } = currentTab.value.connection;
    if (instanceId === String(UNKNOWN_ID)) {
      return true;
    }
    const instance = instanceStore.getInstanceByUID(instanceId);
    if (instanceV1AllowsCrossDatabaseQuery(instance)) {
      // Connecting to instance directly.
      return false;
    }
    return databaseId === String(UNKNOWN_ID);
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

    if (!newTab.sheetName) {
      // Switch the tab to "database" after adding a new sheet
      // because users need to select a database to continue editing
      asidePanelTab.value = "databases";
    }

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
    Object.assign(currentTab.value, payload);
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
    // Load tabIdList and currentTabId
    tabIdList.value = storage.load(KEYS.tabIdList, []);
    currentTabId.value = storage.load(KEYS.currentTabId, INITIAL_TAB.id);
    if (tabIdList.value.length === 0) {
      // tabIdList is empty accidentally
      tabIdList.value = [INITIAL_TAB.id];
    }
    if (tabIdList.value.indexOf(currentTabId.value) < 0) {
      // currentTabId is not in tabIdList accidentally
      // fallback to the first tab
      currentTabId.value = tabIdList.value[0];
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
    const sheetV1Store = useSheetV1Store();
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
    updateCurrentTab,
    setCurrentTabId,
    selectOrAddTempTab,
    selectOrAddSimilarTab,
    reset,
    asidePanelTab,
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
