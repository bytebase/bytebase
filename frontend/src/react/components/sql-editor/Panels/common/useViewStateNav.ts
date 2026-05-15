import { useCallback } from "react";
import { useVueState } from "@/react/hooks/useVueState";
import { useSQLEditorTabStore } from "@/react/stores/sqlEditor/tab-vue-state";
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
  const tabStore = useSQLEditorTabStore();
  const viewState = useVueState(() => tabStore.currentTab?.viewState, {
    deep: true,
  });
  const tabId = useVueState(() => tabStore.currentTab?.id);

  const updateViewState = useCallback(
    (patch: Partial<EditorPanelViewState>) => {
      const id = tabStore.currentTab?.id;
      if (!id) return;
      tabStore.updateTab(id, {
        viewState: {
          ...defaultViewState(),
          ...tabStore.currentTab?.viewState,
          ...patch,
        },
      });
    },
    [tabStore]
  );

  const setDetail = useCallback(
    (patch: Partial<EditorPanelViewState["detail"]>) => {
      updateViewState({
        detail: {
          ...(tabStore.currentTab?.viewState.detail ?? {}),
          ...patch,
        },
      });
    },
    [tabStore, updateViewState]
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
