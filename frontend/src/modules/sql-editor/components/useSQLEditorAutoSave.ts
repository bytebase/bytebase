import { useEffect, useRef } from "react";
import { useSQLEditorStore } from "@/modules/sql-editor/store";
import { getSQLEditorEditorState } from "@/modules/sql-editor/store/editor";
import {
  getSQLEditorTabsState,
  useSQLEditorTabState,
} from "@/modules/sql-editor/store/tab";
import { useAppStore } from "@/stores/app";
import { canCreateSavedQueryInProject, isSavedQueryWritableV1 } from "@/utils";

const AUTO_SAVE_DEBOUNCE_MS = 2000;

/**
 * Watches the active tab's `statement` and after a 2s debounce persists a
 * dirty, writable tab. Local drafts become saved queries after their first
 * non-whitespace SQL; saved queries are updated in place. Aborts any in-flight
 * auto-save when a newer one starts, reverts the tab to DIRTY on error (unless
 * aborted), and re-flags DIRTY when the statement keeps changing during save.
 *
 * Mounted once at the SQL Editor layout level; safe to call from any
 * component but should only be active while the SQL Editor route is.
 */
export function useSQLEditorAutoSave() {
  const abortAutoSave = useSQLEditorStore((s) => s.abortAutoSave);
  const setAutoSaveController = useSQLEditorStore(
    (s) => s.setAutoSaveController
  );
  const maybeUpdateSavedQuery = useSQLEditorStore(
    (s) => s.maybeUpdateSavedQuery
  );
  const createSavedQuery = useSQLEditorStore((s) => s.createSavedQuery);

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
    // reads tab.statement from the tab store at fire time, so capturing
    // only the change-trigger here is sufficient.
  }, [statement]);

  const runAutoSave = async () => {
    const tabsState = getSQLEditorTabsState();
    const tab = tabsState.tabsById.get(tabsState.currentTabId);
    if (
      !tab ||
      tab.status === "CLEAN" ||
      (!tab.savedQuery && !tab.statement.trim())
    ) {
      return;
    }

    if (tab.savedQuery) {
      const savedQuery = useAppStore
        .getState()
        .getSavedQueryByName(tab.savedQuery);
      if (!savedQuery || !isSavedQueryWritableV1(savedQuery)) return;
    } else if (
      !canCreateSavedQueryInProject(getSQLEditorEditorState().project)
    ) {
      return;
    }

    abortAutoSave();

    const statementToSave = tab.statement;
    const tabId = tab.id;

    const controller = new AbortController();
    setAutoSaveController(controller);
    tabsState.updateTab(tabId, { status: "SAVING" });

    let wasAborted = false;
    try {
      if (tab.savedQuery) {
        await maybeUpdateSavedQuery({
          tabId,
          savedQuery: tab.savedQuery,
          database: tab.connection.database,
          statement: statementToSave,
          signal: controller.signal,
        });
      } else {
        await createSavedQuery({
          tabId,
          database: tab.connection.database,
          statement: statementToSave,
        });
      }
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
