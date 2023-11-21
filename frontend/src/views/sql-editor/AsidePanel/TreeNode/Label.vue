<template>
  <template v-if="type === 'project'">
    <ProjectNode :node="node" :factors="factors" :keyword="keyword" />
  </template>
  <template v-if="type === 'instance'">
    <InstanceNode :node="node" :factors="factors" :keyword="keyword" />
  </template>
  <template v-if="type === 'environment'">
    <EnvironmentNode :node="node" :factors="factors" :keyword="keyword" />
  </template>
  <template v-if="type === 'database'">
    <DatabaseNode :node="node" :factors="factors" :keyword="keyword" />
  </template>
  <template v-if="type === 'schema'">
    <SchemaNode :node="node" :factors="factors" :keyword="keyword" />
  </template>
  <template v-if="type === 'table'">
    <TableNode :node="node" :factors="factors" :keyword="keyword" />
  </template>
  <template v-if="type === 'view'">
    <ViewNode :node="node" :factors="factors" :keyword="keyword" />
  </template>
  <template v-if="type === 'expandable-text'">
    <TextNode :node="node" :factors="factors" :keyword="keyword" />
  </template>
  <template v-if="type === 'label'">
    <LabelNode :node="node" :factors="factors" :keyword="keyword" />
  </template>
  <template v-if="type === 'dummy'">
    <DummyNode :node="node" :factors="factors" :keyword="keyword" />
  </template>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import {
  SQLEditorTreeNode as TreeNode,
  SQLEditorTreeFactor as Factor,
} from "@/types";
import DatabaseNode from "./DatabaseNode.vue";
import DummyNode from "./DummyNode.vue";
import EnvironmentNode from "./EnvironmentNode.vue";
import InstanceNode from "./InstanceNode.vue";
import LabelNode from "./LabelNode.vue";
import ProjectNode from "./ProjectNode.vue";
import SchemaNode from "./SchemaNode.vue";
import TableNode from "./TableNode.vue";
import TextNode from "./TextNode.vue";
import ViewNode from "./ViewNode.vue";

const props = defineProps<{
  node: TreeNode;
  factors: Factor[];
  keyword: string;
}>();

const type = computed(() => props.node.meta.type);
</script>
