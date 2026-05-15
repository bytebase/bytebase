import type { WebTerminalQueryItemV1 } from "@/types";
import type { SQLEditorSliceCreator, WebTerminalSlice } from "./types";

const MAX_QUERY_ITEM_COUNT = 20;

let nextItemId = 0;
const generateItemId = () => `wt-${++nextItemId}`;

export const createWebTerminalQueryItemV1 = (
  statement = "",
  status: WebTerminalQueryItemV1["status"] = "IDLE"
): WebTerminalQueryItemV1 => ({
  id: generateItemId(),
  statement,
  status,
});

const EMPTY_ITEMS: WebTerminalQueryItemV1[] = [];

export const createWebTerminalSlice: SQLEditorSliceCreator<WebTerminalSlice> = (
  set,
  get
) => ({
  webTerminalQueryItemsByTabId: {},

  ensureWebTerminalQueryState: (tabId) => {
    if (get().webTerminalQueryItemsByTabId[tabId]) return;
    set((s) => ({
      webTerminalQueryItemsByTabId: {
        ...s.webTerminalQueryItemsByTabId,
        [tabId]: [createWebTerminalQueryItemV1()],
      },
    }));
  },

  clearWebTerminalQueryState: (tabId) => {
    set((s) => {
      if (!(tabId in s.webTerminalQueryItemsByTabId)) return s;
      const next = { ...s.webTerminalQueryItemsByTabId };
      delete next[tabId];
      return { webTerminalQueryItemsByTabId: next };
    });
  },

  replaceWebTerminalQueryItems: (tabId, items) => {
    set((s) => ({
      webTerminalQueryItemsByTabId: {
        ...s.webTerminalQueryItemsByTabId,
        [tabId]: items,
      },
    }));
  },

  pushWebTerminalQueryItem: (tabId) => {
    set((s) => {
      const prev = s.webTerminalQueryItemsByTabId[tabId] ?? EMPTY_ITEMS;
      const next = [...prev, createWebTerminalQueryItemV1()];
      // Trim oldest items above the per-tab cap so memory stays bounded.
      while (next.length > MAX_QUERY_ITEM_COUNT) next.shift();
      return {
        webTerminalQueryItemsByTabId: {
          ...s.webTerminalQueryItemsByTabId,
          [tabId]: next,
        },
      };
    });
  },

  updateWebTerminalQueryItem: (tabId, itemId, patch) => {
    set((s) => {
      const prev = s.webTerminalQueryItemsByTabId[tabId];
      if (!prev) return s;
      const next = prev.map((item) =>
        item.id === itemId ? { ...item, ...patch } : item
      );
      return {
        webTerminalQueryItemsByTabId: {
          ...s.webTerminalQueryItemsByTabId,
          [tabId]: next,
        },
      };
    });
  },
});
