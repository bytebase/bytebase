<template>
  <CommonNode
    :text="target.foreignKey"
    :keyword="keyword"
    :highlight="true"
    :indent="0"
  >
    <template #icon>
      <ForeignKeyIcon class="w-3.5! h-3.5! text-gray-500" />
    </template>
  </CommonNode>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { ForeignKeyIcon } from "@/components/Icon";
import type { TreeNode } from "../tree";
import CommonNode from "./CommonNode.vue";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
}>();

const target = computed(
  () => (props.node as TreeNode<"foreign-key">).meta.target
);
</script>
