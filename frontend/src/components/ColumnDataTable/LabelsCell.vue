<template>
  <div class="flex items-center space-x-1">
    <DatabaseLabelsCell :labels="labels" :show-count="2" />
    <button
      v-if="!readonly"
      class="w-5 h-5 p-0.5 hover:bg-gray-300 rounded cursor-pointer"
      @click.prevent="openLabelsDrawer()"
    >
      <heroicons-outline:pencil class="w-4 h-4" />
    </button>
  </div>

  <LabelEditorDrawer
    v-if="state.showLabelsDrawer"
    :show="true"
    :readonly="!!readonly"
    :title="$t('db.labels-for-resource', { resource: `'${column.name}'` })"
    :labels="[labels]"
    @dismiss="state.showLabelsDrawer = false"
    @apply="onLabelsApply($event)"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { reactive } from "vue";
import { DatabaseLabelsCell } from "@/components/v2/Model/DatabaseV1Table/cells";
import { useDBSchemaV1Store } from "@/store";
import type { ComposedDatabase } from "@/types";
import type {
  ColumnMetadata,
  TableMetadata,
} from "@/types/proto/v1/database_service";
import LabelEditorDrawer from "../LabelEditorDrawer.vue";
import { updateColumnConfig } from "./utils";

type LocalState = {
  showLabelsDrawer: boolean;
};

const props = defineProps<{
  database: ComposedDatabase;
  schema: string;
  table: TableMetadata;
  column: ColumnMetadata;
  readonly?: boolean;
}>();

const dbSchemaV1Store = useDBSchemaV1Store();
const state = reactive<LocalState>({
  showLabelsDrawer: false,
});

const columnConfig = computed(() =>
  dbSchemaV1Store.getColumnConfig({
    database: props.database.name,
    schema: props.schema,
    table: props.table.name,
    column: props.column.name,
  })
);

const labels = computed(() => columnConfig.value?.labels ?? {});

const openLabelsDrawer = () => {
  state.showLabelsDrawer = true;
};

const onLabelsApply = async (labelsList: { [key: string]: string }[]) => {
  await updateColumnConfig({
    database: props.database.name,
    schema: props.schema,
    table: props.table.name,
    column: props.column.name,
    columnCatalog: { labels: labelsList[0] },
  });
};
</script>
