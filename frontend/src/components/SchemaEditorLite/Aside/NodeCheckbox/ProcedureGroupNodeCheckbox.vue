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
  node: TreeNodeForGroup<"procedure">;
}>();

const { getAllProceduresSelectionState, updateAllProceduresSelection } =
  useSchemaEditorContext();

const state = computed(() => {
  return getAllProceduresSelectionState(
    props.node.db,
    props.node.metadata,
    props.node.metadata.schema.procedures
  );
});

const update = (on: boolean) => {
  updateAllProceduresSelection(
    props.node.db,
    props.node.metadata,
    props.node.metadata.schema.procedures,
    on
  );
};
</script>
