<template>
  <CommonNode :text="text" :highlight="false" :data-mock-type="type">
    <template v-if="render" #default>
      <component :is="render" />
    </template>

    <template #icon>
      <FolderIcon class="w-4 h-4 stroke-accent fill-accent/10" />
    </template>
  </CommonNode>
</template>

<script setup lang="ts">
import { isFunction } from "lodash-es";
import { FolderIcon } from "lucide-vue-next";
import { computed } from "vue";
import type { TreeNode } from "../tree";
import CommonNode from "./CommonNode.vue";

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
