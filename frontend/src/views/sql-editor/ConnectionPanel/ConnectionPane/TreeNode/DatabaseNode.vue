<template>
  <div class="flex items-center max-w-full overflow-hidden gap-x-1">
    <LinkIcon v-if="connected" class="w-4 textinfolabel" />
    <NCheckbox
      v-else-if="!disallowBatchQuery && canQuery"
      :checked="checked"
      :disabled="tabStore.currentTab?.connection.database === database.name"
      @click.stop.prevent=""
      @update:checked="$emit('update:checked', $event)"
    />

    <RichDatabaseName
      :database="database"
      :show-instance="!hasInstanceContext"
      :show-engine-icon="!hasInstanceContext"
      :show-environment="showEnvironment"
      :show-arrow="true"
      :keyword="keyword"
    />

    <span v-if="connected" class="truncate textinfolabel">
      ({{ $t("sql-editor.connected") }})
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
import { LinkIcon } from "lucide-vue-next";
import { NCheckbox } from "naive-ui";
import { computed } from "vue";
import { RichDatabaseName } from "@/components/v2";
import { useAppFeature, useSQLEditorTabStore } from "@/store";
import type {
  SQLEditorTreeNode as TreeNode,
  SQLEditorTreeFactor as Factor,
} from "@/types";
import { isDatabaseV1Queryable } from "@/utils";
import RequestQueryButton from "../../../EditorCommon/ResultView/RequestQueryButton.vue";

const props = defineProps<{
  node: TreeNode;
  factors: Factor[];
  keyword: string;
  connected?: boolean;
  checked?: boolean;
}>();

defineEmits<{
  (event: "update:checked", checked: boolean): void;
}>();

const tabStore = useSQLEditorTabStore();

const disallowBatchQuery = useAppFeature(
  "bb.feature.sql-editor.disallow-batch-query"
);

const disallowRequestQuery = useAppFeature(
  "bb.feature.sql-editor.disallow-request-query"
);
const hideEnvironments = useAppFeature(
  "bb.feature.sql-editor.hide-environments"
);

const database = computed(
  () => (props.node as TreeNode<"database">).meta.target
);

const canQuery = computed(() => isDatabaseV1Queryable(database.value));

const showRequestQueryButton = computed(() => {
  return (
    !disallowRequestQuery.value &&
    !canQuery.value
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
