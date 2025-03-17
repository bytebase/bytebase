<template>
  <div
    v-if="metadata?.schema"
    class="px-2 py-2 gap-y-2 h-full overflow-hidden flex flex-col"
  >
    <div
      v-show="!metadata.table"
      class="w-full flex flex-row gap-x-2 justify-between items-center"
    >
      <div class="flex items-center justify-start gap-2">
        <DatabaseChooser />
        <SchemaSelectToolbar simple />
      </div>
      <div class="flex items-center justify-end">
        <SearchBox
          v-model:value="state.keywords.table"
          size="small"
          style="width: 10rem"
        />
      </div>
    </div>
    <TablesTable
      v-show="!metadata.table"
      :db="database"
      :database="metadata.database"
      :schema="metadata.schema"
      :tables="metadata.schema.tables"
      :keyword="state.keywords.table"
      @click="select"
    />

    <TableDetail
      v-if="metadata.table"
      :db="database"
      :database="metadata.database"
      :schema="metadata.schema"
      :table="metadata.table"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive } from "vue";
import { SearchBox } from "@/components/v2";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
} from "@/store";
import {
  DatabaseMetadata,
  SchemaMetadata,
  TableMetadata,
  TablePartitionMetadata,
} from "@/types/proto/v1/database_service";
import DatabaseChooser from "@/views/sql-editor/EditorCommon/DatabaseChooser.vue";
import { useEditorPanelContext } from "../../context";
import { SchemaSelectToolbar } from "../common";
import TableDetail from "./TableDetail.vue";
import TablesTable from "./TablesTable.vue";

const { database } = useConnectionOfCurrentSQLEditorTab();
const { viewState, updateViewState } = useEditorPanelContext();
const databaseMetadata = computed(() => {
  return useDBSchemaV1Store().getDatabaseMetadata(database.value.name);
});
const state = reactive({
  keywords: {
    table: "",
  },
});

const metadata = computed(() => {
  const database = databaseMetadata.value;
  const schema = database.schemas.find(
    (s) => s.name === viewState.value?.schema
  );
  const table = schema?.tables.find(
    (t) => t.name === viewState.value?.detail?.table
  );
  return { database, schema, table };
});

const select = (selected: {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table?: TableMetadata;
  partition?: TablePartitionMetadata;
}) => {
  updateViewState({
    detail: {
      table: selected.table?.name,
      partition: selected.partition?.name,
    },
  });
};
</script>
