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
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
  useSQLEditorTabStore,
} from "@/store";
import { isValidDatabaseName } from "@/types";
import { DatabaseMetadataView } from "@/types/proto/v1/database_service";
import { extractDatabaseResourceName, type VueClass } from "@/utils";
import GutterBar from "../GutterBar";
import { useEditorPanelContext } from "../context";
import DiagramPanel from "./DiagramPanel";
import ExternalTablesPanel from "./ExternalTablesPanel";
import FunctionsPanel from "./FunctionsPanel";
import InfoPanel from "./InfoPanel";
import ProceduresPanel from "./ProceduresPanel";
import TablesPanel from "./TablesPanel";
import ViewsPanel from "./ViewsPanel";

defineProps<{
  gutterBarClass?: VueClass;
  contentClass?: VueClass;
}>();

const { currentTab: tab } = storeToRefs(useSQLEditorTabStore());
const { viewState, selectedSchemaName } = useEditorPanelContext();
const { database } = useConnectionOfCurrentSQLEditorTab();
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
</script>
