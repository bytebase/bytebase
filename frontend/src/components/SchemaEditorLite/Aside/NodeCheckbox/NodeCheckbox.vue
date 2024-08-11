<template>
  <TableGroupNodeCheckbox
    v-if="node.type === 'group' && node.group === 'table'"
    :node="node as TreeNodeForGroup<'table'>"
  />
  <ViewGroupNodeCheckbox
    v-if="node.type === 'group' && node.group === 'view'"
    :node="node as TreeNodeForGroup<'view'>"
  />
  <ProcedureGroupNodeCheckbox
    v-if="node.type === 'group' && node.group === 'procedure'"
    :node="node as TreeNodeForGroup<'procedure'>"
  />
  <FunctionGroupNodeCheckbox
    v-if="node.type === 'group' && node.group === 'function'"
    :node="node as TreeNodeForGroup<'function'>"
  />
  <TableNodeCheckbox v-if="node.type === 'table'" :node="node" />
  <ColumnNodeCheckbox v-if="node.type === 'column'" :node="node" />
  <ViewNodeCheckbox v-if="node.type === 'view'" :node="node" />
  <ProcedureNodeCheckbox v-if="node.type === 'procedure'" :node="node" />
  <FunctionNodeCheckbox v-if="node.type === 'function'" :node="node" />
</template>

<script setup lang="ts">
import type { TreeNode, TreeNodeForGroup } from "../common";
import ColumnNodeCheckbox from "./ColumnNodeCheckbox.vue";
import FunctionGroupNodeCheckbox from "./FunctionGroupNodeCheckbox.vue";
import FunctionNodeCheckbox from "./FunctionNodeCheckbox.vue";
import ProcedureGroupNodeCheckbox from "./ProcedureGroupNodeCheckbox.vue";
import ProcedureNodeCheckbox from "./ProcedureNodeCheckbox.vue";
import TableGroupNodeCheckbox from "./TableGroupNodeCheckbox.vue";
import TableNodeCheckbox from "./TableNodeCheckbox.vue";
import ViewGroupNodeCheckbox from "./ViewGroupNodeCheckbox.vue";
import ViewNodeCheckbox from "./ViewNodeCheckbox.vue";

defineProps<{
  node: TreeNode;
}>();
</script>
