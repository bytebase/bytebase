import { useEffect, useRef } from "react";
import { useVueState } from "@/react/hooks/useVueState";
import { useSQLEditorStore } from "@/react/stores/sqlEditor";
import { useSQLEditorTabStore } from "@/store";
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
  const updateWebTerminalQueryItem = useSQLEditorStore(
    (s) => s.updateWebTerminalQueryItem
  );
  const historyByTabIdRef = useRef(new Map<string, HistoryState>());

  // The selector returns the (immutable) tail item from zustand; React
  // re-renders whenever the underlying array changes.
  const currentTabId = useVueState(() => tabStore.currentTab?.id);
  const currentQuery = useSQLEditorStore((s) => {
    if (!currentTabId) return undefined;
    const list = s.webTerminalQueryItemsByTabId[currentTabId];
    return list && list.length > 0 ? list[list.length - 1] : undefined;
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
  // `historyByTabIdRef` is stable; `tabStore` is stable; `push` reads
  // through them, so we only need to refire on `currentQuery` identity.
  const currentQueryRef = useRef(currentQuery);
  currentQueryRef.current = currentQuery;
  useEffect(() => {
    const q = currentQueryRef.current;
    if (q) push(q);
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
    const tabItems =
      useSQLEditorStore.getState().webTerminalQueryItemsByTabId[tab.id];
    const tail = tabItems?.[tabItems.length - 1];
    if (!tail) return;

    if (nextIndex === list.length - 1) {
      updateWebTerminalQueryItem(tab.id, tail.id, { statement: "" });
    } else {
      const historyQuery = list[nextIndex];
      if (historyQuery)
        updateWebTerminalQueryItem(tab.id, tail.id, {
          statement: historyQuery.statement,
        });
    }
    stack.index = nextIndex;
  };

  return { push, move };
}
