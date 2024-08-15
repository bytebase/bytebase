<template>
  <div
    v-if="metadata?.schema"
    class="px-2 py-2 gap-y-2 h-full overflow-hidden flex flex-col"
  >
    <SchemaSelectToolbar v-show="!metadata.procedure" />
    <ProceduresTable
      v-show="!metadata.procedure"
      :db="database"
      :database="metadata.database"
      :schema="metadata.schema"
      :procedures="metadata.schema.procedures"
      @click="select"
    />

    <template v-if="metadata.procedure">
      <CodeViewer
        :db="database"
        :title="metadata.procedure.name"
        :code="metadata.procedure.definition"
        @back="deselect"
      >
        <template #title-icon>
          <ProcedureIcon class="w-4 h-4 text-main" />
        </template>
      </CodeViewer>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { ProcedureIcon } from "@/components/Icon";
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
const { viewState, updateViewState } = useEditorPanelContext();
const databaseMetadata = computed(() => {
  return useDBSchemaV1Store().getDatabaseMetadata(
    database.value.name,
    DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL
  );
});

const metadata = computed(() => {
  const database = databaseMetadata.value;
  const schema = database.schemas.find(
    (s) => s.name === viewState.value?.schema
  );
  const procedure = schema?.procedures.find(
    (p) => p.name === viewState.value?.detail?.procedure
  );
  return { database, schema, procedure };
});

const select = (selected: {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  procedure: ProcedureMetadata;
}) => {
  updateViewState({
    detail: { procedure: selected.procedure.name },
  });
};

const deselect = () => {
  updateViewState({
    detail: {},
  });
};
</script>
