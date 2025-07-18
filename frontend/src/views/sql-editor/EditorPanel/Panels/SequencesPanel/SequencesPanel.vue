<template>
  <div v-if="metadata?.schema" class="h-full overflow-hidden flex flex-col">
    <div
      v-show="!metadata.func"
      class="w-full h-[44px] py-2 px-2 border-b flex flex-row gap-x-2 justify-end items-center"
    >
      <SearchBox
        v-model:value="state.keyword"
        size="small"
        style="width: 10rem"
      />
    </div>
    <SequencesTable
      :db="database"
      :database="metadata.database"
      :schema="metadata.schema"
      :sequences="metadata.schema.sequences"
      :keyword="state.keyword"
      @click="select"
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
import type {
  DatabaseMetadata,
  SchemaMetadata,
  SequenceMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import {
  extractKeyWithPosition,
  keyWithPosition,
} from "@/views/sql-editor/EditorCommon";
import { useCurrentTabViewStateContext } from "../../context/viewState";
import SequencesTable from "./SequencesTable.vue";

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
    viewState.value?.detail?.func ?? ""
  );
  const func = schema?.functions.find(
    (f, i) => f.name === name && i === position
  );
  return { database, schema, func };
});

const select = (selected: {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  sequence: SequenceMetadata;
  position: number;
}) => {
  updateViewState({
    detail: {
      sequence: keyWithPosition(selected.sequence.name, selected.position),
    },
  });
};
</script>
