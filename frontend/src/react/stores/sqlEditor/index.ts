import { create } from "zustand";
import { createQueryHistorySlice } from "./queryHistory";
import { createTreeSlice } from "./tree";
import type { SQLEditorStoreState } from "./types";
import { createUIStateSlice } from "./uiState";
import { createWebTerminalSlice } from "./webTerminal";
import { createWorksheetSaveSlice } from "./worksheet";

export type {
  AsidePanelTab,
  QueryHistoryEntry,
  QueryHistoryFilter,
  SQLEditorStoreState,
} from "./types";
export {
  getQueryHistoryCacheKey,
  selectQueryHistoryEntry,
} from "./queryHistory";
export { idForSQLEditorTreeNodeTarget, selectAllTreeNodeKeys } from "./tree";
export { selectEditorPanelSize } from "./uiState";
export { createWebTerminalQueryItemV1 } from "./webTerminal";

export const useSQLEditorStore = create<SQLEditorStoreState>()((...args) => ({
  ...createUIStateSlice(...args),
  ...createQueryHistorySlice(...args),
  ...createTreeSlice(...args),
  ...createWebTerminalSlice(...args),
  ...createWorksheetSaveSlice(...args),
}));
