<template>
  <div v-if="metadata?.schema" class="h-full overflow-hidden flex flex-col">
    <div
      v-show="!metadata.procedure"
      class="w-full h-11 py-2 px-2 border-b flex flex-row gap-x-2 justify-end items-center"
    >
      <SearchBox
        v-model:value="state.keyword"
        size="small"
        style="width: 10rem"
      />
    </div>
    <ProceduresTable
      v-show="!metadata.procedure"
      :db="database"
      :database="metadata.database"
      :schema="metadata.schema"
      :procedures="metadata.schema.procedures"
      :keyword="state.keyword"
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
import { computed, reactive } from "vue";
import { ProcedureIcon } from "@/components/Icon";
import { SearchBox } from "@/components/v2";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
} from "@/store";
import type {
  DatabaseMetadata,
  ProcedureMetadata,
  SchemaMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import {
  extractKeyWithPosition,
  keyWithPosition,
} from "@/views/sql-editor/EditorCommon";
import { useCurrentTabViewStateContext } from "../../context/viewState";
import { CodeViewer } from "../common";
import ProceduresTable from "./ProceduresTable.vue";

const { database } = useConnectionOfCurrentSQLEditorTab();
const { viewState, updateViewState } = useCurrentTabViewStateContext();
const databaseMetadata = computed(() => {
  return useDBSchemaV1Store().getDatabaseMetadata(database.value.name);
});
const state = reactive({
  keyword: "",
});

const metadata = computed(() => {
  const database = databaseMetadata.value;
  const schema = database.schemas.find(
    (s) => s.name === viewState.value?.schema
  );
  const [name, position] = extractKeyWithPosition(
    viewState.value?.detail?.procedure ?? ""
  );
  const procedure = schema?.procedures.find(
    (p, i) => p.name === name && i === position
  );
  return { database, schema, procedure };
});

const select = (selected: {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  procedure: ProcedureMetadata;
  position: number;
}) => {
  updateViewState({
    detail: {
      procedure: keyWithPosition(selected.procedure.name, selected.position),
    },
  });
};

const deselect = () => {
  updateViewState({
    detail: {},
  });
};
</script>
