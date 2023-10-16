<template>
  <div class="flex items-center max-w-full overflow-hidden gap-x-1">
    <InstanceV1EngineIcon
      v-if="!hasInstanceContext"
      :instance="database.instanceEntity"
    />

    <EnvironmentV1Name
      v-if="
        !hasEnvironmentContext ||
        database.effectiveEnvironment !== database.instanceEntity.environment
      "
      :environment="database.effectiveEnvironmentEntity"
      :link="false"
      class="text-control-light"
    />

    <DatabaseIcon />

    <span v-if="!hasProjectContext" class="text-control-light">
      {{ database.projectEntity.key }}
    </span>

    <span class="flex-1 truncate">
      <HighlightLabelText :text="database.databaseName" :keyword="keyword" />
      <span v-if="!hasInstanceContext" class="text-control-light">
        ({{ database.instanceEntity.title }})</span
      >
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import DatabaseIcon from "~icons/heroicons-outline/circle-stack";
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

const database = computed(
  () => (props.node as TreeNode<"database">).meta.target
);

const hasInstanceContext = computed(() => {
  return props.factors.includes("instance");
});

const hasEnvironmentContext = computed(() => {
  return props.factors.includes("environment");
});

const hasProjectContext = computed(() => {
  return props.factors.includes("project");
});
</script>
