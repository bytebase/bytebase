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
import { useDatabaseV1Store } from "@/store";
import type { TreeNode } from "../tree";
import CommonNode from "./CommonNode.vue";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
}>();

const databaseStore = useDatabaseV1Store();

const database = computed(() =>
  databaseStore.getDatabaseByName(
    (props.node as TreeNode<"database">).meta.target.database
  )
);
</script>
