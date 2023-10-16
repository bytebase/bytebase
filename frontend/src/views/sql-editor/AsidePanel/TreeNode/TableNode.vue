<template>
  <div class="flex items-center max-w-full overflow-hidden gap-x-1">
    <TableIcon />
    <HighlightLabelText
      :text="target.table.name"
      :keyword="keyword"
      class="flex-1 truncate"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import TableIcon from "~icons/heroicons-outline/table-cells";
import {
  SQLEditorTreeNode as TreeNode,
  SQLEditorTreeFactor as Factor,
} from "@/types";
import HighlightLabelText from "./HighlightLabelText.vue";

const props = defineProps<{
  node: TreeNode;
  factors: Factor[];
  keyword: string;
}>();

const target = computed(() => (props.node as TreeNode<"table">).meta.target);
</script>
