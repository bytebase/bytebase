<template>
  <!-- eslint-disable vue/no-v-html -->
  <span
    class="truncate"
    :data-schema-editor-nav-tree-node-id="id"
    v-html="html"
  />
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { escape } from "lodash-es";

import type { TreeNode } from "../types";
import { getHighlightHTMLByKeyWords } from "@/utils";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
}>();

// render an unique id for every node
// for auto scroll to the node when tab switches
const id = computed(() => {
  const { node } = props;
  return `tree-node-label-${node.type}-${node.id}`;
});

const text = computed(() => {
  const { node } = props;
  return node.data.name;
});

const html = computed(() => {
  return getHighlightHTMLByKeyWords(escape(text.value), escape(props.keyword));
});
</script>
