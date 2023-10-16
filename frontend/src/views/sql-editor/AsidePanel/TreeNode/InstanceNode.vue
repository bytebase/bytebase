<template>
  <div class="flex items-center max-w-full overflow-hidden gap-x-1">
    <InstanceV1EngineIcon :instance="instance" />
    <EnvironmentV1Name
      v-if="!hasEnvironmentContext"
      :environment="instance.environmentEntity"
      :link="false"
      class="text-control-light"
    />
    <HighlightLabelText
      :text="instance.title"
      :keyword="keyword"
      class="flex-1 truncate"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { EnvironmentV1Name, InstanceV1EngineIcon } from "@/components/v2";
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

const instance = computed(
  () => (props.node as TreeNode<"instance">).meta.target
);

const hasEnvironmentContext = computed(() => {
  return props.factors.includes("environment");
});
</script>
