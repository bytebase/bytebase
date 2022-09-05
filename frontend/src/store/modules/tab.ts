import { defineStore } from "pinia";
import { cloneDeep } from "lodash-es";
import { TabInfo, TabState } from "@/types";
import { getDefaultTab } from "@/utils/tab";
import { useSQLEditorStore } from "./sqlEditor";

export const useTabStore = defineStore("tab", {
  state: (): TabState => ({
    tabList: [],
    currentTabId: "",
  }),

  getters: {
    currentTab(state) {
      const tab = state.tabList.find((tab) => tab.id === state.currentTabId);
      return tab ?? ({} as TabInfo);
    },
    hasTabs(state) {
      return state.tabList.length > 0;
    },
  },

  actions: {
    addTab(payload?: Partial<TabInfo>) {
      const defaultTab = getDefaultTab();

      // Clone current connection context to the newly created tab temporarily.
      const newTab = {
        ...defaultTab,
        connectionContext: cloneDeep(useSQLEditorStore().connectionContext),
        ...payload,
      };

      this.currentTabId = newTab.id;
      this.tabList.push(newTab);
    },
    removeTab(payload: TabInfo) {
      this.tabList.splice(this.tabList.indexOf(payload), 1);
    },
    updateCurrentTab(payload: Partial<TabInfo>) {
      const tab = this.tabList.find((tab) => tab.id === this.currentTabId);
      if (tab) {
        Object.assign(tab, payload);
      }
    },
    setCurrentTabId(tabId: string) {
      this.currentTabId = tabId;
    },
  },
});
