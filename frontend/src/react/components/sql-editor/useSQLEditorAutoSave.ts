import { useEffect, useRef } from "react";
import { useAppStore } from "@/react/stores/app";
import { useSQLEditorStore } from "@/react/stores/sqlEditor";
import {
  getSQLEditorTabsState,
  useSQLEditorTabState,
} from "@/react/stores/sqlEditor/tab";
import { isWorksheetWritableV1 } from "@/utils";

const AUTO_SAVE_DEBOUNCE_MS = 2000;

/**
 * React port of the auto-save block previously embedded in
 * `views/sql-editor/context.ts`'s `provideSQLEditorContext()`.
 *
 * Watches the active tab's `statement` and after a 2s debounce calls
 * `maybeUpdateWorksheet` if the tab is dirty + writable. Mirrors the
 * Vue `watchDebounced` behavior: aborts any in-flight auto-save when a
 * newer one starts, reverts the tab to DIRTY on error (unless aborted),
 * and re-flags DIRTY when the statement keeps changing during the save.
 *
 * Mounted once at the SQL Editor layout level; safe to call from any
 * component but should only be active while the SQL Editor route is.
 */
export function useSQLEditorAutoSave() {
  const abortAutoSave = useSQLEditorStore((s) => s.abortAutoSave);
  const setAutoSaveController = useSQLEditorStore(
    (s) => s.setAutoSaveController
  );
  const maybeUpdateWorksheet = useSQLEditorStore((s) => s.maybeUpdateWorksheet);

  const statement = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.statement
  );
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (debounceTimerRef.current !== null) {
      clearTimeout(debounceTimerRef.current);
    }
    debounceTimerRef.current = setTimeout(() => {
      void runAutoSave();
    }, AUTO_SAVE_DEBOUNCE_MS);
    return () => {
      if (debounceTimerRef.current !== null) {
        clearTimeout(debounceTimerRef.current);
        debounceTimerRef.current = null;
      }
    };
    // We re-arm the debounce on every statement change. The save itself
    // reads tab.statement from Pinia at fire time, so capturing only the
    // change-trigger here is sufficient.
  }, [statement]);

  const runAutoSave = async () => {
    const tabsState = getSQLEditorTabsState();
    const tab = tabsState.tabsById.get(tabsState.currentTabId);
    if (!tab || !tab.worksheet || tab.status === "CLEAN") return;

    const worksheet = useAppStore.getState().getWorksheetByName(tab.worksheet);
    if (!worksheet || !isWorksheetWritableV1(worksheet)) return;

    abortAutoSave();

    const statementToSave = tab.statement;
    const tabId = tab.id;

    const controller = new AbortController();
    setAutoSaveController(controller);
    tabsState.updateTab(tabId, { status: "SAVING" });

    let wasAborted = false;
    try {
      await maybeUpdateWorksheet({
        tabId,
        worksheet: tab.worksheet,
        database: tab.connection.database,
        statement: statementToSave,
        signal: controller.signal,
      });
    } catch (error) {
      if (error instanceof Error && error.name === "AbortError") {
        wasAborted = true;
        return;
      }
      if (getSQLEditorTabsState().tabsById.get(tabId)?.status === "SAVING") {
        getSQLEditorTabsState().updateTab(tabId, { status: "DIRTY" });
      }
      useAppStore.getState().notify({
        module: "bytebase",
        style: "CRITICAL",
        title: "Auto-save failed",
        description: error instanceof Error ? error.message : "Unknown error",
      });
    } finally {
      setAutoSaveController(null);
      if (!wasAborted) {
        const currentStatement =
          getSQLEditorTabsState().tabsById.get(tabId)?.statement;
        if (currentStatement !== statementToSave) {
          getSQLEditorTabsState().updateTab(tabId, { status: "DIRTY" });
        }
      }
    }
  };
}
