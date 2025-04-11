<template>
  <template v-if="type === 'instance'">
    <InstanceNode :node="node" :factors="factors" :keyword="keyword" />
  </template>
  <template v-if="type === 'environment'">
    <EnvironmentNode :node="node" :factors="factors" :keyword="keyword" />
  </template>
  <template v-if="type === 'database'">
    <DatabaseNode
      :node="node"
      :factors="factors"
      :keyword="keyword"
      :connected="
        connectedDatabases.has((node as TreeNode<'database'>).meta.target.name)
      "
    />
  </template>
  <template v-if="type === 'label'">
    <LabelNode :node="node" :factors="factors" :keyword="keyword" />
  </template>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import type {
  SQLEditorTreeNode as TreeNode,
  SQLEditorTreeFactor as Factor,
} from "@/types";
import DatabaseNode from "./DatabaseNode.vue";
import EnvironmentNode from "./EnvironmentNode.vue";
import InstanceNode from "./InstanceNode.vue";
import LabelNode from "./LabelNode.vue";

const props = defineProps<{
  node: TreeNode;
  factors: Factor[];
  keyword: string;
  connectedDatabases: Set<string>;
}>();

const type = computed(() => props.node.meta.type);
</script>
