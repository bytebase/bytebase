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
import type { TreeNodeForProcedure } from "../common";

const props = defineProps<{
  node: TreeNodeForProcedure;
}>();

const { getProcedureSelectionState, updateProcedureSelection } =
  useSchemaEditorContext();

const state = computed(() => {
  return getProcedureSelectionState(props.node.db, props.node.metadata);
});
const update = (on: boolean) => {
  updateProcedureSelection(props.node.db, props.node.metadata, on);
};
</script>
