<template>
  <div class="px-2 pt-0 pb-2 h-full overflow-hidden">
    <DatabaseEditor
      :db="database"
      :database="databaseMetadata"
      :selected-schema-name="'public'"
      :search-pattern="''"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
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

const editorStore = useSQLEditorStore();
const { database } = useConnectionOfCurrentSQLEditorTab();
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
