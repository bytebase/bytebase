<template>
  <div
    v-if="metadata?.schema"
    class="px-2 py-2 gap-y-2 h-full overflow-hidden flex flex-col"
  >
    <div
      v-show="!metadata.view"
      class="w-full flex flex-row gap-x-2 justify-between items-center"
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

    <ViewsTable
      v-show="!metadata.view"
      :db="database"
      :database="metadata.database"
      :schema="metadata.schema"
      :views="metadata.schema.views"
      :keyword="state.keyword"
      @click="select"
    />

    <template v-if="metadata.view">
      <CodeViewer
        :db="database"
        :title="metadata.view.name"
        :code="metadata.view.definition"
        @back="deselect"
      >
        <template #title-icon>
          <ViewIcon class="w-4 h-4" />
        </template>
      </CodeViewer>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive } from "vue";
import { ViewIcon } from "@/components/Icon";
import { SearchBox } from "@/components/v2";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
} from "@/store";
import {
  DatabaseMetadata,
  DatabaseMetadataView,
  ViewMetadata,
  SchemaMetadata,
} from "@/types/proto/v1/database_service";
import DatabaseChooser from "@/views/sql-editor/EditorCommon/DatabaseChooser.vue";
import { useEditorPanelContext } from "../../context";
import { SchemaSelectToolbar, CodeViewer } from "../common";
import ViewsTable from "./ViewsTable.vue";

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
  const view = schema?.views.find(
    (v) => v.name === viewState.value?.detail?.view
  );
  return { database, schema, view };
});

const select = (selected: {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  view: ViewMetadata;
}) => {
  updateViewState({
    detail: { view: selected.view.name },
  });
};

const deselect = () => {
  updateViewState({
    detail: {},
  });
};
</script>
