<template>
  <div class="flex-1 flex items-stretch overflow-hidden">
    <GutterBar class="border-r border-control-border" :class="gutterBarClass" />

    <div class="flex-1 overflow-y-hidden overflow-x-auto" :class="contentClass">
      <slot v-if="!viewState || viewState.view === 'CODE'" name="code-panel" />

      <template v-if="viewState">
        <InfoPanel v-if="viewState.view === 'INFO'" :key="tab?.id" />
        <TablesPanel v-if="viewState.view === 'TABLES'" :key="tab?.id" />
        <ViewsPanel v-if="viewState.view === 'VIEWS'" :key="tab?.id" />
        <FunctionsPanel v-if="viewState.view === 'FUNCTIONS'" :key="tab?.id" />
        <ProceduresPanel
          v-if="viewState.view === 'PROCEDURES'"
          :key="tab?.id"
        />
        <SequencesPanel v-if="viewState.view === 'SEQUENCES'" :key="tab?.id" />
        <TriggersPanel v-if="viewState.view === 'TRIGGERS'" :key="tab?.id" />
        <PackagesPanel v-if="viewState.view === 'PACKAGES'" :key="tab?.id" />
        <ExternalTablesPanel
          v-if="viewState.view === 'EXTERNAL_TABLES'"
          :key="tab?.id"
        />
        <DiagramPanel v-if="viewState.view === 'DIAGRAM'" :key="tab?.id" />
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
import { DatabaseMetadataView } from "@/types/proto/v1/database_service";
import {
  extractDatabaseResourceName,
  nextAnimationFrame,
  type VueClass,
} from "@/utils";
import GutterBar from "../GutterBar";
import { useEditorPanelContext } from "../context";
import DiagramPanel from "./DiagramPanel";
import ExternalTablesPanel from "./ExternalTablesPanel";
import FunctionsPanel from "./FunctionsPanel";
import InfoPanel from "./InfoPanel";
import PackagesPanel from "./PackagesPanel";
import ProceduresPanel from "./ProceduresPanel";
import SequencesPanel from "./SequencesPanel";
import TablesPanel from "./TablesPanel";
import TriggersPanel from "./TriggersPanel";
import ViewsPanel from "./ViewsPanel";

defineProps<{
  gutterBarClass?: VueClass;
  contentClass?: VueClass;
}>();

const { currentTab: tab } = storeToRefs(useSQLEditorTabStore());
const { viewState, selectedSchemaName, updateViewState } =
  useEditorPanelContext();
const { database } = useConnectionOfCurrentSQLEditorTab();
const { execute } = useExecuteSQL();
const { events: AIEvents } = useAIContext();
const databaseMetadata = computedAsync(() => {
  if (!isValidDatabaseName(database.value.name)) {
    return undefined;
  }
  return useDBSchemaV1Store().getOrFetchDatabaseMetadata({
    database: database.value.name,
    view: DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL,
    silent: true,
  });
});

watch(
  [() => tab.value?.id, databaseMetadata, selectedSchemaName],
  ([id, database, schema]) => {
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
  const database = useDatabaseV1Store().getDatabaseByName(connection.database);
  execute({
    connection,
    statement,
    engine: database.instanceResource.engine,
    explain: false,
    selection: tab.value.editorState.selection,
  });
});
</script>
