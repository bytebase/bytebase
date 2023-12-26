<template>
  <NCheckbox
    :checked="state.checked"
    :indeterminate="state.indeterminate"
    size="small"
    @update:checked="update"
    @click.prevent.stop
  />
</template>

<script setup lang="ts">
import { NCheckbox } from "naive-ui";
import { computed } from "vue";
import { useSchemaEditorContext } from "../../context";
import { TreeNodeForTable } from "../types";

const props = defineProps<{
  node: TreeNodeForTable;
}>();

const { getTableSelectionState, updateTableSelection } =
  useSchemaEditorContext();

const state = computed(() => {
  return getTableSelectionState(props.node.db, props.node.metadata);
});
const update = (on: boolean) => {
  updateTableSelection(props.node.db, props.node.metadata, on);
};
</script>
