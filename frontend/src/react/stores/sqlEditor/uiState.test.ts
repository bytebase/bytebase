import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { create, type StoreApi } from "zustand";
import { STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE } from "@/utils/storage-keys";
import type {
  QueryHistorySlice,
  SQLEditorStoreState,
  TreeSlice,
  WebTerminalSlice,
  WorksheetSaveSlice,
} from "./types";

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

// Inline stub for the queryHistory slice so this test stays decoupled
// from the real `./queryHistory` module — that module pulls in
// `@/connect` and (transitively) the Pinia store layout, which can
// create circular-import issues during test load and isn't relevant
// to validating uiState behavior.
const stubQueryHistorySlice = (): QueryHistorySlice => ({
  queryHistoryByKey: {},
  fetchQueryHistoryList: vi.fn().mockResolvedValue(undefined),
  resetPageToken: vi.fn(),
  mergeLatest: vi.fn().mockResolvedValue(undefined),
});

const stubTreeSlice = (): TreeSlice => ({
  treeState: "UNSET",
  treeNodeKeysById: {},
  setTreeState: vi.fn(),
  collectTreeNode: vi.fn(),
  treeNodeKeysByTarget: vi.fn(() => []),
});

const stubWebTerminalSlice = (): WebTerminalSlice => ({
  webTerminalQueryItemsByTabId: {},
  ensureWebTerminalQueryState: vi.fn(),
  clearWebTerminalQueryState: vi.fn(),
  replaceWebTerminalQueryItems: vi.fn(),
  pushWebTerminalQueryItem: vi.fn(),
  updateWebTerminalQueryItem: vi.fn(),
});

const stubWorksheetSaveSlice = (): WorksheetSaveSlice => ({
  autoSaveController: null,
  setAutoSaveController: vi.fn(),
  abortAutoSave: vi.fn(),
  maybeSwitchProject: vi.fn(async () => undefined),
  maybeUpdateWorksheet: vi.fn(async () => undefined),
  createWorksheet: vi.fn(async () => undefined),
});

// Build a fresh store for each test so the slice's `aiPanelSize`
// initialiser re-reads the (mocked) localStorage and we don't leak
// state between tests. The slice module is dynamically re-imported
// with a query string so the module-level localStorage read in
// `createUIStateSlice` runs again under the mock.
const makeStore = async (): Promise<StoreApi<SQLEditorStoreState>> => {
  const mod = (await import(
    `./uiState?t=${Date.now()}`
  )) as typeof import("./uiState");
  return create<SQLEditorStoreState>()((...args) => ({
    ...mod.createUIStateSlice(...args),
    ...stubQueryHistorySlice(),
    ...stubTreeSlice(),
    ...stubWebTerminalSlice(),
    ...stubWorksheetSaveSlice(),
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
