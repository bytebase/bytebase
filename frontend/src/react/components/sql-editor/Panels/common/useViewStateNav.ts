import { useCallback } from "react";
import {
  getSQLEditorTabsState,
  useSQLEditorTabState,
} from "@/react/stores/sqlEditor/tab";
import { defaultViewState } from "@/types";
import type { EditorPanelViewState } from "@/types/sqlEditor/tabViewState";

/**
 * React equivalent of Vue's `useCurrentTabViewStateContext` for the
 * read/write surface used by panels. Exposes the current tab's
 * `viewState` plus helpers to patch it. The bidirectional
 * connection.schema ↔ viewState.schema sync stays in `Panels.vue`
 * because that file remains the Vue host through Stage 16.
 */
export function useViewStateNav() {
  const viewState = useSQLEditorTabState(
    (s) => s.tabsById.get(s.currentTabId)?.viewState
  );
  const tabId = useSQLEditorTabState((s) => s.tabsById.get(s.currentTabId)?.id);

  const updateViewState = useCallback(
    (patch: Partial<EditorPanelViewState>) => {
      const tabsState = getSQLEditorTabsState();
      const currentTab = tabsState.tabsById.get(tabsState.currentTabId);
      const id = currentTab?.id;
      if (!id) return;
      tabsState.updateTab(id, {
        viewState: {
          ...defaultViewState(),
          ...currentTab?.viewState,
          ...patch,
        },
      });
    },
    []
  );

  const setDetail = useCallback(
    (patch: Partial<EditorPanelViewState["detail"]>) => {
      const tabsState = getSQLEditorTabsState();
      const currentTab = tabsState.tabsById.get(tabsState.currentTabId);
      updateViewState({
        detail: {
          ...(currentTab?.viewState.detail ?? {}),
          ...patch,
        },
      });
    },
    [updateViewState]
  );

  const clearDetail = useCallback(() => {
    updateViewState({ detail: {} });
  }, [updateViewState]);

  const setSchema = useCallback(
    (schema: string) => {
      updateViewState({ schema });
    },
    [updateViewState]
  );

  return {
    tabId,
    viewState,
    schema: viewState?.schema,
    detail: viewState?.detail,
    updateViewState,
    setDetail,
    clearDetail,
    setSchema,
  };
}
