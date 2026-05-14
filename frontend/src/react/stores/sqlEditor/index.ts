import { create } from "zustand";
import type { SQLEditorStoreState } from "./types";
import { createUIStateSlice } from "./uiState";

export type { AsidePanelTab, SQLEditorStoreState } from "./types";
export { selectEditorPanelSize } from "./uiState";

export const useSQLEditorStore = create<SQLEditorStoreState>()((...args) => ({
  ...createUIStateSlice(...args),
}));
