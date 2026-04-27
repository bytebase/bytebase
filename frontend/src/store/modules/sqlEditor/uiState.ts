import { useLocalStorage } from "@vueuse/core";
import { defineStore } from "pinia";
import { computed, ref } from "vue";
import { STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE } from "@/utils/storage-keys";

export type AsidePanelTab = "SCHEMA" | "WORKSHEET" | "HISTORY" | "ACCESS";

const minimumEditorPanelSize = 0.5;

/**
 * UI state for the SQL Editor shell.
 *
 * Extracted from `useSQLEditorContext` so React leaves can access the same
 * reactive state that Vue consumers read via inject. Vue's
 * `useSQLEditorContext()` wraps this store via `storeToRefs` and preserves
 * its existing API shape.
 */
export const useSQLEditorUIStore = defineStore("sqlEditorUI", () => {
  const asidePanelTab = ref<AsidePanelTab>("WORKSHEET");
  const showConnectionPanel = ref(false);
  const showAIPanel = ref(false);
  const pendingInsertAtCaret = ref<string | undefined>();
  const highlightAccessGrantName = ref<string | undefined>();
  // True while a CodeViewer-style surface is mounted (procedure/function/
  // package body, view definition, trigger body). Panels.vue uses this to
  // decide whether to render the AIChatToSQL side pane — matches the Vue
  // CodeViewer's behavior of unmounting AIChatToSQL when the user navigates
  // back to the list view.
  const isShowingCode = ref(false);

  const aiPanelSize = useLocalStorage(
    STORAGE_KEY_SQL_EDITOR_AI_PANEL_SIZE,
    0.3
  );

  const editorPanelSize = computed(() => {
    if (!showAIPanel.value) {
      return { size: 1, max: 1, min: 1 };
    }
    return {
      size: Math.max(1 - aiPanelSize.value, minimumEditorPanelSize),
      max: 0.9,
      min: minimumEditorPanelSize,
    };
  });

  const handleEditorPanelResize = (size: number) => {
    if (size >= 1) return;
    aiPanelSize.value = 1 - size;
  };

  return {
    asidePanelTab,
    showConnectionPanel,
    showAIPanel,
    pendingInsertAtCaret,
    highlightAccessGrantName,
    isShowingCode,
    editorPanelSize,
    handleEditorPanelResize,
  };
});
