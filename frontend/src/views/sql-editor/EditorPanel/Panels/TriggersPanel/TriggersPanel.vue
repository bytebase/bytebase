<template>
  <div v-if="metadata?.schema" class="h-full overflow-hidden flex flex-col">
    <div
      v-show="!metadata.trigger"
      class="w-full h-[44px] py-2 px-2 border-b flex flex-row gap-x-2 justify-between items-center"
    >
      <div class="flex items-center justify-start gap-2">
        <DatabaseChooser />
        <SchemaSelectToolbar simple />
      </div>
      <div class="flex items-center justify-end">
        <SearchBox
          v-model:value="state.keyword"
          size="small"
          style="width: 10rem"
        />
      </div>
    </div>
    <TriggersTable
      v-show="!metadata.trigger"
      :db="database"
      :database="metadata.database"
      :schema="metadata.schema"
      :triggers="metadata.schema.triggers"
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
          <ProcedureIcon class="w-4 h-4 text-main" />
        </template>
        <template #content-prefix>
          <div
            class="grid gap-2 text-sm border-b p-2"
            style="grid-template-columns: repeat(3, 1fr)"
          >
            <div class="inline-flex flex-col gap-1">
              <dt class="whitespace-nowrap text-xs font-medium text-gray-500">
                {{ $t("db.trigger.table-name") }}
              </dt>
              <dd
                class="whitespace-pre-wrap break-all hover:underline hover:text-accent cursor-pointer"
                @click="goTable"
              >
                {{ $t(metadata.trigger.tableName) }}
              </dd>
            </div>
            <div class="inline-flex flex-col gap-1">
              <dt class="whitespace-nowrap text-xs font-medium text-gray-500">
                {{ $t("db.trigger.event") }}
              </dt>
              <dd>{{ $t(metadata.trigger.event) }}</dd>
            </div>
            <div class="inline-flex flex-col gap-1">
              <dt class="whitespace-nowrap text-xs font-medium text-gray-500">
                {{ $t("db.trigger.timing") }}
              </dt>
              <dd>{{ $t(metadata.trigger.timing) }}</dd>
            </div>
            <div class="col-span-3 inline-flex flex-col gap-1">
              <dt class="whitespace-nowrap text-xs font-medium text-gray-500">
                SQL mode
              </dt>
              <dd class="whitespace-pre-wrap break-all">
                {{ metadata.trigger.sqlMode }}
              </dd>
            </div>
          </div>
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
import {
  DatabaseMetadata,
  DatabaseMetadataView,
  SchemaMetadata,
  TriggerMetadata,
} from "@/types/proto/v1/database_service";
import {
  extractKeyWithPosition,
  keyWithPosition,
} from "@/views/sql-editor/EditorCommon";
import DatabaseChooser from "@/views/sql-editor/EditorCommon/DatabaseChooser.vue";
import { useEditorPanelContext } from "../../context";
import { SchemaSelectToolbar, CodeViewer } from "../common";
import TriggersTable from "./TriggersTable.vue";

const { database } = useConnectionOfCurrentSQLEditorTab();
const { viewState, updateViewState } = useEditorPanelContext();
const databaseMetadata = computed(() => {
  return useDBSchemaV1Store().getDatabaseMetadata(
    database.value.name,
    DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL
  );
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
    viewState.value?.detail?.trigger ?? ""
  );
  const trigger = schema?.triggers.find(
    (t, i) => t.name === name && i === position
  );
  return { database, schema, trigger };
});

const select = (selected: {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  trigger: TriggerMetadata;
  position: number;
}) => {
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

const goTable = () => {
  const { schema, trigger } = metadata.value;
  if (!schema || !trigger) return;
  updateViewState({
    view: "TABLES",
    schema: schema.name,
    detail: {
      table: trigger.tableName,
    },
  });
};
</script>
