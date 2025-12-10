<template>
  <div v-if="metadata?.schema" class="h-full overflow-hidden flex flex-col">
    <div
      v-show="!metadata.package"
      class="w-full h-11 py-2 px-2 border-b flex flex-row gap-x-2 justify-end items-center"
    >
      <SearchBox
        v-model:value="state.keyword"
        size="small"
        style="width: 10rem"
      />
    </div>
    <PackagesTable
      v-show="!metadata.package"
      :db="database"
      :database="metadata.database"
      :schema="metadata.schema"
      :packages="metadata.schema.packages"
      :keyword="state.keyword"
      @click="select"
    />

    <template v-if="metadata.package">
      <CodeViewer
        :db="database"
        :title="metadata.package.name"
        :code="metadata.package.definition"
        @back="deselect"
      >
        <template #title-icon>
          <PackageIcon class="w-4 h-4 text-main" />
        </template>
      </CodeViewer>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive } from "vue";
import { PackageIcon } from "@/components/Icon";
import { SearchBox } from "@/components/v2";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
} from "@/store";
import type {
  DatabaseMetadata,
  PackageMetadata,
  SchemaMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import {
  extractKeyWithPosition,
  keyWithPosition,
} from "@/views/sql-editor/EditorCommon";
import { useCurrentTabViewStateContext } from "../../context/viewState";
import { CodeViewer } from "../common";
import PackagesTable from "./PackagesTable.vue";

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
    viewState.value?.detail?.package ?? ""
  );
  const pack = schema?.packages.find(
    (p, i) => p.name === name && i === position
  );
  return { database, schema, package: pack };
});

const select = (selected: {
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  package: PackageMetadata;
  position: number;
}) => {
  updateViewState({
    detail: {
      package: keyWithPosition(selected.package.name, selected.position),
    },
  });
};

const deselect = () => {
  updateViewState({
    detail: {},
  });
};
</script>
