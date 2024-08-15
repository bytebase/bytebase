<template>
  <div
    v-if="metadata?.schema"
    class="px-2 py-2 gap-y-2 h-full overflow-hidden flex flex-col"
  >
    <SchemaSelectToolbar v-show="!metadata.view" />
    <ViewsTable
      v-show="!metadata.view"
      :db="database"
      :database="metadata.database"
      :schema="metadata.schema"
      :views="metadata.schema.views"
      @click="select"
    />

    <template v-if="metadata.view">
      <CodeViewer
        :db="database"
        :title="metadata.view.name"
        :code="metadata.view.definition"
        @back="deselect"
      >
        <template #title-icon> </template>
        <ViewIcon class="w-4 h-4" />
      </CodeViewer>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { ViewIcon } from "@/components/Icon";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
} from "@/store";
import {
  DatabaseMetadata,
  DatabaseMetadataView,
  ViewMetadata,
  SchemaMetadata,
} from "@/types/proto/v1/database_service";
import { useEditorPanelContext } from "../../context";
import { SchemaSelectToolbar, CodeViewer } from "../common";
import ViewsTable from "./ViewsTable.vue";

const { database } = useConnectionOfCurrentSQLEditorTab();
const { selectedSchemaName } = useEditorPanelContext();
const databaseMetadata = computed(() => {
  return useDBSchemaV1Store().getDatabaseMetadata(
    database.value.name,
    DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL
  );
});

const metadata = ref<{
  database: DatabaseMetadata;
  schema?: SchemaMetadata;
  view?: ViewMetadata;
}>();

const select = (selected: {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  view: ViewMetadata;
}) => {
  metadata.value = selected;
};

const deselect = () => {
  if (!metadata.value) return;
  metadata.value.view = undefined;
};

watch(
  [databaseMetadata, selectedSchemaName],
  ([database, schema]) => {
    metadata.value = {
      database,
      schema: database.schemas.find((s) => s.name === schema),
      view: undefined,
    };
  },
  { immediate: true }
);
</script>
