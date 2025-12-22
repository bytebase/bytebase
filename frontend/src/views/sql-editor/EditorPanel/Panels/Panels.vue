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
              <DatabaseChooser :disabled="true" />
              <SchemaSelectToolbar simple />
            </div>
          </div>
          <div class="flex-1">
            <InfoPanel v-if="viewState.view === 'INFO'" :key="tab?.id" />
            <TablesPanel v-if="viewState.view === 'TABLES'" :key="tab?.id" />
            <ViewsPanel v-if="viewState.view === 'VIEWS'" :key="tab?.id" />
            <FunctionsPanel
              v-if="viewState.view === 'FUNCTIONS'"
              :key="tab?.id"
            />
            <ProceduresPanel
              v-if="viewState.view === 'PROCEDURES'"
              :key="tab?.id"
            />
            <SequencesPanel
              v-if="viewState.view === 'SEQUENCES'"
              :key="tab?.id"
            />
            <PackagesPanel
              v-if="viewState.view === 'PACKAGES'"
              :key="tab?.id"
            />
            <TriggersPanel
              v-if="viewState.view === 'TRIGGERS'"
              :key="tab?.id"
            />
            <ExternalTablesPanel
              v-if="viewState.view === 'EXTERNAL_TABLES'"
              :key="tab?.id"
            />
            <DiagramPanel v-if="viewState.view === 'DIAGRAM'" :key="tab?.id" />
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computedAsync } from "@vueuse/core";
import { first } from "lodash-es";
import { storeToRefs } from "pinia";
import { watch } from "vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import { useExecuteSQL } from "@/composables/useExecuteSQL";
import { useAIContext } from "@/plugins/ai/logic";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDatabaseV1Store,
  useDBSchemaV1Store,
  useSQLEditorTabStore,
} from "@/store";
import { isValidDatabaseName } from "@/types";
import type { DatabaseMetadata } from "@/types/proto-es/v1/database_service_pb";
import {
  extractDatabaseResourceName,
  nextAnimationFrame,
  type VueClass,
} from "@/utils";
import DatabaseChooser from "@/views/sql-editor/EditorCommon/DatabaseChooser.vue";
import { useCurrentTabViewStateContext } from "../context/viewState.tsx";
import { SchemaSelectToolbar } from "./common";
import DiagramPanel from "./DiagramPanel";
import ExternalTablesPanel from "./ExternalTablesPanel";
import FunctionsPanel from "./FunctionsPanel";
import InfoPanel from "./InfoPanel";
import PackagesPanel from "./PackagesPanel";
import ProceduresPanel from "./ProceduresPanel";
import SequencesPanel from "./SequencesPanel";
import TablesPanel from "./TablesPanel";
import TriggersPanel from "./TriggersPanel/TriggersPanel.vue";
import ViewsPanel from "./ViewsPanel";

defineProps<{
  contentClass?: VueClass;
}>();

const { currentTab: tab } = storeToRefs(useSQLEditorTabStore());
const { viewState, selectedSchemaName, updateViewState } =
  useCurrentTabViewStateContext();
const { database } = useConnectionOfCurrentSQLEditorTab();
const { execute } = useExecuteSQL();
const { events: AIEvents } = useAIContext();
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
    engine: database.instanceResource.engine,
    explain: false,
    selection: tab.value.editorState.selection,
  });
});
</script>
