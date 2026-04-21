import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, test } from "vitest";
import { STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE } from "@/utils/storage-keys";

let useSQLEditorUIStore: typeof import("./uiState").useSQLEditorUIStore;

beforeEach(async () => {
  localStorage.clear();
  setActivePinia(createPinia());
  ({ useSQLEditorUIStore } = await import("./uiState"));
});

describe("useSQLEditorUIStore", () => {
  test("initial state has documented defaults", () => {
    const store = useSQLEditorUIStore();
    expect(store.asidePanelTab).toBe("WORKSHEET");
    expect(store.showConnectionPanel).toBe(false);
    expect(store.showAIPanel).toBe(false);
    expect(store.schemaViewer).toBeUndefined();
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
