import { useLocalStorage } from "@vueuse/core";
import { defineStore } from "pinia";
import { computed, ref } from "vue";
import type { GetSchemaStringRequest_ObjectType } from "@/types/proto-es/v1/database_service_pb";
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
  const schemaViewer = ref<
    | {
        schema?: string;
        object?: string;
        type?: GetSchemaStringRequest_ObjectType;
      }
    | undefined
  >(undefined);
  const pendingInsertAtCaret = ref<string | undefined>();
  const highlightAccessGrantName = ref<string | undefined>();

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
    schemaViewer,
    pendingInsertAtCaret,
    highlightAccessGrantName,
    editorPanelSize,
    handleEditorPanelResize,
  };
});
