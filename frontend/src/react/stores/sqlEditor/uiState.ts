import { STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE } from "@/utils/storage-keys";
import type {
  SQLEditorSliceCreator,
  SQLEditorStoreState,
  UIStateSlice,
} from "./types";

const MINIMUM_EDITOR_PANEL_SIZE = 0.5;
const DEFAULT_AI_PANEL_SIZE = 0.3;

const readAIPanelSize = (): number => {
  try {
    const raw = localStorage.getItem(STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE);
    if (raw !== null) {
      const parsed = JSON.parse(raw);
      if (typeof parsed === "number") return parsed;
    }
  } catch {
    // ignore — fall back to default
  }
  return DEFAULT_AI_PANEL_SIZE;
};

const writeAIPanelSize = (size: number) => {
  try {
    localStorage.setItem(
      STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE,
      JSON.stringify(size)
    );
  } catch {
    // ignore
  }
};

export const createUIStateSlice: SQLEditorSliceCreator<UIStateSlice> = (
  set
) => ({
  asidePanelTab: "WORKSHEET",
  showConnectionPanel: false,
  showAIPanel: false,
  pendingInsertAtCaret: undefined,
  highlightAccessGrantName: undefined,
  isShowingCode: false,
  aiPanelSize: readAIPanelSize(),

  setAsidePanelTab: (tab) => set({ asidePanelTab: tab }),
  setShowConnectionPanel: (v) => set({ showConnectionPanel: v }),
  setShowAIPanel: (v) => set({ showAIPanel: v }),
  setPendingInsertAtCaret: (v) => set({ pendingInsertAtCaret: v }),
  setHighlightAccessGrantName: (v) => set({ highlightAccessGrantName: v }),
  setIsShowingCode: (v) => set({ isShowingCode: v }),
  handleEditorPanelResize: (size) => {
    if (size >= 1) return;
    const next = 1 - size;
    writeAIPanelSize(next);
    set({ aiPanelSize: next });
  },
});

export const selectEditorPanelSize = (
  state: SQLEditorStoreState
): { size: number; max: number; min: number } => {
  if (!state.showAIPanel) {
    return { size: 1, max: 1, min: 1 };
  }
  return {
    size: Math.max(1 - state.aiPanelSize, MINIMUM_EDITOR_PANEL_SIZE),
    max: 0.9,
    min: MINIMUM_EDITOR_PANEL_SIZE,
  };
};
