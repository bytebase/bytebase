<template>
  <CommonNode
    :text="target.procedure.name"
    :keyword="keyword"
    :highlight="true"
    :indent="0"
  >
    <template #icon>
      <ProcedureIcon class="w-4 h-4" />
    </template>
  </CommonNode>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { ProcedureIcon } from "@/components/Icon";
import type { TreeNode } from "../common";
import CommonNode from "./CommonNode.vue";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
}>();

const target = computed(
  () => (props.node as TreeNode<"procedure">).meta.target
);
</script>
