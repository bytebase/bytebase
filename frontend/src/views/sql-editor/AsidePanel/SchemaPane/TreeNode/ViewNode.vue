<template>
  <CommonNode
    :text="target.view.name"
    :keyword="keyword"
    :highlight="true"
    :indent="1"
  >
    <template #icon>
      <ViewIcon class="w-4 h-4" />
    </template>
  </CommonNode>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { ViewIcon } from "@/components/Icon";
import type { TreeNode } from "../common";
import CommonNode from "./CommonNode.vue";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
}>();

const target = computed(() => (props.node as TreeNode<"view">).meta.target);
</script>
