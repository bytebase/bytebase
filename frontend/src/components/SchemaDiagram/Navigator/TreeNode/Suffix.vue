<template>
  <template v-if="node.type === 'schema'">
    <!-- nothing -->
  </template>
  <template v-if="node.type === 'table'">
    <FocusButton
      :table="(node as TreeNode<'table'>).data"
      class="focus-btn"
      @toggle="(on, e) => on && e.stopPropagation()"
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
