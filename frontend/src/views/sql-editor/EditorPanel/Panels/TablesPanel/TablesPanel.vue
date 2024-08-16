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

    <template v-if="metadata.table">
      <div
        class="w-full h-[28px] flex flex-row gap-x-2 justify-between items-center"
      >
        <div class="flex items-center justify-start">
          <NButton text @click="deselect">
            <ChevronLeftIcon class="w-5 h-5" />
            <div class="flex items-center gap-1">
              <TableIcon class="w-4 h-4" />
              <span>{{ metadata.table.name }}</span>
            </div>
          </NButton>
        </div>
        <div class="flex items-center justify-end">
          <SearchBox
            v-model:value="state.keywords.column"
            size="small"
            style="width: 10rem"
          />
        </div>
      </div>
      <ColumnsTable
        :db="database"
        :database="metadata.database"
        :schema="metadata.schema"
        :table="metadata.table"
        :keyword="state.keywords.column"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { ChevronLeftIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, reactive } from "vue";
import { TableIcon } from "@/components/Icon";
import { SearchBox } from "@/components/v2";
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
import DatabaseChooser from "@/views/sql-editor/EditorCommon/DatabaseChooser.vue";
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
const state = reactive({
  keywords: {
    table: "",
    column: "",
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
