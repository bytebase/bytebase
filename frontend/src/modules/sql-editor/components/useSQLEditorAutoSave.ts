import { useEffect, useRef, useState } from "react";
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
 * non-whitespace SQL; saved queries are updated in place. A newer edit waits
 * for any active save to finish, then saves the latest statement. Errors and
 * aborted saves return the tab to DIRTY without retrying unchanged content.
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
  const database = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.connection.database
  );
  const currentTabId = useSQLEditorTabState((s) => s.currentTabId);
  const [saveVersion, setSaveVersion] = useState(0);
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
    // A completed save may reveal newer SQL or a newer database selection.
    // The active tab is included because the save reads live tab state at fire.
  }, [statement, database, currentTabId, saveVersion]);

  const runAutoSave = async () => {
    const tabsState = getSQLEditorTabsState();
    const tab = tabsState.tabsById.get(tabsState.currentTabId);
    if (
      !tab ||
      tab.status === "CLEAN" ||
      tab.status === "SAVING" ||
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
    const databaseToSave = tab.connection.database;
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
          database: databaseToSave,
          statement: statementToSave,
          signal: controller.signal,
        });
      } else {
        await createSavedQuery({
          tabId,
          database: databaseToSave,
          statement: statementToSave,
          signal: controller.signal,
        });
      }
    } catch (error) {
      if (
        typeof error === "object" &&
        error !== null &&
        "name" in error &&
        error.name === "AbortError"
      ) {
        wasAborted = true;
        if (getSQLEditorTabsState().tabsById.get(tabId)?.status === "SAVING") {
          getSQLEditorTabsState().updateTab(tabId, { status: "DIRTY" });
        }
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
      if (useSQLEditorStore.getState().autoSaveController === controller) {
        setAutoSaveController(null);
      }
      if (!wasAborted) {
        const currentTab = getSQLEditorTabsState().tabsById.get(tabId);
        if (
          currentTab?.statement !== statementToSave ||
          currentTab?.connection.database !== databaseToSave
        ) {
          getSQLEditorTabsState().updateTab(tabId, { status: "DIRTY" });
          setSaveVersion((version) => version + 1);
        }
      }
    }
  };
}
