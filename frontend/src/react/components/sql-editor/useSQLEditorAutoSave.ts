import { useEffect, useRef } from "react";
import { useVueState } from "@/react/hooks/useVueState";
import {
  pushNotification,
  useSQLEditorTabStore,
  useSQLEditorWorksheetStore,
  useWorkSheetStore,
} from "@/store";
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
  const tabStore = useSQLEditorTabStore();
  const worksheetStore = useWorkSheetStore();
  const sqlEditorWorksheetStore = useSQLEditorWorksheetStore();

  const statement = useVueState(() => tabStore.currentTab?.statement);
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
    const tab = tabStore.currentTab;
    if (!tab || !tab.worksheet || tab.status === "CLEAN") return;

    const worksheet = worksheetStore.getWorksheetByName(tab.worksheet);
    if (!worksheet || !isWorksheetWritableV1(worksheet)) return;

    sqlEditorWorksheetStore.abortAutoSave();

    const statementToSave = tab.statement;
    const tabId = tab.id;

    const controller = new AbortController();
    sqlEditorWorksheetStore.autoSaveController = controller;
    tabStore.updateTab(tabId, { status: "SAVING" });

    let wasAborted = false;
    try {
      await sqlEditorWorksheetStore.maybeUpdateWorksheet({
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
      if (tabStore.getTabById(tabId)?.status === "SAVING") {
        tabStore.updateTab(tabId, { status: "DIRTY" });
      }
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: "Auto-save failed",
        description: error instanceof Error ? error.message : "Unknown error",
      });
    } finally {
      sqlEditorWorksheetStore.autoSaveController = null;
      if (!wasAborted) {
        const currentStatement = tabStore.getTabById(tabId)?.statement;
        if (currentStatement !== statementToSave) {
          tabStore.updateTab(tabId, { status: "DIRTY" });
        }
      }
    }
  };
}
