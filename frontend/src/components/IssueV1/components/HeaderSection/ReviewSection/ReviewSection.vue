<template>
  <div class="flex items-center gap-x-2">
    <div class="textlabel flex items-center gap-x-1">
      <span>{{ $t("issue.approval-flow.self") }}</span>
      <NTooltip v-if="showApprovalTooltip">
        <div class="max-w-[22rem]">
          {{ $t("issue.approval-flow.tooltip") }}
        </div>
        <template #trigger>
          <heroicons:question-mark-circle />
        </template>
      </NTooltip>
    </div>

    <div class="">
      <div
        v-if="!ready"
        class="flex items-center gap-x-2 text-sm text-control-placeholder"
      >
        <BBSpin class="w-4 h-4" />
        <span>
          {{ $t("custom-approval.issue-review.generating-approval-flow") }}
        </span>
      </div>
      <div v-else-if="error" class="flex items-center gap-x-2">
        <NTooltip>
          <template #trigger>
            <span class="text-error text-sm">{{ $t("common.error") }}</span>
          </template>

          <div class="max-w-[20rem]">{{ error }}</div>
        </NTooltip>
        <NButton
          size="tiny"
          :loading="retrying"
          @click="retryFindingApprovalFlow"
          >{{ $t("common.retry") }}</NButton
        >
      </div>
      <Timeline
        v-else-if="wrappedSteps && wrappedSteps.length > 0"
        :steps="wrappedSteps"
      />
      <div
        v-else
        class="flex items-center text-sm text-control-placeholder gap-x-1"
      >
        {{ $t("custom-approval.approval-flow.skip") }}
        <FeatureBadgeForInstanceLicense
          feature="bb.feature.custom-approval"
          :instance="selectedDatabase.instanceEntity"
        />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";

import { isGrantRequestIssue } from "@/utils";
import {
  extractCoreDatabaseInfoFromDatabaseCreateTask,
  useIssueContext,
  useWrappedReviewStepsV1,
} from "@/components/IssueV1";
import Timeline from "./Timeline.vue";

const { issue, reviewContext, selectedTask } = useIssueContext();
const { ready, error } = reviewContext;
const selectedDatabase = computed(() =>
  extractCoreDatabaseInfoFromDatabaseCreateTask(
    issue.value.projectEntity,
    selectedTask.value
  )
);

const wrappedSteps = useWrappedReviewStepsV1(issue, reviewContext);

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
