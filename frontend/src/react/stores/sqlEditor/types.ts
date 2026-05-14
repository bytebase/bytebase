import type { StateCreator } from "zustand";

export type AsidePanelTab = "SCHEMA" | "WORKSHEET" | "HISTORY" | "ACCESS";

export interface UIStateSlice {
  asidePanelTab: AsidePanelTab;
  showConnectionPanel: boolean;
  showAIPanel: boolean;
  pendingInsertAtCaret: string | undefined;
  highlightAccessGrantName: string | undefined;
  // True while a CodeViewer-style surface is mounted (procedure / function /
  // package body, view definition, trigger body). Used by panels to decide
  // whether to render the AIChatToSQL side pane.
  isShowingCode: boolean;
  // Persisted fraction of the editor width given to the AI side pane when it
  // is shown. Stored in localStorage and re-read on store creation.
  aiPanelSize: number;

  setAsidePanelTab: (tab: AsidePanelTab) => void;
  setShowConnectionPanel: (value: boolean) => void;
  setShowAIPanel: (value: boolean) => void;
  setPendingInsertAtCaret: (value: string | undefined) => void;
  setHighlightAccessGrantName: (value: string | undefined) => void;
  setIsShowingCode: (value: boolean) => void;
  handleEditorPanelResize: (size: number) => void;
}

export type SQLEditorStoreState = UIStateSlice;

export type SQLEditorSliceCreator<Slice> = StateCreator<
  SQLEditorStoreState,
  [],
  [],
  Slice
>;
