<template>
  <div class="flex flex-row items-center gap-x-1">
    <TableIcon v-if="type === 'table'" class="w-4" />
    <ViewIcon v-if="type === 'view'" class="w-4 h-4" />
    <component :is="render" v-if="render" />
    <span v-else>{{ text }}</span>
  </div>
</template>

<script setup lang="ts">
import { isFunction } from "lodash-es";
import { computed } from "vue";
import { TableIcon, ViewIcon } from "@/components/Icon";
import {
  SQLEditorTreeNode as TreeNode,
  SQLEditorTreeFactor as Factor,
} from "@/types";

const props = defineProps<{
  node: TreeNode;
  factors: Factor[];
  keyword: string;
}>();

const target = computed(() => {
  return (props.node as TreeNode<"expandable-text">).meta.target;
});

const type = computed(() => {
  return target.value.type;
});
const text = computed(() => {
  const { text } = target.value;
  return isFunction(text) ? text() : text;
});
const render = computed(() => {
  return target.value.render;
});
</script>
