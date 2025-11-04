<template>
  <CommonNode
    :text="target.column"
    :keyword="keyword"
    :highlight="true"
    :indent="0"
  >
    <template #icon>
      <PrimaryKeyIcon v-if="isPrimaryKey" class="w-4 h-4" />
      <IndexIcon v-else-if="isIndex" class="w-4! h-4! text-accent/80" />
      <ColumnIcon v-else class="w-4 h-4" />
    </template>
    <template #suffix>
      <div
        class="flex items-center justify-end gap-1 overflow-hidden whitespace-nowrap shrink opacity-80 font-normal!"
      >
        <NTag
          size="small"
          class="text-size-adjust-none"
          style="--n-height: 16px; --n-padding: 0 3px; --n-font-size: 10px"
        >
          {{ columnMetadata?.type }}
        </NTag>
      </div>
    </template>
  </CommonNode>
</template>

<script setup lang="ts">
import { NTag } from "naive-ui";
import { computed } from "vue";
import { ColumnIcon, IndexIcon, PrimaryKeyIcon } from "@/components/Icon";
import { useDBSchemaV1Store } from "@/store";
import type { TreeNode } from "../tree";
import CommonNode from "./CommonNode.vue";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
}>();

const dbSchema = useDBSchemaV1Store();
const target = computed(() => (props.node as TreeNode<"column">).meta.target);

const tableMetadata = computed(() => {
  if ("table" in target.value) {
    const { database, schema, table } = target.value;
    return dbSchema.getTableMetadata({ database, schema, table });
  }
  return undefined;
});

const columnMetadata = computed(() => {
  const { database, schema, column } = target.value;
  const schemaMetadata = dbSchema.getSchemaMetadata({ database, schema });

  if ("table" in target.value) {
    return tableMetadata.value?.columns.find((c) => c.name === column);
  } else if ("externalTable" in target.value) {
    const { externalTable } = target.value;
    return schemaMetadata.externalTables
      .find((t) => t.name === externalTable)
      ?.columns.find((c) => c.name === column);
  } else if ("view" in target.value) {
    const { view } = target.value;
    return schemaMetadata.views
      .find((v) => v.name === view)
      ?.columns.find((c) => c.name === column);
  }
  return undefined;
});

const isPrimaryKey = computed(() => {
  if ("table" in target.value) {
    const { column } = target.value;
    const pk = tableMetadata.value?.indexes.find((idx) => idx.primary);
    if (!pk) return false;
    return pk.expressions.includes(column);
  }

  return false;
});

const isIndex = computed(() => {
  if (isPrimaryKey.value) return false;

  if ("table" in target.value) {
    const { column } = target.value;
    return tableMetadata.value?.indexes.some((idx) =>
      idx.expressions.includes(column)
    );
  }

  return false;
});
</script>
