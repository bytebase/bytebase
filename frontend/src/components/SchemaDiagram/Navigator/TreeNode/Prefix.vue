<template>
  <template v-if="node.type === 'schema'">
    <Layers class="w-4 h-4" />
  </template>
  <template v-if="node.type === 'table'">
    <Table2 class="w-4 h-4" />
  </template>
</template>

<script lang="ts" setup>
import { Layers, Table2 } from "lucide-vue-next";
import type { TreeNode } from "../types";

defineProps<{
  node: TreeNode;
}>();
</script>
