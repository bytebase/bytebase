import { useEffect, useRef } from "react";
import { useSQLEditorStore } from "@/react/stores/sqlEditor";
import {
  getSQLEditorTabsState,
  useSQLEditorTabState,
} from "@/react/stores/sqlEditor/tab";
import type { WebTerminalQueryItemV1 } from "@/types";
import { minmax } from "@/utils";

const MAX_HISTORY_ITEM_COUNT = 1000;

interface HistoryState {
  // Position in `list`. `list.length` means "live tail" (no history loaded).
  index: number;
  // Snapshots of FINALIZED query items, captured at the moment a new tail
  // gets appended (so the captured statement is the one that was actually
  // executed, not the empty initial value).
  list: WebTerminalQueryItemV1[];
}

/**
 * React port of `frontend/src/views/sql-editor/EditorPanel/TerminalPanel/useHistory.ts`.
 *
 * Tracks an in-memory command history per ADMIN-mode tab so the up/down
 * arrow keys cycle through previously executed statements. Differs from the
 * Vue original in HOW it captures snapshots: Vue mutated reactive proxies in
 * place, so it could safely push the live tail at creation time and read its
 * statement later. zustand replaces items immutably on every patch, so we
 * push only when an item is *finalized* (a new tail just got appended,
 * meaning the prior tail's statement is locked in).
 */
export function useHistory() {
  const updateWebTerminalQueryItem = useSQLEditorStore(
    (s) => s.updateWebTerminalQueryItem
  );
  const historyByTabIdRef = useRef(new Map<string, HistoryState>());

  const currentTabId = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.id
  );
  // Subscribe to the tail item's id. The id changes only when a new tail is
  // appended (the slice patches statement updates in place by id, preserving
  // it). Tracking the id — not the list length — lets detection survive
  // after the list hits its 20-item cap and length stops growing on each
  // append.
  const currentTailId = useSQLEditorStore((s) => {
    if (!currentTabId) return undefined;
    const list = s.webTerminalQueryItemsByTabId[currentTabId];
    return list && list.length > 0 ? list[list.length - 1].id : undefined;
  });

  const currentStack = (): HistoryState | undefined => {
    const tabsState = getSQLEditorTabsState();
    const tab = tabsState.tabsById.get(tabsState.currentTabId);
    if (!tab) return undefined;
    if (tab.mode !== "ADMIN") return undefined;
    const map = historyByTabIdRef.current;
    const existed = map.get(tab.id);
    if (existed) return existed;
    const initial: HistoryState = { index: 0, list: [] };
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
    // After push, position the cursor "past the tail" so Up goes one back.
    stack.index = stack.list.length;
  };

  // Detect a new live tail being appended via tail-id change. When the id
  // moves to a fresh one, the previous tail (whose id we just rotated off)
  // is now finalized — its statement is locked in. The finalized item is at
  // `length - 2` in the current list regardless of whether the list grew or
  // was capped (cap evicts oldest from the head, so position from the tail
  // is stable).
  const lastTabIdRef = useRef<string | undefined>(undefined);
  const lastSeenTailIdRef = useRef<string | undefined>(undefined);
  useEffect(() => {
    if (lastTabIdRef.current !== currentTabId) {
      lastTabIdRef.current = currentTabId;
      lastSeenTailIdRef.current = currentTailId;
      return;
    }
    if (currentTailId && currentTailId !== lastSeenTailIdRef.current) {
      const tabsState = getSQLEditorTabsState();
      const tab = tabsState.tabsById.get(tabsState.currentTabId);
      if (tab && tab.mode === "ADMIN") {
        const items =
          useSQLEditorStore.getState().webTerminalQueryItemsByTabId[tab.id];
        if (items && items.length >= 2) {
          const finalized = items[items.length - 2];
          if (finalized?.statement) push(finalized);
        }
      }
    }
    lastSeenTailIdRef.current = currentTailId;
  }, [currentTailId, currentTabId]);

  const move = (direction: "up" | "down") => {
    const stack = currentStack();
    if (!stack) return;
    const { index, list } = stack;
    if (list.length === 0) return;
    const delta = direction === "up" ? -1 : 1;
    const nextIndex = minmax(index + delta, 0, list.length);
    if (nextIndex === index) return;

    const tabsState = getSQLEditorTabsState();
    const tab = tabsState.tabsById.get(tabsState.currentTabId);
    if (!tab) return;
    const tabItems =
      useSQLEditorStore.getState().webTerminalQueryItemsByTabId[tab.id];
    const tail = tabItems?.[tabItems.length - 1];
    if (!tail) return;

    if (nextIndex === list.length) {
      // Stepped past the newest entry — back to the live (empty) tail.
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
