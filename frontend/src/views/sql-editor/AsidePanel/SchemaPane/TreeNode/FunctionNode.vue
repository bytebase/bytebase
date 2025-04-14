<template>
  <CommonNode
    :text="(functionMetadata?.signature || functionMetadata?.name) ?? ''"
    :keyword="keyword"
    :highlight="true"
    :indent="0"
  >
    <template #icon>
      <FunctionIcon class="w-4 h-4" />
    </template>
  </CommonNode>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { FunctionIcon } from "@/components/Icon";
import { useDBSchemaV1Store } from "@/store";
import type { TreeNode } from "../tree";
import CommonNode from "./CommonNode.vue";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
}>();

const dbSchema = useDBSchemaV1Store();

const target = computed(() => (props.node as TreeNode<"function">).meta.target);

const functionMetadata = computed(() =>
  dbSchema
    .getSchemaMetadata({
      database: target.value.database,
      schema: target.value.schema,
    })
    .functions.find((f) => f.name === target.value.function)
);
</script>
