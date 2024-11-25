<template>
  <template v-if="type === 'database'">
    <DatabaseNode :node="node" :keyword="keyword" />
  </template>
  <template v-if="type === 'schema'">
    <SchemaNode :node="node" :keyword="keyword" />
  </template>
  <template v-if="type === 'table'">
    <TableNode :node="node" :keyword="keyword" />
  </template>
  <template v-if="type === 'external-table'">
    <ExternalTableNode :node="node" :keyword="keyword" />
  </template>
  <template v-if="type === 'column'">
    <ColumnNode :node="node" :keyword="keyword" />
  </template>
  <template v-if="type === 'index'">
    <IndexNode :node="node" :keyword="keyword" />
  </template>
  <template v-if="type === 'foreign-key'">
    <ForeignKeyNode :node="node" :keyword="keyword" />
  </template>
  <template v-if="type === 'partition-table'">
    <PartitionTableNode :node="node" :keyword="keyword" />
  </template>
  <template v-if="type === 'view'">
    <ViewNode :node="node" :keyword="keyword" />
  </template>
  <template v-if="type === 'dependent-column'">
    <DependentColumnNode :node="node" :keyword="keyword" />
  </template>
  <template v-if="type === 'procedure'">
    <ProcedureNode :node="node" :keyword="keyword" />
  </template>
  <template v-if="type === 'package'">
    <PackageNode :node="node" :keyword="keyword" />
  </template>
  <template v-if="type === 'function'">
    <FunctionNode :node="node" :keyword="keyword" />
  </template>
  <template v-if="type === 'sequence'">
    <SequenceNode :node="node" :keyword="keyword" />
  </template>
  <template v-if="type === 'trigger'">
    <TriggerNode :node="node" :keyword="keyword" />
  </template>
  <template v-if="type === 'expandable-text'">
    <TextNode :node="node" :keyword="keyword" />
  </template>
  <template v-if="type === 'error'">
    <DummyNode :node="node" :keyword="keyword" />
  </template>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import type { TreeNode } from "../common";
import ColumnNode from "./ColumnNode.vue";
import DatabaseNode from "./DatabaseNode.vue";
import DependentColumnNode from "./DependentColumnNode.vue";
import DummyNode from "./DummyNode.vue";
import ExternalTableNode from "./ExternalTableNode.vue";
import ForeignKeyNode from "./ForeignKeyNode.vue";
import FunctionNode from "./FunctionNode.vue";
import IndexNode from "./IndexNode.vue";
import PackageNode from "./PackageNode.vue";
import PartitionTableNode from "./PartitionTableNode.vue";
import ProcedureNode from "./ProcedureNode.vue";
import SchemaNode from "./SchemaNode.vue";
import SequenceNode from "./SequenceNode.vue";
import TableNode from "./TableNode.vue";
import TextNode from "./TextNode.vue";
import TriggerNode from "./TriggerNode.vue";
import ViewNode from "./ViewNode.vue";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
}>();

const type = computed(() => props.node.meta.type);
</script>
