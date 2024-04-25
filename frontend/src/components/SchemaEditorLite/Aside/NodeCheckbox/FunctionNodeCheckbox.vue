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
import type { TreeNodeForFunction } from "../common";

const props = defineProps<{
  node: TreeNodeForFunction;
}>();

const { getFunctionSelectionState, updateFunctionSelection } =
  useSchemaEditorContext();

const state = computed(() => {
  return getFunctionSelectionState(props.node.db, props.node.metadata);
});
const update = (on: boolean) => {
  updateFunctionSelection(props.node.db, props.node.metadata, on);
};
</script>
