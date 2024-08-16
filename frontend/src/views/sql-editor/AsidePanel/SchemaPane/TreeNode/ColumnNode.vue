<template>
  <CommonNode
    :text="target.column.name"
    :keyword="keyword"
    :highlight="true"
    :indent="0"
  >
    <template #icon>
      <PrimaryKeyIcon v-if="isPrimaryKey" class="w-4 h-4" />
      <IndexIcon v-else-if="isIndex" class="!w-4 !h-4 text-accent/80" />
      <ColumnIcon v-else class="w-4 h-4" />
    </template>
  </CommonNode>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { ColumnIcon, IndexIcon, PrimaryKeyIcon } from "@/components/Icon";
import type { TreeNode } from "../common";
import CommonNode from "./CommonNode.vue";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
}>();

const target = computed(() => (props.node as TreeNode<"column">).meta.target);

const isPrimaryKey = computed(() => {
  if ("table" in target.value) {
    const { table, column } = target.value;
    const pk = table.indexes.find((idx) => idx.primary);
    if (!pk) return false;
    return pk.expressions.includes(column.name);
  }

  return false;
});
const isIndex = computed(() => {
  if (isPrimaryKey.value) return false;

  if ("table" in target.value) {
    const { table, column } = target.value;
    return table.indexes.some((idx) => idx.expressions.includes(column.name));
  }

  return false;
});
</script>
