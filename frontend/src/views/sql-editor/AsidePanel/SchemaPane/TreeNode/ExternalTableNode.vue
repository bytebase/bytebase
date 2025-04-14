<template>
  <CommonNode :text="target.externalTable" :keyword="keyword" :highlight="true">
    <template #icon>
      <ExternalTableIcon class="w-4 h-4" />
    </template>
  </CommonNode>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { ExternalTableIcon } from "@/components/Icon";
import type { TreeNode } from "../tree";
import CommonNode from "./CommonNode.vue";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
}>();

const target = computed(
  () => (props.node as TreeNode<"external-table">).meta.target
);
</script>
