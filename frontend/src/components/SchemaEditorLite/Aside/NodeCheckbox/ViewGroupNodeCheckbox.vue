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
  node: TreeNodeForGroup<"view">;
}>();

const { getAllViewsSelectionState, updateAllViewsSelection } =
  useSchemaEditorContext();

const state = computed(() => {
  return getAllViewsSelectionState(
    props.node.db,
    props.node.metadata,
    props.node.metadata.schema.views
  );
});

const update = (on: boolean) => {
  updateAllViewsSelection(
    props.node.db,
    props.node.metadata,
    props.node.metadata.schema.views,
    on
  );
};
</script>
