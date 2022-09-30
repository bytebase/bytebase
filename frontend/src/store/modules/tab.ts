import { defineStore } from "pinia";
import { computed, ref } from "vue";
import type { TabInfo, AnyTabInfo } from "@/types";
import { UNKNOWN_ID } from "@/types";
import { getDefaultTab, INITIAL_TAB, isTempTab } from "@/utils";
import { useInstanceStore } from "./instance";

export const useTabStore = defineStore("tab", () => {
  const instanceStore = useInstanceStore();

  // states
  const tabList = ref<TabInfo[]>([INITIAL_TAB]);
  const currentTabId = ref(INITIAL_TAB.id);

  // getters
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
    const defaultTab = getDefaultTab();

    const newTab = {
      ...defaultTab,
      ...payload,
    };

    currentTabId.value = newTab.id;
    tabList.value.push(newTab);
  };
  const removeTab = (payload: TabInfo) => {
    const index = tabList.value.indexOf(payload);
    if (index >= 0) {
      tabList.value.splice(index, 1);
    }
  };
  const updateCurrentTab = (payload: AnyTabInfo) => {
    Object.assign(currentTab.value, payload);
  };
  const setCurrentTabId = (id: string) => {
    currentTabId.value = id;
  };
  const selectOrAddTempTab = () => {
    const tempTab = tabList.value.find(isTempTab);
    if (tempTab) {
      setCurrentTabId(tempTab.id);
    } else {
      addTab();
    }
  };

  // exposure
  return {
    tabList,
    currentTabId,
    currentTab,
    isDisconnected,
    addTab,
    removeTab,
    updateCurrentTab,
    setCurrentTabId,
    selectOrAddTempTab,
  };
});
