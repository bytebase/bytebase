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
import { TreeNodeForColumn } from "../types";

const props = defineProps<{
  node: TreeNodeForColumn;
}>();

const { getColumnSelectionState, updateColumnSelection } =
  useSchemaEditorContext();

const state = computed(() => {
  return getColumnSelectionState(props.node.db, props.node.metadata);
});

const update = (on: boolean) => {
  updateColumnSelection(props.node.db, props.node.metadata, on);
};
</script>
