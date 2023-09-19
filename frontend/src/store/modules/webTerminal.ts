import { uniqueId } from "lodash-es";
import { defineStore } from "pinia";
import { reactive, ref } from "vue";
import type { TabInfo, WebTerminalQueryItem } from "@/types";

const createInitialQueryItemByTab = (tab: TabInfo): WebTerminalQueryItem => {
  return createQueryItem(tab.statement);
};

export const createQueryItem = (
  sql = "",
  status: WebTerminalQueryItem["status"] = "IDLE"
): WebTerminalQueryItem => ({
  id: uniqueId(),
  sql,
  status,
});

export const useWebTerminalStore = defineStore("webTerminal", () => {
  const map = ref(new Map<string, WebTerminalQueryItem[]>());

  const getQueryListByTab = (tab: TabInfo) => {
    const existed = map.value.get(tab.id);
    if (existed) return existed;

    const init = reactive([createInitialQueryItemByTab(tab)]);
    map.value.set(tab.id, init);
    return init;
  };

  const clearQueryListByTab = (tab: TabInfo) => {
    map.value.delete(tab.id);
  };

  return { getQueryListByTab, clearQueryListByTab };
});
