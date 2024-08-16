<template>
  <div
    v-if="metadata?.schema"
    class="px-2 py-2 gap-y-2 h-full overflow-hidden flex flex-col"
  >
    <div
      v-show="!metadata.func"
      class="w-full flex flex-row gap-x-2 justify-between items-center"
    >
      <div class="flex items-center justify-start">
        <SchemaSelectToolbar />
      </div>
      <div class="flex items-center justify-end">
        <DatabaseChooser />
      </div>
    </div>
    <FunctionsTable
      v-show="!metadata.func"
      :db="database"
      :database="metadata.database"
      :schema="metadata.schema"
      :funcs="metadata.schema.functions"
      @click="select"
    />

    <template v-if="metadata.func">
      <CodeViewer
        :db="database"
        :title="metadata.func.name"
        :code="metadata.func.definition"
        @back="deselect"
      >
        <template #title-icon>
          <FunctionIcon class="w-4 h-4 text-main" />
        </template>
      </CodeViewer>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { FunctionIcon } from "@/components/Icon";
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
import DatabaseChooser from "@/views/sql-editor/EditorCommon/DatabaseChooser.vue";
import { useEditorPanelContext } from "../../context";
import { SchemaSelectToolbar, CodeViewer } from "../common";
import FunctionsTable from "./FunctionsTable.vue";

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
  const func = schema?.functions.find(
    (f) => f.name === viewState.value?.detail?.func
  );
  return { database, schema, func };
});

const select = (selected: {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  func: FunctionMetadata;
}) => {
  updateViewState({
    detail: { func: selected.func.name },
  });
};

const deselect = () => {
  updateViewState({
    detail: {},
  });
};
</script>
