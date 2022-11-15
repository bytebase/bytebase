import { defineStore } from "pinia";
import { computed, reactive, ref, watch } from "vue";
import { pick } from "lodash-es";
import { watchThrottled } from "@vueuse/core";
import type { TabInfo, AnyTabInfo } from "@/types";
import { UNKNOWN_ID } from "@/types";
import {
  getDefaultTab,
  INITIAL_TAB,
  isTempTab,
  WebStorageHelper,
} from "@/utils";
import { useInstanceStore } from "./instance";
import { useSheetStore } from "./sheet";

const LOCAL_STORAGE_KEY_PREFIX = "bb.sql-editor.tab-list";
const KEYS = {
  tabIdList: "tab-id-list",
  currentTabId: "current-tab-id",
  tab: (id: string) => `tab.${id}`,
};

// Only store the core fields of a tab.
// Don't store anything which might be too large.
const PERSISTENT_TASK_FIELDS = [
  "id",
  "name",
  "connection",
  "isSaved",
  "savedAt",
  "statement",
  "sheetId",
  "mode",
] as const;
type PersistentTaskInfo = Pick<TabInfo, typeof PERSISTENT_TASK_FIELDS[number]>;

export const useTabStore = defineStore("tab", () => {
  const storage = new WebStorageHelper("bb.sql-editor.tab-list", localStorage);
  const instanceStore = useInstanceStore();

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
    const tab = tabList.value.find((tab) => tab.id === currentTabId.value);
    return tab ?? getDefaultTab();
  });
  const isDisconnected = computed((): boolean => {
    const { instanceId, databaseId } = currentTab.value.connection;
    if (instanceId === UNKNOWN_ID) {
      return true;
    }
    const instance = instanceStore.getInstanceById(instanceId);
    if (instance.engine === "MYSQL" || instance.engine === "TIDB") {
      // Connecting to instance directly.
      return false;
    }
    return databaseId === UNKNOWN_ID;
  });

  // actions
  const addTab = (payload?: AnyTabInfo) => {
    const newTab = reactive<TabInfo>({
      ...getDefaultTab(),
      ...payload,
    });

    const { id } = newTab;
    tabIdList.value.push(id);
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
    }
  };
  const updateCurrentTab = (payload: AnyTabInfo) => {
    Object.assign(currentTab.value, payload);
  };
  const setCurrentTabId = (id: string) => {
    currentTabId.value = id;
  };
  const selectOrAddTempTab = () => {
    if (isTempTab(currentTab.value)) {
      return;
    }
    const tempTab = tabList.value.find(isTempTab);
    if (tempTab) {
      setCurrentTabId(tempTab.id);
    } else {
      addTab();
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
      () => pick(tab, ...PERSISTENT_TASK_FIELDS),
      (tabPartial: PersistentTaskInfo) => {
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
      const tabPartial = storage.load<PersistentTaskInfo | undefined>(
        KEYS.tab(id),
        undefined
      );
      // Use a stored tab info if possible.
      // Fallback to getDefaultTab() otherwise.
      const tab = reactive<TabInfo>({
        ...getDefaultTab(),
        ...tabPartial,
        id,
      });
      watchTab(tab, false);
      tabs.value.set(id, tab);
    });

    // Fetch opening sheets if needed
    const sheetStore = useSheetStore();
    tabList.value.forEach((tab) => {
      if (tab.sheetId && tab.sheetId !== UNKNOWN_ID) {
        sheetStore.getOrFetchSheetById(tab.sheetId);
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
    reset,
  };
});
