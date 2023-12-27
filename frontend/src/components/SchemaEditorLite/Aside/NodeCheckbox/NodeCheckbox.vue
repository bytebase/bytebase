<template>
  <SchemaNodeCheckbox v-if="node.type === 'schema'" :node="node" />
  <TableNodeCheckbox v-if="node.type === 'table'" :node="node" />
  <ColumnNodeCheckbox v-if="node.type === 'column'" :node="node" />
</template>

<script setup lang="ts">
import { TreeNode } from "../types";
import ColumnNodeCheckbox from "./ColumnNodeCheckbox.vue";
import SchemaNodeCheckbox from "./SchemaNodeCheckbox.vue";
import TableNodeCheckbox from "./TableNodeCheckbox.vue";

defineProps<{
  node: TreeNode;
}>();
</script>
