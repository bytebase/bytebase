<template>
  <div
    v-if="metadata?.schema"
    class="px-2 py-2 gap-y-2 h-full overflow-hidden flex flex-col"
  >
    <template v-if="!metadata.func">
      <SchemaSelectToolbar />
      <FunctionsTable
        v-if="!metadata.func"
        :db="database"
        :database="metadata.database"
        :schema="metadata.schema"
        :funcs="metadata.schema.functions"
        @click="select"
      />
    </template>

    <template v-if="metadata.func">
      <CodeViewer
        :db="database"
        :code="metadata.func.definition"
        @back="deselect"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
} from "@/store";
import {
  DatabaseMetadata,
  DatabaseMetadataView,
  FunctionMetadata,
  SchemaMetadata,
} from "@/types/proto/v1/database_service";
import { useEditorPanelContext } from "../../context";
import { SchemaSelectToolbar, CodeViewer } from "../common";
import FunctionsTable from "./FunctionsTable.vue";

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
  func?: FunctionMetadata;
}>();

const select = (selected: {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  func: FunctionMetadata;
}) => {
  metadata.value = selected;
};

const deselect = () => {
  if (!metadata.value) return;
  metadata.value.func = undefined;
};

watch(
  [databaseMetadata, selectedSchemaName],
  ([database, schema]) => {
    metadata.value = {
      database,
      schema: database.schemas.find((s) => s.name === schema),
      func: undefined,
    };
  },
  { immediate: true }
);
</script>
