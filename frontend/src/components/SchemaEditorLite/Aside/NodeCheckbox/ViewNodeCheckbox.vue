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
import type { TreeNodeForView } from "../common";

const props = defineProps<{
  node: TreeNodeForView;
}>();

const { getViewSelectionState, updateViewSelection } = useSchemaEditorContext();

const state = computed(() => {
  return getViewSelectionState(props.node.db, props.node.metadata);
});
const update = (on: boolean) => {
  updateViewSelection(props.node.db, props.node.metadata, on);
};
</script>
