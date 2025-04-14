<template>
  <CommonNode :text="text" :keyword="keyword" :highlight="true" :indent="0">
    <template #icon>
      <ColumnIcon class="w-4 h-4" />
    </template>
  </CommonNode>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { ColumnIcon } from "@/components/Icon";
import type { TreeNode } from "../tree";
import CommonNode from "./CommonNode.vue";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
}>();

const target = computed(
  () => (props.node as TreeNode<"dependency-column">).meta.target
);

const text = computed(() => {
  const { schema, table, column } = target.value.dependencyColumn;
  const parts = [table, column];
  if (schema) parts.unshift(schema);
  return parts.join(".");
});
</script>
