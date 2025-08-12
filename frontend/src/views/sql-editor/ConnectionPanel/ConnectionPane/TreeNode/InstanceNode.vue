<template>
  <div class="flex items-center max-w-full overflow-hidden gap-x-1">
    <EnvironmentV1Name
      v-if="!hasEnvironmentContext"
      :environment="environment"
      :link="false"
      class="text-control-light"
    />
    <InstanceV1Name :link="false" :instance="instance" :keyword="keyword" />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { EnvironmentV1Name, InstanceV1Name } from "@/components/v2";
import { useEnvironmentV1Store } from "@/store";
import type { SQLEditorTreeNode as TreeNode } from "@/types";

const props = defineProps<{
  node: TreeNode;
  keyword: string;
}>();

const instance = computed(() => {
  const target = (props.node as TreeNode<"instance">).meta.target;
  // Add $typeName to match InstanceResource type expected by InstanceV1Name
  return {
    ...target,
    $typeName: "bytebase.v1.InstanceResource" as const,
  };
});

const environment = computed(() =>
  useEnvironmentV1Store().getEnvironmentByName(instance.value.environment ?? "")
);

const hasEnvironmentContext = computed(() => {
  return true;
});
</script>
