<template>
  <template v-if="node.type === 'schema'">
    <heroicons-outline:rectangle-stack class="w-4 h-4" />
  </template>
  <template v-if="node.type === 'table'">
    <heroicons-outline:table-cells class="w-4 h-4" />
  </template>
</template>

<script lang="ts" setup>
import type { TreeNode } from "../types";

defineProps<{
  node: TreeNode;
}>();
</script>
