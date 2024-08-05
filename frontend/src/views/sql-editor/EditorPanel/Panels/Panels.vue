<template>
  <div class="flex-1 flex items-stretch overflow-hidden">
    <GutterBar class="border-r border-control-border" :class="gutterBarClass" />

    <div class="flex-1 overflow-y-hidden overflow-x-auto" :class="contentClass">
      <slot v-if="!viewState || viewState.view === 'CODE'" name="code-panel" />

      <template v-if="viewState">
        <InfoPanel v-if="viewState.view === 'INFO'" />
        <TablesPanel v-if="viewState.view === 'TABLES'" />
        <ViewsPanel v-if="viewState.view === 'VIEWS'" />
        <FunctionsPanel v-if="viewState.view === 'FUNCTIONS'" />
        <ProceduresPanel v-if="viewState.view === 'PROCEDURES'" />
        <DiagramPanel v-if="viewState.view === 'DIAGRAM'" />
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { first } from "lodash-es";
import { computed, watch } from "vue";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
} from "@/store";
import { DatabaseMetadataView } from "@/types/proto/v1/database_service";
import type { VueClass } from "@/utils";
import GutterBar from "../GutterBar";
import { useEditorPanelContext } from "../context";
import DiagramPanel from "./DiagramPanel";
import FunctionsPanel from "./FunctionsPanel";
import InfoPanel from "./InfoPanel";
import ProceduresPanel from "./ProceduresPanel";
import TablesPanel from "./TablesPanel";
import ViewsPanel from "./ViewsPanel";

defineProps<{
  gutterBarClass?: VueClass;
  contentClass?: VueClass;
}>();

const { viewState, selectedSchemaName } = useEditorPanelContext();
const { database } = useConnectionOfCurrentSQLEditorTab();
const databaseMetadata = computed(() => {
  return useDBSchemaV1Store().getDatabaseMetadata(
    database.value.name,
    DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL
  );
});

watch(
  [databaseMetadata, selectedSchemaName],
  ([database, schema]) => {
    if (database && schema === undefined) {
      selectedSchemaName.value = first(database.schemas)?.name;
    }
  },
  { immediate: true }
);
</script>
