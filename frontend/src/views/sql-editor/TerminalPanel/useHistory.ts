import { computed, watch } from "vue";

import { TabMode } from "@/types";
import { useTabStore, useWebTerminalStore } from "@/store";
import { minmax } from "@/utils";

const MAX_HISTORY_ITEM_COUNT = 1000;

type HistoryItem = {
  statement: string;
};

type HistoryState = {
  index: number;
  list: HistoryItem[];
};

export const useHistory = () => {
  const tabStore = useTabStore();
  const webTerminalStore = useWebTerminalStore();
  const historyByTabId = new Map<string, HistoryState>();

  const currentQuery = computed(() => {
    const queryList = webTerminalStore.getQueryListByTab(tabStore.currentTab);
    return queryList[queryList.length - 1];
  });

  const currentStack = () => {
    const tab = tabStore.currentTab;
    if (tab.mode === TabMode.ReadOnly) return undefined;

    const existed = historyByTabId.get(tab.id);
    if (existed) {
      return existed;
    }
    const initial: HistoryState = {
      index: -1,
      list: [],
    };
    historyByTabId.set(tab.id, initial);
    return initial;
  };

  const push = (item: HistoryItem) => {
    const stack = currentStack();
    if (!stack) return;
    const { list } = stack;
    list.push(item);
    if (list.length > MAX_HISTORY_ITEM_COUNT) {
      list.shift();
    }
    stack.index = list.length - 1;
  };

  const move = (direction: "up" | "down") => {
    const stack = currentStack();
    if (!stack) return;
    const { index, list } = stack;
    const delta = direction === "up" ? -1 : 1;
    const nextIndex = minmax(index + delta, 0, list.length - 1);
    if (nextIndex === index) {
      return;
    }
    if (nextIndex === list.length - 1) {
      currentQuery.value.sql = "";
    } else {
      const history = list[nextIndex];
      if (history) {
        currentQuery.value.sql = history.statement;
      }
    }

    stack.index = nextIndex;
  };

  watch(
    () => currentQuery.value,
    (query) => {
      push({ statement: query.sql });
    },
    {
      immediate: true,
    }
  );

  return { push, move };
};
