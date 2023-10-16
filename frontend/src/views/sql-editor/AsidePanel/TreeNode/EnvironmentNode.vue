<template>
  <div class="truncate leading-5">
    <HighlightLabelText :text="environment.title" :keyword="keyword" />
    <ProductionEnvironmentV1Icon :environment="environment" class="ml-1" />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { ProductionEnvironmentV1Icon } from "@/components/v2";
import {
  SQLEditorTreeNode as TreeNode,
  SQLEditorTreeFactor as Factor,
} from "@/types";
import HighlightLabelText from "./HighlightLabelText.vue";

const props = defineProps<{
  node: TreeNode;
  factors: Factor[];
  keyword: string;
}>();

const environment = computed(
  () => (props.node as TreeNode<"environment">).meta.target
);
</script>
