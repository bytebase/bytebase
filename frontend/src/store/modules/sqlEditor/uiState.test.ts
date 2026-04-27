import { createPinia, setActivePinia } from "pinia";
import { afterEach, beforeEach, describe, expect, test } from "vitest";
import { STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE } from "@/utils/storage-keys";

let useSQLEditorUIStore: typeof import("./uiState").useSQLEditorUIStore;
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

beforeEach(async () => {
  storage.clear();
  Object.defineProperty(globalThis, "localStorage", {
    value: localStorageMock,
    configurable: true,
  });
  setActivePinia(createPinia());
  ({ useSQLEditorUIStore } = await import("./uiState"));
});

afterEach(() => {
  Object.defineProperty(globalThis, "localStorage", {
    value: originalLocalStorage,
    configurable: true,
  });
});

describe("useSQLEditorUIStore", () => {
  test("initial state has documented defaults", () => {
    const store = useSQLEditorUIStore();
    expect(store.asidePanelTab).toBe("WORKSHEET");
    expect(store.showConnectionPanel).toBe(false);
    expect(store.showAIPanel).toBe(false);
    expect(store.pendingInsertAtCaret).toBeUndefined();
    expect(store.highlightAccessGrantName).toBeUndefined();
  });

  test("editorPanelSize returns full-width when AI panel is hidden", () => {
    const store = useSQLEditorUIStore();
    expect(store.showAIPanel).toBe(false);
    expect(store.editorPanelSize).toEqual({ size: 1, max: 1, min: 1 });
  });

  test("editorPanelSize returns clamped split when AI panel is shown", () => {
    const store = useSQLEditorUIStore();
    store.showAIPanel = true;
    expect(store.editorPanelSize).toEqual({ size: 0.7, max: 0.9, min: 0.5 });
  });

  test("editorPanelSize clamps to minimum 0.5 when AI panel is huge", () => {
    localStorage.setItem(
      STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE,
      JSON.stringify(0.8)
    );
    const store = useSQLEditorUIStore();
    store.showAIPanel = true;
    expect(store.editorPanelSize.size).toBe(0.5);
  });

  test("handleEditorPanelResize writes complement to aiPanelSize LocalStorage", () => {
    const store = useSQLEditorUIStore();
    store.showAIPanel = true;
    store.handleEditorPanelResize(0.6);
    expect(store.editorPanelSize.size).toBe(0.6);
  });

  test("handleEditorPanelResize no-ops when size is >= 1", () => {
    const store = useSQLEditorUIStore();
    store.showAIPanel = true;
    store.handleEditorPanelResize(1);
    expect(store.editorPanelSize.size).toBe(0.7);
  });
});
