<template>
  <CommonNode
    :text="database.databaseName"
    :keyword="keyword"
    :highlight="true"
  >
    <template #icon>
      <InstanceV1EngineIcon :instance="database.instanceResource" />
    </template>
  </CommonNode>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { InstanceV1EngineIcon } from "@/components/v2";
import type { TreeNode } from "../common";
import CommonNode from "./CommonNode.vue";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
}>();

const database = computed(
  () => (props.node as TreeNode<"database">).meta.target.db
);
</script>
