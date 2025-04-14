<template>
  <CommonNode :text="target.schema" :keyword="keyword" :highlight="true">
    <template #icon>
      <SchemaIcon class="w-4 h-4" />
    </template>
    <template #fallback-text>
      {{ $t("db.schema.default") }}
    </template>
  </CommonNode>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { SchemaIcon } from "@/components/Icon";
import type { TreeNode } from "../tree";
import CommonNode from "./CommonNode.vue";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
}>();

const target = computed(() => (props.node as TreeNode<"schema">).meta.target);
</script>
