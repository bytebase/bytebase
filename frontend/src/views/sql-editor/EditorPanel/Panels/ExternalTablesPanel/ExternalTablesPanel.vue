<template>
  <div
    v-if="metadata?.schema"
    class="px-2 py-2 gap-y-2 h-full overflow-hidden flex flex-col"
  >
    <div
      v-show="!metadata.externalTable"
      class="w-full flex flex-row gap-x-2 justify-end items-center"
    >
      <SearchBox
        v-model:value="state.keywords.table"
        size="small"
        style="width: 10rem"
      />
    </div>
    <ExternalTablesTable
      v-show="!metadata.externalTable"
      :db="database"
      :database="metadata.database"
      :schema="metadata.schema"
      :external-tables="metadata.schema.externalTables"
      :keyword="state.keywords.table"
      @click="select"
    />

    <template v-if="metadata.externalTable">
      <div
        class="w-full h-7 flex flex-row gap-x-2 justify-between items-center"
      >
        <div class="flex items-center justify-start">
          <NButton text @click="deselect">
            <ChevronLeftIcon class="w-5 h-5" />
            <div class="flex items-center gap-1">
              <TableIcon class="w-4 h-4" />
              <span>{{ metadata.externalTable.name }}</span>
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
      <ExternalTableColumnsTable
        :db="database"
        :database="metadata.database"
        :schema="metadata.schema"
        :external-table="metadata.externalTable"
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
import type {
  DatabaseMetadata,
  ExternalTableMetadata,
  SchemaMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { useCurrentTabViewStateContext } from "../../context/viewState";
import ExternalTableColumnsTable from "./ExternalTableColumnsTable.vue";
import ExternalTablesTable from "./ExternalTablesTable.vue";

const { database } = useConnectionOfCurrentSQLEditorTab();
const { viewState, updateViewState } = useCurrentTabViewStateContext();
const databaseMetadata = computed(() => {
  return useDBSchemaV1Store().getDatabaseMetadata(database.value.name);
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
  const externalTable = schema?.externalTables.find(
    (t) => t.name === viewState.value?.detail?.externalTable
  );
  return { database, schema, externalTable };
});

const select = (selected: {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  externalTable: ExternalTableMetadata;
}) => {
  updateViewState({
    detail: { externalTable: selected.externalTable.name },
  });
};

const deselect = () => {
  updateViewState({
    detail: {},
  });
};
</script>
