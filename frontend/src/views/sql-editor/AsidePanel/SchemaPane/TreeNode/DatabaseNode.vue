<template>
  <div class="flex items-center max-w-full overflow-hidden gap-x-1">
    <InstanceV1EngineIcon :instance="database.instanceEntity" />

    <span class="flex-1 truncate">
      <HighlightLabelText :text="database.databaseName" :keyword="keyword" />
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { InstanceV1EngineIcon } from "@/components/v2";
import type { TreeNode } from "../common";
import HighlightLabelText from "./HighlightLabelText.vue";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
}>();

const database = computed(
  () => (props.node as TreeNode<"database">).meta.target.db
);
</script>
