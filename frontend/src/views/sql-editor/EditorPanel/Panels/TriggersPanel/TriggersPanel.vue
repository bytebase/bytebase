<template>
  <div v-if="metadata?.schema" class="h-full overflow-hidden flex flex-col">
    <div
      v-show="!metadata.trigger"
      class="w-full h-11 py-2 px-2 border-b flex flex-row gap-x-2 justify-end items-center"
    >
      <SearchBox
        v-model:value="state.keyword"
        size="small"
        style="width: 10rem"
      />
    </div>
    <TriggersTable
      v-show="!metadata.trigger"
      :db="database"
      :database="metadata.database"
      :schema="metadata.schema"
      :table="metadata.table"
      :triggers="metadata.table?.triggers"
      :keyword="state.keyword"
      @click="select"
    />

    <template v-if="metadata.trigger">
      <CodeViewer
        :db="database"
        :title="metadata.trigger.name"
        :code="metadata.trigger.body"
        @back="deselect"
      >
        <template #title-icon>
          <TriggerIcon class="w-4 h-4 text-main" />
        </template>
      </CodeViewer>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive } from "vue";
import TriggerIcon from "@/components/Icon/TriggerIcon.vue";
import { SearchBox } from "@/components/v2";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
} from "@/store";
import type { TriggerMetadata } from "@/types/proto-es/v1/database_service_pb";
import {
  extractKeyWithPosition,
  keyWithPosition,
} from "@/views/sql-editor/EditorCommon";
import { useCurrentTabViewStateContext } from "../../context/viewState";
import { CodeViewer } from "../common";
import TriggersTable from "./TriggersTable.vue";

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
  const table = schema?.tables.find((t) => t.name === viewState.value?.table);
  const [name, position] = extractKeyWithPosition(
    viewState.value?.detail?.trigger ?? ""
  );
  const trigger = table?.triggers.find(
    (p, i) => p.name === name && i === position
  );
  return { database, schema, table, trigger };
});

const select = (selected: { trigger: TriggerMetadata; position: number }) => {
  updateViewState({
    detail: {
      trigger: keyWithPosition(selected.trigger.name, selected.position),
    },
  });
};

const deselect = () => {
  updateViewState({
    detail: {},
  });
};
</script>
