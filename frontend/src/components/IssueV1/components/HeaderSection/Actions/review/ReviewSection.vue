<template>
  <h2 class="textlabel flex items-start col-span-1 col-start-1 pt-1">
    <div class="flex items-center gap-x-1">
      <span>{{ $t("issue.approval-flow.self") }}</span>
      <NTooltip v-if="showApprovalTooltip">
        <div class="max-w-[24rem]">
          {{ $t("issue.approval-flow.tooltip") }}
        </div>
        <template #trigger>
          <heroicons-outline:question-mark-circle />
        </template>
      </NTooltip>
    </div>
  </h2>
  <div class="col-span-2">
    <div
      v-if="!ready"
      class="flex items-center gap-x-2 mt-1 text-sm text-control-placeholder"
    >
      <BBSpin class="w-4 h-4" />
      <span>
        {{ $t("custom-approval.issue-review.generating-approval-flow") }}
      </span>
    </div>
    <div v-else-if="error" class="flex items-center gap-x-2 mt-0.5">
      <NTooltip>
        <template #trigger>
          <span class="text-error text-sm">{{ $t("common.error") }}</span>
        </template>

        <div class="max-w-[20rem]">
          {{ error }}
        </div>
      </NTooltip>
      <NButton
        size="tiny"
        :loading="retrying"
        @click="retryFindingApprovalFlow"
        >{{ $t("common.retry") }}</NButton
      >
    </div>
    <ReviewPanel v-else-if="steps.length > 0" />
    <div
      v-else
      class="flex items-center mt-1 text-sm text-control-placeholder gap-x-1"
    >
      {{ $t("custom-approval.approval-flow.skip") }}
      <FeatureBadgeForInstanceLicense
        feature="bb.feature.custom-approval"
        :instance="selectedDatabase?.instanceEntity"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";

import ReviewPanel from "./ReviewPanel.vue";
import { isGrantRequestIssue } from "@/utils";
import {
  extractCoreDatabaseInfoFromDatabaseCreateTask,
  useIssueContext,
} from "@/components/IssueV1";

const { issue, reviewContext, selectedTask } = useIssueContext();
const { ready, error, flow } = reviewContext;
const selectedDatabase = computed(() =>
  extractCoreDatabaseInfoFromDatabaseCreateTask(
    issue.value.projectEntity,
    selectedTask.value
  )
);

const steps = computed(() => flow.value.template.flow?.steps ?? []);

const retrying = ref(false);
const retryFindingApprovalFlow = async () => {
  retrying.value = true;
  try {
    // await store.regenerateReview(issue.value);
    // TODO
    await new Promise((r) => setTimeout(r, 500));
  } finally {
    retrying.value = false;
  }
};

const showApprovalTooltip = computed(() => {
  // Don't show the tooltip if the issue type is grant request.
  if (isGrantRequestIssue(issue.value)) {
    return false;
  }
  return true;
});
</script>
