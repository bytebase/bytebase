import { defineStore } from "pinia";
import { TabInfo, AnyTabInfo, TabState, UNKNOWN_ID } from "@/types";
import { getDefaultTab, INITIAL_TAB } from "@/utils";
import { useInstanceStore } from "./instance";

export const useTabStore = defineStore("tab", {
  state: (): TabState => ({
    tabList: [INITIAL_TAB],
    currentTabId: INITIAL_TAB.id,
  }),

  getters: {
    currentTab(state): TabInfo {
      const tab = state.tabList.find((tab) => tab.id === state.currentTabId);
      return tab ?? getDefaultTab();
    },
    hasTabs(state: TabState): boolean {
      return state.tabList.length > 0;
    },
    isDisconnected(): boolean {
      const { instanceId, databaseId } = this.currentTab.connection;
      if (instanceId === UNKNOWN_ID) {
        return true;
      }
      const instance = useInstanceStore().getInstanceById(instanceId);
      if (instance.engine === "MYSQL" || instance.engine === "TIDB") {
        // Connecting to instance directly.
        return false;
      }
      return databaseId === UNKNOWN_ID;
    },
  },

  actions: {
    setTabState(payload: Partial<TabState>) {
      Object.assign(this, payload);
    },
    addTab(payload?: AnyTabInfo) {
      const defaultTab = getDefaultTab();

      const newTab = {
        ...defaultTab,
        ...payload,
      };

      this.setTabState({
        currentTabId: newTab.id,
      });
      this.tabList.push(newTab);
    },
    removeTab(payload: TabInfo) {
      this.tabList.splice(this.tabList.indexOf(payload), 1);
    },
    updateCurrentTab(payload: AnyTabInfo) {
      Object.assign(this.currentTab, payload);
    },
    setCurrentTabId(payload: string) {
      this.currentTabId = payload;
    },
  },
});
