<template>
  <CommonNode
    :text="target.index"
    :keyword="keyword"
    :highlight="true"
    :indent="0"
  >
    <template #icon>
      <PrimaryKeyIcon v-if="isPrimaryKey" class="w-4 h-4" />
      <IndexIcon v-else class="w-4! h-4! text-accent/80" />
    </template>
  </CommonNode>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { IndexIcon, PrimaryKeyIcon } from "@/components/Icon";
import { useDBSchemaV1Store } from "@/store";
import type { TreeNode } from "../tree";
import CommonNode from "./CommonNode.vue";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
}>();

const dbSchema = useDBSchemaV1Store();

const target = computed(() => (props.node as TreeNode<"index">).meta.target);

const indexMetadata = computed(() => {
  const { database, schema, table, index } = target.value;
  const tableMetadata = dbSchema.getTableMetadata({
    database,
    schema,
    table,
  });
  return tableMetadata.indexes.find((i) => i.name === index);
});

const isPrimaryKey = computed(() => {
  return indexMetadata.value?.primary;
});
</script>
