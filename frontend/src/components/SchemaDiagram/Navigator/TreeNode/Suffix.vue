<template>
  <template v-if="node.type === 'schema'">
    <!-- nothing -->
  </template>
  <template v-if="node.type === 'table'">
    <FocusButton
      :table="(node as TreeNode<'table'>).data"
      :set-center="true"
      class="focus-btn"
    />
  </template>
</template>

<script lang="ts" setup>
import type { TreeNode } from "../types";
import { FocusButton } from "../../common";

defineProps<{
  node: TreeNode;
}>();
</script>

<style lang="postcss">
.bb-schema-diagram-nav-tree .n-tree-node:hover .focus-btn {
  @apply !visible;
}
</style>
