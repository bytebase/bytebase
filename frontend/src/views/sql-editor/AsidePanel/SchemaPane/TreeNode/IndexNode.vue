<template>
  <CommonNode
    :text="target.index.name"
    :keyword="keyword"
    :highlight="true"
    :indent="0"
  >
    <template #icon>
      <PrimaryKeyIcon v-if="isPrimaryKey" class="w-4 h-4" />
      <IndexIcon v-else class="!w-4 !h-4 text-accent/80" />
    </template>
  </CommonNode>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { IndexIcon, PrimaryKeyIcon } from "@/components/Icon";
import type { TreeNode } from "../common";
import CommonNode from "./CommonNode.vue";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
}>();

const target = computed(() => (props.node as TreeNode<"index">).meta.target);
const isPrimaryKey = computed(() => {
  return target.value.index.primary;
});
</script>
