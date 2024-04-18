<template>
  <div
    class="flex flex-row items-center gap-x-1"
    :data-mock-type="target.mockType"
  >
    <TableIcon
      v-if="type === 'table' || type === 'external-table'"
      class="w-4"
    />
    <ViewIcon v-if="type === 'view'" class="w-4 h-4" />
    <ProcedureIcon v-if="type === 'procedure'" class="w-4 h-4" />
    <FunctionIcon v-if="type === 'function'" class="w-4 h-4" />
    <TablePartitionIcon v-if="type === 'partition-table'" class="w-4 h-4" />

    <component :is="render" v-if="render" />

    <span v-else>{{ text }}</span>
  </div>
</template>

<script setup lang="ts">
import { isFunction } from "lodash-es";
import { computed } from "vue";
import {
  FunctionIcon,
  ProcedureIcon,
  TableIcon,
  TablePartitionIcon,
  ViewIcon,
} from "@/components/Icon";
import type { TreeNode } from "../common";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
}>();

const target = computed(() => {
  return (props.node as TreeNode<"expandable-text">).meta.target;
});

const type = computed(() => {
  return target.value.mockType;
});
const text = computed(() => {
  const { text } = target.value;
  return isFunction(text) ? text() : text;
});
const render = computed(() => {
  return target.value.render;
});
</script>
