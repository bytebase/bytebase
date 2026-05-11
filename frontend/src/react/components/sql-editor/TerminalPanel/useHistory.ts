import { useEffect, useRef } from "react";
import { useVueState } from "@/react/hooks/useVueState";
import { useSQLEditorTabStore, useWebTerminalStore } from "@/store";
import type { WebTerminalQueryItemV1 } from "@/types";
import { minmax } from "@/utils";

const MAX_HISTORY_ITEM_COUNT = 1000;

interface HistoryState {
  index: number;
  list: WebTerminalQueryItemV1[];
}

/**
 * React port of `frontend/src/views/sql-editor/EditorPanel/TerminalPanel/useHistory.ts`.
 * Tracks an in-memory command history per ADMIN-mode tab so the up/down
 * arrow keys cycle through previously executed statements (replacing the
 * editor's current statement). Mirrors the Vue composable exactly:
 * push on every new query item, move respecting bounds, clear current
 * statement when stepping past the tail.
 */
export function useHistory() {
  const tabStore = useSQLEditorTabStore();
  const webTerminalStore = useWebTerminalStore();
  const historyByTabIdRef = useRef(new Map<string, HistoryState>());

  const currentQuery = useVueState((): WebTerminalQueryItemV1 | undefined => {
    const tab = tabStore.currentTab;
    if (!tab) return undefined;
    const list = webTerminalStore.getQueryStateByTab(tab).queryItemList.value;
    return list[list.length - 1];
  });

  const currentStack = (): HistoryState | undefined => {
    const tab = tabStore.currentTab;
    if (!tab) return undefined;
    if (tab.mode !== "ADMIN") return undefined;
    const map = historyByTabIdRef.current;
    const existed = map.get(tab.id);
    if (existed) return existed;
    const initial: HistoryState = { index: -1, list: [] };
    map.set(tab.id, initial);
    return initial;
  };

  const push = (query: WebTerminalQueryItemV1) => {
    const stack = currentStack();
    if (!stack) return;
    stack.list.push(query);
    if (stack.list.length > MAX_HISTORY_ITEM_COUNT) {
      stack.list.shift();
    }
    stack.index = stack.list.length - 1;
  };

  // Push the new query item onto the per-tab history whenever the tail
  // changes. Mirrors Vue's `watch(currentQuery, push, { immediate: true })`.
  // `push` reads through refs and store getters, so it doesn't need to be
  // a dependency.
  useEffect(() => {
    if (currentQuery) push(currentQuery);
  }, [currentQuery]);

  const move = (direction: "up" | "down") => {
    const stack = currentStack();
    if (!stack) return;
    const { index, list } = stack;
    const delta = direction === "up" ? -1 : 1;
    const nextIndex = minmax(index + delta, 0, list.length - 1);
    if (nextIndex === index) return;

    const tab = tabStore.currentTab;
    if (!tab) return;
    const head = webTerminalStore.getQueryStateByTab(tab).queryItemList.value;
    const tail = head[head.length - 1];
    if (!tail) return;

    if (nextIndex === list.length - 1) {
      tail.statement = "";
    } else {
      const historyQuery = list[nextIndex];
      if (historyQuery) tail.statement = historyQuery.statement;
    }
    stack.index = nextIndex;
  };

  return { push, move };
}
