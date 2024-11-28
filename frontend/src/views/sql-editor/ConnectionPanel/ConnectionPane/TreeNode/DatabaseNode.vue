<template>
  <div class="flex items-center max-w-full overflow-hidden gap-x-1">
    <InstanceV1EngineIcon
      v-if="!hasInstanceContext"
      :instance="database.instanceResource"
    />

    <EnvironmentV1Name
      v-if="showEnvironment"
      :environment="database.effectiveEnvironmentEntity"
      :link="false"
      class="text-control-light"
    />

    <DatabaseIcon />

    <span class="text-control-light">
      {{ database.projectEntity.key }}
    </span>

    <span class="flex-1 truncate">
      <HighlightLabelText :text="database.databaseName" :keyword="keyword" />
      <span v-if="!hasInstanceContext" class="text-control-light">
        ({{ database.instanceResource.title }})
      </span>
    </span>
    <RequestQueryButton
      v-if="showRequestQueryButton"
      :database-resource="{
        databaseFullName: database.name,
      }"
      :size="'tiny'"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import DatabaseIcon from "~icons/heroicons-outline/circle-stack";
import { EnvironmentV1Name, InstanceV1EngineIcon } from "@/components/v2";
import { hasFeature, useAppFeature } from "@/store";
import type {
  SQLEditorTreeNode as TreeNode,
  SQLEditorTreeFactor as Factor,
} from "@/types";
import { isDatabaseV1Queryable } from "@/utils";
import RequestQueryButton from "../../../EditorCommon/ResultView/RequestQueryButton.vue";
import HighlightLabelText from "./HighlightLabelText.vue";

const props = defineProps<{
  node: TreeNode;
  factors: Factor[];
  keyword: string;
}>();

const disallowRequestQuery = useAppFeature(
  "bb.feature.sql-editor.disallow-request-query"
);
const hideEnvironments = useAppFeature(
  "bb.feature.sql-editor.hide-environments"
);

const database = computed(
  () => (props.node as TreeNode<"database">).meta.target
);

const showRequestQueryButton = computed(() => {
  // Developer self-helped request query is guarded by "Access Control" feature
  return (
    hasFeature("bb.feature.access-control") &&
    !disallowRequestQuery &&
    !isDatabaseV1Queryable(database.value)
  );
});

const hasInstanceContext = computed(() => {
  return props.factors.includes("instance");
});

const hasEnvironmentContext = computed(() => {
  return props.factors.includes("environment");
});

const showEnvironment = computed(() => {
  // Don't show environment tag anyway if disabled via appFeature
  if (hideEnvironments.value) {
    return false;
  }
  // If we don't have "environment" factor in the custom tree structure
  // we should indicate the database's environment
  if (!hasEnvironmentContext.value) {
    return true;
  }
  // If we have "environment" factor in the custom tree structure
  // only show the environment tag when a database's effectiveEnvironment is
  // not equal to it's physical instance's environment
  return (
    database.value.effectiveEnvironment !==
    database.value.instanceResource.environment
  );
});
</script>
