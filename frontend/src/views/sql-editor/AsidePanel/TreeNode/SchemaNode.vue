<template>
  <div class="flex items-center max-w-full overflow-hidden gap-x-1">
    <SchemaIcon />
    <HighlightLabelText
      :text="target.schema.name"
      :keyword="keyword"
      class="flex-1 truncate"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import SchemaIcon from "~icons/heroicons-outline/view-columns";
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

const target = computed(() => (props.node as TreeNode<"schema">).meta.target);
</script>
