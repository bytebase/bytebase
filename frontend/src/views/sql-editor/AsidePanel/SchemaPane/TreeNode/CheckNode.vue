<template>
  <CommonNode
    :text="target.check"
    :keyword="keyword"
    :highlight="true"
    :indent="0"
  >
    <template #icon>
      <CheckIcon class="w-3.5! h-3.5! text-gray-500" />
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
          {{ checkMetadata?.expression }}
        </NTag>
      </div>
    </template>
  </CommonNode>
</template>

<script setup lang="ts">
import { NTag } from "naive-ui";
import { computed } from "vue";
import { CheckIcon } from "@/components/Icon";
import { useDBSchemaV1Store } from "@/store";
import type { TreeNode } from "../tree";
import CommonNode from "./CommonNode.vue";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
}>();

const dbSchema = useDBSchemaV1Store();
const target = computed(() => (props.node as TreeNode<"check">).meta.target);

const tableMetadata = computed(() => {
  if ("table" in target.value) {
    const { database, schema, table } = target.value;
    return dbSchema.getTableMetadata({ database, schema, table });
  }
  return undefined;
});

const checkMetadata = computed(() => {
  const { check } = target.value;

  if ("table" in target.value) {
    return tableMetadata.value?.checkConstraints.find((c) => c.name === check);
  }
  return undefined;
});
</script>
