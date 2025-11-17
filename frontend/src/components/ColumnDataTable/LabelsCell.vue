<template>
  <div class="flex items-center gap-x-1">
    <LabelsCell :labels="labels" :show-count="2" />
    <MiniActionButton v-if="!readonly" @click.prevent="openLabelsDrawer()">
      <PencilIcon class="w-3 h-3" />
    </MiniActionButton>
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
import { PencilIcon } from "lucide-vue-next";
import { computed, reactive } from "vue";
import { MiniActionButton } from "@/components/v2";
import { LabelsCell } from "@/components/v2/Model/cells";
import { getColumnCatalog, useDatabaseCatalog } from "@/store";
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
