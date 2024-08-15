<template>
  <div
    v-if="metadata?.schema"
    class="px-2 py-2 gap-y-2 h-full overflow-hidden flex flex-col"
  >
    <SchemaSelectToolbar v-show="!metadata.table" />
    <TablesTable
      v-show="!metadata.table"
      :db="database"
      :database="metadata.database"
      :schema="metadata.schema"
      :tables="metadata.schema.tables"
      @click="select"
    />

    <template v-if="metadata.table">
      <div class="w-full flex flex-row gap-x-2 justify-start items-center">
        <NButton text @click="deselect">
          <ChevronLeftIcon class="w-5 h-5" />
          <div class="flex items-center gap-1">
            <TableIcon class="w-4 h-4" />
            <span>{{ metadata.table.name }}</span>
          </div>
        </NButton>
      </div>
      <ColumnsTable
        v-if="metadata.table"
        :db="database"
        :database="metadata.database"
        :schema="metadata.schema"
        :table="metadata.table"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { ChevronLeftIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { TableIcon } from "@/components/Icon";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
} from "@/store";
import {
  DatabaseMetadata,
  DatabaseMetadataView,
  SchemaMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import { useEditorPanelContext } from "../../context";
import { SchemaSelectToolbar } from "../common";
import ColumnsTable from "./ColumnsTable.vue";
import TablesTable from "./TablesTable.vue";

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
  const table = schema?.tables.find(
    (t) => t.name === viewState.value?.detail?.table
  );
  return { database, schema, table };
});

const select = (selected: {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  table: TableMetadata;
}) => {
  updateViewState({
    detail: { table: selected.table.name },
  });
};

const deselect = () => {
  updateViewState({
    detail: {},
  });
};
</script>
