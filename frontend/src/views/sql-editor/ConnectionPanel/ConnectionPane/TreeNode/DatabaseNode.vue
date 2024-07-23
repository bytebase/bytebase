<template>
  <div class="flex items-center max-w-full overflow-hidden gap-x-1">
    <InstanceV1EngineIcon
      v-if="!hasInstanceContext"
      :instance="database.instanceResource"
    />

    <EnvironmentV1Name
      v-if="
        !hasEnvironmentContext ||
        database.effectiveEnvironment !== database.instanceResource.environment
      "
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
      v-if="!disallowRequestQuery && !canQuery"
      :database="database"
      :size="'tiny'"
      :panel-placement="'left'"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import DatabaseIcon from "~icons/heroicons-outline/circle-stack";
import { EnvironmentV1Name, InstanceV1EngineIcon } from "@/components/v2";
import { useAppFeature, useCurrentUserV1 } from "@/store";
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

const me = useCurrentUserV1();
const disallowRequestQuery = useAppFeature(
  "bb.feature.sql-editor.disallow-request-query"
);

const database = computed(
  () => (props.node as TreeNode<"database">).meta.target
);

const canQuery = computed(() =>
  isDatabaseV1Queryable(database.value, me.value)
);

const hasInstanceContext = computed(() => {
  return props.factors.includes("instance");
});

const hasEnvironmentContext = computed(() => {
  return props.factors.includes("environment");
});
</script>
