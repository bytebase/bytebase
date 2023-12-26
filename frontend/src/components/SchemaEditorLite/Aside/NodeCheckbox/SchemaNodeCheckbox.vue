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
import { TreeNodeForSchema } from "../types";

const props = defineProps<{
  node: TreeNodeForSchema;
}>();

const { getAllTablesSelectionState, updateAllTablesSelection } =
  useSchemaEditorContext();

const state = computed(() => {
  return getAllTablesSelectionState(
    props.node.db,
    props.node.metadata,
    props.node.metadata.schema.tables
  );
});

const update = (on: boolean) => {
  updateAllTablesSelection(
    props.node.db,
    props.node.metadata,
    props.node.metadata.schema.tables,
    on
  );
};
</script>
