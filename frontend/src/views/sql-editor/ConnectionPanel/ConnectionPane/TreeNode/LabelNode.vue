<template>
  <div class="truncate">
    <span>{{ label.key }}</span>
    <span class="mr-0.5">:</span>
    <HighlightLabelText
      v-if="label.value"
      :text="label.value"
      :keyword="keyword"
    />
    <span v-else class="text-control-placeholder">
      {{ $t("label.empty-label-value") }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { HighlightLabelText } from "@/components/v2";
import type { SQLEditorTreeNode as TreeNode } from "@/types";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
}>();

const label = computed(() => (props.node as TreeNode<"label">).meta.target);
</script>
