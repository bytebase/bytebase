<template>
  <div class="flex items-center space-x-1">
    <LabelsCell :labels="labels" :show-count="2" />
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
    :title="$t('db.labels-for-resource', { resource: `'${column}'` })"
    :labels="[labels]"
    @dismiss="state.showLabelsDrawer = false"
    @apply="onLabelsApply($event)"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { reactive } from "vue";
import { LabelsCell } from "@/components/v2/Model/cells";
import { useDatabaseCatalog, getColumnCatalog } from "@/store";
import LabelEditorDrawer from "../LabelEditorDrawer.vue";
import { updateColumnCatalog } from "./utils";

type LocalState = {
  showLabelsDrawer: boolean;
};

const props = defineProps<{
  database: string;
  schema: string;
  table: string;
  column: string;
  readonly?: boolean;
}>();

const state = reactive<LocalState>({
  showLabelsDrawer: false,
});

const databaseCatalog = useDatabaseCatalog(props.database, false);

const columnCatalog = computed(() =>
  getColumnCatalog(
    databaseCatalog.value,
    props.schema,
    props.table,
    props.column
  )
);

const labels = computed(() => columnCatalog.value?.labels ?? {});

const openLabelsDrawer = () => {
  state.showLabelsDrawer = true;
};

const onLabelsApply = async (labelsList: { [key: string]: string }[]) => {
  await updateColumnCatalog({
    database: props.database,
    schema: props.schema,
    table: props.table,
    column: props.column,
    columnCatalog: { labels: labelsList[0] },
  });
};
</script>
