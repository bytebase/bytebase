import { storeToRefs } from "pinia";
import type { InjectionKey } from "vue";
import { t } from "@/plugins/i18n";
import {
  pushNotification,
  useSQLEditorTabStore,
  useSQLEditorWorksheetStore,
  useWorkSheetStore,
} from "@/store";
import type {
  FolderContext,
  ViewContext,
  WorksheetFilter,
  WorksheetFolderNode,
  WorksheetLikeItem,
} from "@/store/modules/sqlEditor/worksheet";
import type { SQLEditorTab, SQLEditorTabMode } from "@/types";
import {
  extractWorksheetConnection,
  getSheetStatement,
  isWorksheetReadableV1,
} from "@/utils";
import type { SheetViewMode } from "./types";

// ---- re-export types that consumers import from this module ----------------

export type {
  FolderContext,
  ViewContext,
  WorksheetFilter,
  WorksheetFolderNode,
  WorksheetLikeItem,
};

// ---- public API ------------------------------------------------------------

/**
 * Returns the full sheet context (filter, expandedKeys, selectedKeys, etc.).
 * Previously backed by Vue provide/inject; now delegates to Pinia store.
 * Returns storeToRefs() merged with non-ref methods so destructuring behaves
 * the same as the original inject pattern (refs preserved as Ref<T>).
 */
export const useSheetContext = () => {
  const store = useSQLEditorWorksheetStore();
  const refs = storeToRefs(store);
  return {
    ...refs,
    isWorksheetCreator: store.isWorksheetCreator,
    batchUpdateWorksheetFolders: store.batchUpdateWorksheetFolders,
    getContextByView: store.getContextByView,
  };
};

export type SheetContext = ReturnType<typeof useSheetContext>;

// ---- InjectionKey kept for source-compat (no longer used) ------------------

export const KEY = Symbol("bb.sql-editor.sheet") as InjectionKey<SheetContext>;

/**
 * Returns the per-view context for `view`.
 * Previously backed by Vue inject; now delegates to Pinia store.
 */
export const useSheetContextByView = (view: SheetViewMode) =>
  useSQLEditorWorksheetStore().getContextByView(view);

// ---- tree helpers ----------------------------------------------------------

export const revealNodes = <T>(
  node: WorksheetFolderNode,
  callback: (node: WorksheetFolderNode) => T | undefined
): T[] => {
  const results: T[] = [];
  const item = callback(node);
  if (item) {
    results.push(item);
  }
  for (const child of node.children) {
    results.push(...revealNodes(child, callback));
  }
  return results;
};

export const revealWorksheets = <T>(
  node: WorksheetFolderNode,
  callback: (node: WorksheetFolderNode) => T | undefined
): T[] => {
  return revealNodes(node, (n) => {
    if (!n.worksheet) {
      return undefined;
    }
    return callback(n);
  });
};

// ---- provideSheetContext: kept for backward compat (no-op) -----------------

// Called from SQLEditorLayout.vue to eagerly boot the store so watchers
// (tab-selection → selectedKeys) start immediately on app mount.
export const provideSheetContext = () => {
  return useSQLEditorWorksheetStore();
};

// ---- openWorksheetByName (unchanged) ----------------------------------------

export const openWorksheetByName = async ({
  worksheet,
  forceNewTab,
  mode,
}: {
  worksheet: string;
  forceNewTab: boolean;
  mode?: SQLEditorTabMode;
}) => {
  const sheet = await useWorkSheetStore().getOrFetchWorksheetByName(worksheet);
  if (!sheet) {
    return undefined;
  }

  if (!isWorksheetReadableV1(sheet)) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.access-denied"),
    });
    return undefined;
  }

  const tabStore = useSQLEditorTabStore();
  const openingSheetTab = tabStore.getTabByWorksheet(sheet.name);

  if (openingSheetTab && !forceNewTab) {
    tabStore.setCurrentTabId(openingSheetTab.id);
    if (mode && mode !== openingSheetTab.mode) {
      tabStore.updateTab(openingSheetTab.id, { mode });
    }
    return openingSheetTab;
  }

  const statement = getSheetStatement(sheet);
  const connection = await extractWorksheetConnection(sheet);
  const newTab: Partial<SQLEditorTab> = {
    connection,
    worksheet: sheet.name,
    title: sheet.title,
    statement,
    status: "CLEAN",
    mode: mode ?? "WORKSHEET",
  };

  return tabStore.addTab(newTab, forceNewTab /* beside */);
};
