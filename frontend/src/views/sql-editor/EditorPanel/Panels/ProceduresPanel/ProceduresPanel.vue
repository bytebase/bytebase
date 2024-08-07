<template>
  <div
    v-if="metadata?.schema"
    class="px-2 py-2 gap-y-2 h-full overflow-hidden flex flex-col"
  >
    <template v-if="!metadata.procedure">
      <SchemaSelectToolbar />
      <ProceduresTable
        v-if="!metadata.procedure"
        :db="database"
        :database="metadata.database"
        :schema="metadata.schema"
        :procedures="metadata.schema.procedures"
        @click="select"
      />
    </template>

    <template v-if="metadata.procedure">
      <CodeViewer
        :db="database"
        :code="metadata.procedure.definition"
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
  ProcedureMetadata,
  SchemaMetadata,
} from "@/types/proto/v1/database_service";
import { useEditorPanelContext } from "../../context";
import { SchemaSelectToolbar, CodeViewer } from "../common";
import ProceduresTable from "./ProceduresTable.vue";

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
  procedure?: ProcedureMetadata;
}>();

const select = (selected: {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  procedure: ProcedureMetadata;
}) => {
  metadata.value = selected;
};

const deselect = () => {
  if (!metadata.value) return;
  metadata.value.procedure = undefined;
};

watch(
  [databaseMetadata, selectedSchemaName],
  ([database, schema]) => {
    metadata.value = {
      database,
      schema: database.schemas.find((s) => s.name === schema),
      procedure: undefined,
    };
  },
  { immediate: true }
);
</script>
