import { useEffect, useRef } from "react";
import { useVueState } from "@/react/hooks/useVueState";
import { useSQLEditorStore } from "@/react/stores/sqlEditor";
import { useSQLEditorTabStore } from "@/react/stores/sqlEditor/tab-vue-state";
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
  const tabStore = useSQLEditorTabStore();
  const updateWebTerminalQueryItem = useSQLEditorStore(
    (s) => s.updateWebTerminalQueryItem
  );
  const historyByTabIdRef = useRef(new Map<string, HistoryState>());

  const currentTabId = useVueState(() => tabStore.currentTab?.id);
  // Subscribe to list length only — statement edits replace the tail
  // identity but don't grow the list, so they don't trigger this effect.
  const queryListLength = useSQLEditorStore((s) => {
    if (!currentTabId) return 0;
    return s.webTerminalQueryItemsByTabId[currentTabId]?.length ?? 0;
  });

  const currentStack = (): HistoryState | undefined => {
    const tab = tabStore.currentTab;
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

  // Detect a new live tail being appended — at that moment, the item now
  // at `length - 2` was the just-finalized query (status FINISHED, statement
  // locked). Capture it as a history snapshot.
  const lastTabIdRef = useRef<string | undefined>(undefined);
  const lastLengthRef = useRef(0);
  useEffect(() => {
    if (lastTabIdRef.current !== currentTabId) {
      lastTabIdRef.current = currentTabId;
      lastLengthRef.current = queryListLength;
      return;
    }
    if (queryListLength > lastLengthRef.current) {
      const tab = tabStore.currentTab;
      if (tab && tab.mode === "ADMIN") {
        const items =
          useSQLEditorStore.getState().webTerminalQueryItemsByTabId[tab.id];
        // The just-appended tail is at length-1; the finalized previous
        // item is at length-2. Skip empty statements (e.g. initial blank).
        const finalized = items?.[queryListLength - 2];
        if (finalized && finalized.statement) push(finalized);
      }
    }
    lastLengthRef.current = queryListLength;
  }, [queryListLength, currentTabId, tabStore]);

  const move = (direction: "up" | "down") => {
    const stack = currentStack();
    if (!stack) return;
    const { index, list } = stack;
    if (list.length === 0) return;
    const delta = direction === "up" ? -1 : 1;
    const nextIndex = minmax(index + delta, 0, list.length);
    if (nextIndex === index) return;

    const tab = tabStore.currentTab;
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
