import { afterEach, beforeEach, describe, expect, test } from "vitest";
import { create, type StoreApi } from "zustand";
import { STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE } from "@/utils/storage-keys";
import type { SQLEditorStoreState } from "./types";

const originalLocalStorage = globalThis.localStorage;
const storage = new Map<string, string>();
const localStorageMock = {
  getItem: (key: string) => storage.get(key) ?? null,
  setItem: (key: string, value: string) => {
    storage.set(key, value);
  },
  removeItem: (key: string) => {
    storage.delete(key);
  },
  clear: () => {
    storage.clear();
  },
  key: (index: number) => Array.from(storage.keys())[index] ?? null,
  get length() {
    return storage.size;
  },
};

// Build a fresh store for each test so the slice's `aiPanelSize`
// initialiser re-reads the (mocked) localStorage and we don't leak state
// between tests. The slice module is dynamically imported with a query
// string so Vitest doesn't cache the module-level read of localStorage.
const makeStore = async (): Promise<StoreApi<SQLEditorStoreState>> => {
  const mod = (await import(
    `./uiState?t=${Date.now()}`
  )) as typeof import("./uiState");
  return create<SQLEditorStoreState>()((...args) => ({
    ...mod.createUIStateSlice(...args),
  }));
};

let selectEditorPanelSize: typeof import("./uiState").selectEditorPanelSize;

beforeEach(async () => {
  storage.clear();
  Object.defineProperty(globalThis, "localStorage", {
    value: localStorageMock,
    configurable: true,
  });
  ({ selectEditorPanelSize } = await import(`./uiState?t=${Date.now()}`));
});

afterEach(() => {
  Object.defineProperty(globalThis, "localStorage", {
    value: originalLocalStorage,
    configurable: true,
  });
});

describe("sqlEditor uiState slice", () => {
  test("initial state has documented defaults", async () => {
    const useStore = await makeStore();
    const s = useStore.getState();
    expect(s.asidePanelTab).toBe("WORKSHEET");
    expect(s.showConnectionPanel).toBe(false);
    expect(s.showAIPanel).toBe(false);
    expect(s.pendingInsertAtCaret).toBeUndefined();
    expect(s.highlightAccessGrantName).toBeUndefined();
    expect(s.isShowingCode).toBe(false);
    expect(s.aiPanelSize).toBe(0.3);
  });

  test("selectEditorPanelSize returns full-width when AI panel is hidden", async () => {
    const useStore = await makeStore();
    expect(selectEditorPanelSize(useStore.getState())).toEqual({
      size: 1,
      max: 1,
      min: 1,
    });
  });

  test("selectEditorPanelSize returns clamped split when AI panel is shown", async () => {
    const useStore = await makeStore();
    useStore.getState().setShowAIPanel(true);
    expect(selectEditorPanelSize(useStore.getState())).toEqual({
      size: 0.7,
      max: 0.9,
      min: 0.5,
    });
  });

  test("selectEditorPanelSize clamps to minimum 0.5 when AI panel is huge", async () => {
    localStorage.setItem(
      STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE,
      JSON.stringify(0.8)
    );
    const useStore = await makeStore();
    useStore.getState().setShowAIPanel(true);
    expect(selectEditorPanelSize(useStore.getState()).size).toBe(0.5);
  });

  test("handleEditorPanelResize writes complement to aiPanelSize and persists it", async () => {
    const useStore = await makeStore();
    useStore.getState().setShowAIPanel(true);
    useStore.getState().handleEditorPanelResize(0.6);
    expect(selectEditorPanelSize(useStore.getState()).size).toBe(0.6);
    expect(localStorage.getItem(STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE)).toBe(
      JSON.stringify(0.4)
    );
  });

  test("handleEditorPanelResize no-ops when size is >= 1", async () => {
    const useStore = await makeStore();
    useStore.getState().setShowAIPanel(true);
    useStore.getState().handleEditorPanelResize(1);
    expect(selectEditorPanelSize(useStore.getState()).size).toBe(0.7);
  });
});
