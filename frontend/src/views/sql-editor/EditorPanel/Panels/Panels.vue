<template>
  <div class="flex-1 flex items-stretch overflow-hidden">
    <div class="flex-1 overflow-y-hidden overflow-x-auto" :class="contentClass">
      <slot v-if="!viewState || viewState.view === 'CODE'" name="code-panel" />

      <template v-if="viewState && viewState.view !== 'CODE'">
        <div class="h-full flex flex-col">
          <div
            class="py-2 px-2 w-full flex flex-row gap-x-2 justify-between items-center"
          >
            <div class="flex items-center justify-start gap-2">
              <ReactPageMount page="DatabaseChooser" :disabled="true" />
              <ReactPageMount
                page="SchemaSelectToolbar"
                container-class="w-fit"
              />
            </div>
          </div>
          <NSplit
            class="flex-1"
            :disabled="!showAIPaneAlongsidePanel"
            :size="
              showAIPaneAlongsidePanel ? editorPanelSize.size : 1
            "
            :min="showAIPaneAlongsidePanel ? editorPanelSize.min : 1"
            :max="showAIPaneAlongsidePanel ? editorPanelSize.max : 1"
            :resize-trigger-size="3"
            @update:size="handleEditorPanelResize"
          >
            <template #1>
              <ReactPageMount
                v-if="viewState.view === 'INFO'"
                :key="`info-${tab?.id}`"
                page="InfoPanel"
              />
              <ReactPageMount
                v-if="viewState.view === 'TABLES'"
                :key="`tables-${tab?.id}`"
                page="TablesPanel"
              />
              <ReactPageMount
                v-if="viewState.view === 'VIEWS'"
                :key="`views-${tab?.id}`"
                page="ViewsPanel"
              />
              <ReactPageMount
                v-if="viewState.view === 'FUNCTIONS'"
                :key="`functions-${tab?.id}`"
                page="FunctionsPanel"
              />
              <ReactPageMount
                v-if="viewState.view === 'PROCEDURES'"
                :key="`procedures-${tab?.id}`"
                page="ProceduresPanel"
              />
              <ReactPageMount
                v-if="viewState.view === 'SEQUENCES'"
                :key="`sequences-${tab?.id}`"
                page="SequencesPanel"
              />
              <ReactPageMount
                v-if="viewState.view === 'PACKAGES'"
                :key="`packages-${tab?.id}`"
                page="PackagesPanel"
              />
              <ReactPageMount
                v-if="viewState.view === 'TRIGGERS'"
                :key="`triggers-${tab?.id}`"
                page="TriggersPanel"
              />
              <ReactPageMount
                v-if="viewState.view === 'EXTERNAL_TABLES'"
                :key="`external-tables-${tab?.id}`"
                page="ExternalTablesPanel"
              />
              <ReactPageMount
                v-if="viewState.view === 'DIAGRAM'"
                :key="`diagram-${tab?.id}`"
                page="DiagramPanel"
              />
            </template>
            <template v-if="showAIPaneAlongsidePanel" #2>
              <div class="h-full overflow-hidden flex flex-col">
                <Suspense>
                  <AIChatToSQL key="ai-chat-to-sql" />
                  <template #fallback>
                    <div
                      class="w-full h-full grow flex flex-col items-center justify-center"
                    >
                      <BBSpin />
                    </div>
                  </template>
                </Suspense>
              </div>
            </template>
          </NSplit>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computedAsync } from "@vueuse/core";
import { first } from "lodash-es";
import { NSplit } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, watch } from "vue";
import { BBSpin } from "@/bbkit";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { AIChatToSQL } from "@/plugins/ai";
import { useAIContext } from "@/plugins/ai/logic";
import ReactPageMount from "@/react/ReactPageMount.vue";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDatabaseV1Store,
  useDBSchemaV1Store,
  useSQLEditorTabStore,
  useSQLEditorUIStore,
} from "@/store";
import { isValidDatabaseName } from "@/types";
import type { DatabaseMetadata } from "@/types/proto-es/v1/database_service_pb";
import {
  extractDatabaseResourceName,
  getInstanceResource,
  nextAnimationFrame,
  type VueClass,
} from "@/utils";
import { useCurrentTabViewStateContext } from "../context/viewState.tsx";

defineProps<{
  contentClass?: VueClass;
}>();

const { currentTab: tab } = storeToRefs(useSQLEditorTabStore());
const { viewState, selectedSchemaName, updateViewState } =
  useCurrentTabViewStateContext();
const { database } = useConnectionOfCurrentSQLEditorTab();
const { execute } = useExecuteSQL();
const { events: AIEvents } = useAIContext();

// AI pane host: when a CodeViewer-style React surface is mounted under
// this panel (`isShowingCode`), and the user has toggled the AI panel
// open (`showAIPanel`), render `AIChatToSQL` to the right via NSplit.
// This mirrors the Vue `CodeViewer`'s NSplit but hoists the host one
// level up so React panels don't need to embed the Vue chat panel.
const uiStore = useSQLEditorUIStore();
const { showAIPanel, isShowingCode, editorPanelSize } = storeToRefs(uiStore);
const handleEditorPanelResize = uiStore.handleEditorPanelResize;
const showAIPaneAlongsidePanel = computed(
  () => showAIPanel.value && isShowingCode.value
);
const databaseMetadata = computedAsync(() => {
  if (!isValidDatabaseName(database.value.name)) {
    return undefined;
  }
  return useDBSchemaV1Store().getOrFetchDatabaseMetadata({
    database: database.value.name,
    silent: true,
  });
});

watch(
  [() => tab.value?.id, databaseMetadata, selectedSchemaName],
  ([id, database, schema]: [
    string | undefined,
    DatabaseMetadata | undefined,
    string | undefined,
  ]) => {
    if (!id) return;
    if (!database) return;
    if (
      !isValidDatabaseName(extractDatabaseResourceName(database.name).database)
    ) {
      return;
    }
    if (!schema || database.schemas.findIndex((s) => s.name === schema) < 0) {
      selectedSchemaName.value = first(database.schemas)?.name;
    }
  },
  { immediate: true, flush: "post" }
);

useEmitteryEventListener(AIEvents, "run-statement", async ({ statement }) => {
  if (!tab.value) {
    return;
  }
  updateViewState({
    view: "CODE",
  });
  await nextAnimationFrame();
  const connection = tab.value.connection;
  const database = await useDatabaseV1Store().getOrFetchDatabaseByName(
    connection.database
  );
  execute({
    connection,
    statement,
    engine: getInstanceResource(database).engine,
    explain: false,
    selection: tab.value.editorState.selection,
  });
});
</script>
