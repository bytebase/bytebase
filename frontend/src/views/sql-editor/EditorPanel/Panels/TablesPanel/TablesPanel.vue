<template>
  <div class="px-2 pt-0 pb-2 h-full overflow-hidden">
    <DatabaseEditor
      v-model:selected-schema-name="selectedSchemaName"
      :db="database"
      :database="databaseMetadata"
      :search-pattern="''"
    />
  </div>
</template>

<script setup lang="ts">
import { first } from "lodash-es";
import { computed, ref, watch } from "vue";
import {
  provideSchemaEditorContext,
  type EditTarget,
} from "@/components/SchemaEditorLite";
import DatabaseEditor from "@/components/SchemaEditorLite/Panels/DatabaseEditor.vue";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
  useProjectV1Store,
  useSQLEditorStore,
} from "@/store";
import { DatabaseMetadataView } from "@/types/proto/v1/database_service";
import { useEditorPanelContext } from "../../context";

const editorStore = useSQLEditorStore();
const { database } = useConnectionOfCurrentSQLEditorTab();
const { selectedSchemaName } = useEditorPanelContext();
const databaseMetadata = computed(() => {
  return useDBSchemaV1Store().getDatabaseMetadata(
    database.value.name,
    DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL
  );
});

const targets = computed(() => {
  const target: EditTarget = {
    database: database.value,
    metadata: databaseMetadata.value,
    baselineMetadata: databaseMetadata.value,
  };
  return [target];
});

watch(
  [databaseMetadata, selectedSchemaName],
  ([metadata, schema]) => {
    if (metadata && schema === undefined) {
      selectedSchemaName.value = first(metadata.schemas)?.name;
    }
  },
  { immediate: true }
);

provideSchemaEditorContext({
  targets,
  project: computed(() =>
    useProjectV1Store().getProjectByName(editorStore.project)
  ),
  resourceType: ref("branch"),
  readonly: ref(true),
  selectedRolloutObjects: ref(undefined),
  showLastUpdater: ref(false),
  disableDiffColoring: ref(true),
});
</script>
