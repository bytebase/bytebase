<template>
  <CommonNode
    :text="target.trigger.name"
    :keyword="keyword"
    :highlight="true"
    :indent="0"
  >
    <template #icon>
      <TriggerIcon class="w-4 h-4 text-gray-500" />
    </template>
  </CommonNode>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { TriggerIcon } from "@/components/Icon";
import type { TreeNode } from "../common";
import CommonNode from "./CommonNode.vue";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
}>();

const target = computed(() => (props.node as TreeNode<"trigger">).meta.target);
</script>
