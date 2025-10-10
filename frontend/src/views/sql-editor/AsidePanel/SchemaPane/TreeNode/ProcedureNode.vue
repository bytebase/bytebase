<template>
  <CommonNode
    :text="(procedureMetadata?.signature || procedureMetadata?.name) ?? ''"
    :keyword="keyword"
    :highlight="true"
    :indent="0"
  >
    <template #icon>
      <ProcedureIcon class="w-4 h-4" />
    </template>
  </CommonNode>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { ProcedureIcon } from "@/components/Icon";
import { useDBSchemaV1Store } from "@/store";
import type { TreeNode } from "../tree";
import CommonNode from "./CommonNode.vue";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
}>();

const dbSchema = useDBSchemaV1Store();

const target = computed(
  () => (props.node as TreeNode<"procedure">).meta.target
);

const procedureMetadata = computed(
  () =>
    dbSchema.getSchemaMetadata({
      database: target.value.database,
      schema: target.value.schema,
    }).procedures[target.value.position]
);
</script>
