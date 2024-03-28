<template>
  <div class="flex items-center max-w-full overflow-hidden gap-x-1">
    <ViewIcon class="w-4 h-4" />
    <HighlightLabelText
      :text="target.view.name"
      :keyword="keyword"
      class="flex-1 truncate"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { ViewIcon } from "@/components/Icon";
import type { TreeNode } from "../common";
import HighlightLabelText from "./HighlightLabelText.vue";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
}>();

const target = computed(() => (props.node as TreeNode<"view">).meta.target);
</script>
