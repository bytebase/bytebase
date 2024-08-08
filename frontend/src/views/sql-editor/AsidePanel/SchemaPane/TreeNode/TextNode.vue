<template>
  <div
    class="flex flex-row items-center gap-x-1"
    :data-mock-type="target.mockType"
  >
    <component :is="render" v-if="render" />

    <template v-else>
      <FolderIcon
        v-if="isFolder"
        class="w-4 h-4 stroke-accent fill-accent/10"
      />
      <span>{{ text }}</span>
    </template>
  </div>
</template>

<script setup lang="ts">
import { isFunction } from "lodash-es";
import { FolderIcon } from "lucide-vue-next";
import { computed } from "vue";
import type { NodeType, TreeNode } from "../common";

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

const isFolder = computed(() => {
  if (!type.value) return false;
  const types: NodeType[] = [
    "table",
    "external-table",
    "view",
    "procedure",
    "function",
    "partition-table",
  ];
  return types.includes(type.value);
});

const text = computed(() => {
  const { text } = target.value;
  return isFunction(text) ? text() : text;
});
const render = computed(() => {
  return target.value.render;
});
</script>
