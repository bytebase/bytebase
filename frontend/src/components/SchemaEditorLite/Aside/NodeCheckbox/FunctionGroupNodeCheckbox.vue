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
import type { TreeNodeForGroup } from "../common";

const props = defineProps<{
  node: TreeNodeForGroup<"function">;
}>();

const { getAllFunctionsSelectionState, updateAllFunctionsSelection } =
  useSchemaEditorContext();

const state = computed(() => {
  return getAllFunctionsSelectionState(
    props.node.db,
    props.node.metadata,
    props.node.metadata.schema.functions
  );
});

const update = (on: boolean) => {
  updateAllFunctionsSelection(
    props.node.db,
    props.node.metadata,
    props.node.metadata.schema.functions,
    on
  );
};
</script>
