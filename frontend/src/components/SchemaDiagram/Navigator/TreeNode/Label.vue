<template>
  <!-- eslint-disable vue/no-v-html -->
  <div class="flex items-center gap-x-1 max-w-full">
    <span
      class="truncate"
      :data-schema-editor-nav-tree-node-id="id"
      v-html="html"
    />
    <span v-if="isTypedNode(node, 'schema')" class="shrink-0 text-gray-500">
      ({{ node.data.tables.length }})
    </span>
  </div>
</template>

<script lang="ts" setup>
import { escape } from "lodash-es";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { getHighlightHTMLByKeyWords } from "@/utils";
import type { TreeNode } from "../types";
import { isTypedNode } from "../utils";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
}>();

const { t } = useI18n();

// render an unique id for every node
// for auto scroll to the node when tab switches
const id = computed(() => {
  const { node } = props;
  return `tree-node-label-${node.type}-${node.id}`;
});

const text = computed(() => {
  const { node } = props;
  return node.data.name || t("db.schema.default");
});

const html = computed(() => {
  return getHighlightHTMLByKeyWords(escape(text.value), escape(props.keyword));
});
</script>
