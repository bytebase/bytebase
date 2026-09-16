import {
  STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE,
  STORAGE_KEY_SQL_EDITOR_RESULT_PANEL_SIZE,
} from "@/utils/storage-keys";
import type { SQLEditorSliceCreator, UIStateSlice } from "./types";

const MINIMUM_EDITOR_PANEL_SIZE = 0.5;
const DEFAULT_AI_PANEL_SIZE = 0.3;

const DEFAULT_RESULT_PANEL_SIZE = 0.4;
/**
 * The drag range of the result pane. `StandardPanel` derives the panel's
 * `minSize` / `maxSize` from these, so the clamp and the layout cannot drift.
 */
export const MINIMUM_RESULT_PANEL_SIZE = 0.2;
export const MAXIMUM_RESULT_PANEL_SIZE = 0.8;

const clampResultPanelSize = (size: number): number =>
  Math.min(
    Math.max(size, MINIMUM_RESULT_PANEL_SIZE),
    MAXIMUM_RESULT_PANEL_SIZE
  );

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

const readResultPanelSize = (): number => {
  try {
    const raw = localStorage.getItem(STORAGE_KEY_SQL_EDITOR_RESULT_PANEL_SIZE);
    if (raw !== null) {
      const parsed = JSON.parse(raw);
      if (typeof parsed === "number" && Number.isFinite(parsed)) {
        return clampResultPanelSize(parsed);
      }
    }
  } catch {
    // ignore — fall back to default
  }
  return DEFAULT_RESULT_PANEL_SIZE;
};

const writeResultPanelSize = (size: number) => {
  try {
    localStorage.setItem(
      STORAGE_KEY_SQL_EDITOR_RESULT_PANEL_SIZE,
      JSON.stringify(size)
    );
  } catch {
    // ignore
  }
};

// A drag reports a new size every frame; only the one it settles on is worth
// a synchronous storage write.
const RESULT_PANEL_SIZE_WRITE_DELAY_MS = 200;
let resultPanelSizeWriteTimer: ReturnType<typeof setTimeout> | undefined;

const scheduleResultPanelSizeWrite = (size: number) => {
  if (resultPanelSizeWriteTimer !== undefined) {
    clearTimeout(resultPanelSizeWriteTimer);
  }
  resultPanelSizeWriteTimer = setTimeout(() => {
    resultPanelSizeWriteTimer = undefined;
    writeResultPanelSize(size);
  }, RESULT_PANEL_SIZE_WRITE_DELAY_MS);
};

export const createUIStateSlice: SQLEditorSliceCreator<UIStateSlice> = (
  set
) => ({
  asidePanelTab: "SAVED_QUERY",
  showConnectionPanel: false,
  showAIPanel: false,
  pendingInsertAtCaret: undefined,
  highlightAccessGrantName: undefined,
  isShowingCode: false,
  aiPanelSize: readAIPanelSize(),
  resultPanelSize: readResultPanelSize(),
  resultPanelMaximized: false,
  linkedQueryHistory: undefined,
  linkedQueryHistoryTabId: undefined,
  linkedQueryHistoryBaseline: undefined,

  setAsidePanelTab: (tab) => set({ asidePanelTab: tab }),
  setLinkedQueryHistory: (value, meta) =>
    set({
      linkedQueryHistory: value,
      linkedQueryHistoryTabId: value ? meta?.tabId : undefined,
      linkedQueryHistoryBaseline: value ? meta?.baseline : undefined,
    }),
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
  setResultPanelMaximized: (value) => set({ resultPanelMaximized: value }),
  // Only a size the reader dragged is worth keeping. A maximized pane reports
  // the full height and a collapsed one reports nothing, so both are ignored
  // rather than written over the remembered height.
  handleResultPanelResize: (size) => {
    if (!Number.isFinite(size) || size <= 0 || size >= 1) return;
    const next = clampResultPanelSize(size);
    scheduleResultPanelSizeWrite(next);
    set({ resultPanelSize: next });
  },
});

export const selectEditorPanelSize = (
  state: UIStateSlice
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
