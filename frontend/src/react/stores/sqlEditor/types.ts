import type { StateCreator } from "zustand";
import type {
  SQLEditorTreeNodeTarget as NodeTarget,
  SQLEditorTreeNodeType as NodeType,
  SQLEditorTab,
  SQLEditorTreeNode as TreeNode,
  SQLEditorTreeState as TreeState,
  WebTerminalQueryItemV1,
} from "@/types";
import type {
  QueryHistory,
  SearchQueryHistoriesResponse,
} from "@/types/proto-es/v1/sql_service_pb";

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

export interface QueryHistoryFilter {
  statement?: string;
  project?: string;
  database?: string;
}

export interface QueryHistoryEntry {
  nextPageToken?: string;
  queryHistories: QueryHistory[];
}

export interface QueryHistorySlice {
  queryHistoryByKey: Record<string, QueryHistoryEntry>;
  fetchQueryHistoryList: (
    filter: QueryHistoryFilter
  ) => Promise<SearchQueryHistoriesResponse>;
  resetPageToken: (filter: QueryHistoryFilter) => void;
  mergeLatest: (
    filter: QueryHistoryFilter
  ) => Promise<SearchQueryHistoriesResponse>;
}

export interface TreeSlice {
  treeState: TreeState;
  // Index of node-target id → registered node keys.
  treeNodeKeysById: Record<string, string[]>;

  setTreeState: (state: TreeState) => void;
  collectTreeNode: <T extends NodeType>(node: TreeNode<T>) => void;
  treeNodeKeysByTarget: <T extends NodeType>(
    type: T,
    target: NodeTarget<T>
  ) => string[];
}

export interface WebTerminalSlice {
  // Per-tab admin-mode query items. The WebSocket / RxJS / timer
  // services live module-side (see `webTerminal-service.ts`); only the
  // React-visible state is in the slice.
  webTerminalQueryItemsByTabId: Record<string, WebTerminalQueryItemV1[]>;
  ensureWebTerminalQueryState: (tabId: string) => void;
  clearWebTerminalQueryState: (tabId: string) => void;
  replaceWebTerminalQueryItems: (
    tabId: string,
    items: WebTerminalQueryItemV1[]
  ) => void;
  pushWebTerminalQueryItem: (tabId: string) => void;
  updateWebTerminalQueryItem: (
    tabId: string,
    itemId: string,
    patch: Partial<WebTerminalQueryItemV1>
  ) => void;
}

export interface WorksheetSaveSlice {
  // Live AbortController for the active auto-save request. Set by the
  // auto-save composable so a subsequent edit can cancel the in-flight
  // PATCH. `null` between saves.
  autoSaveController: AbortController | null;
  setAutoSaveController: (controller: AbortController | null) => void;
  abortAutoSave: () => void;
  maybeSwitchProject: (projectName: string) => Promise<string | undefined>;
  maybeUpdateWorksheet: (opts: {
    tabId: string;
    worksheet?: string;
    title?: string;
    database: string;
    statement: string;
    folders?: string[];
    signal?: AbortSignal;
  }) => Promise<SQLEditorTab | undefined>;
  createWorksheet: (opts: {
    tabId?: string;
    title?: string;
    statement?: string;
    folders?: string[];
    database?: string;
  }) => Promise<SQLEditorTab | undefined>;
}

export type SQLEditorStoreState = UIStateSlice &
  QueryHistorySlice &
  TreeSlice &
  WebTerminalSlice &
  WorksheetSaveSlice;

export type SQLEditorSliceCreator<Slice> = StateCreator<
  SQLEditorStoreState,
  [],
  [],
  Slice
>;
