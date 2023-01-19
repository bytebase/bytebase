import { reactive } from "vue";
import { defineStore } from "pinia";
import type { TabInfo, WebTerminalQueryItem } from "@/types";

const createInitialQueryItemByTab = (tab: TabInfo): WebTerminalQueryItem => ({
  sql: tab.statement,
  status: "IDLE",
});

export const useWebTerminalStore = defineStore("webTerminal", () => {
  const map = new Map<string, WebTerminalQueryItem[]>();

  const getQueryListByTab = (tab: TabInfo) => {
    const existed = map.get(tab.id);
    if (existed) return existed;

    const init = reactive([createInitialQueryItemByTab(tab)]);
    // TODO(Jim): watch the length of the list, shift the oldest one if len>20 (maybe)
    map.set(tab.id, init);
    return init;
  };

  const clearQueryListByTab = (tab: TabInfo) => {
    map.delete(tab.id);
  };

  return { getQueryListByTab, clearQueryListByTab };
});
