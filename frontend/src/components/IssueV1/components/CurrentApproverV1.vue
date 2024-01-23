<template>
  <div class="flex flex-row items-center gap-x-2 overflow-hidden">
    <template v-if="issue.status !== IssueStatus.OPEN || done || !ready">
      <span>-</span>
    </template>
    <template v-else-if="currentStep?.status === 'REJECTED'">
      <NTooltip :disabled="!currentStep.approver">
        <template #trigger>
          <div class="flex flex-row items-center gap-x-2">
            <div
              class="w-6 h-6 rounded-full flex items-center justify-center text-sm shrink-0 bg-warning"
            >
              <heroicons:pause-solid class="w-5 h-5 text-white" />
            </div>
            <span class="text-warning">
              {{ $t("custom-approval.approval-flow.issue-review.sent-back") }}
            </span>
          </div>
        </template>
        <template #default>
          <i18n-t
            keypath="custom-approval.approval-flow.issue-review.review-sent-back-by"
          >
            <template #user>{{ currentStep.approver?.title }}</template>
          </i18n-t>
        </template>
      </NTooltip>
    </template>
    <template v-else-if="currentApprover">
      <BBAvatar :size="'SMALL'" :username="currentApprover.title" />
      <span class="truncate">
        {{ currentApprover.title }}
      </span>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { toRef } from "vue";
import {
  extractReviewContext,
  useWrappedReviewStepsV1,
} from "@/components/IssueV1";
import { useCurrentUserV1 } from "@/store";
import { ComposedIssue } from "@/types";
import { IssueStatus } from "@/types/proto/v1/issue_service";

const props = defineProps<{
  issue: ComposedIssue;
}>();

const context = extractReviewContext(toRef(props, "issue"));
const { ready, done } = context;
const me = useCurrentUserV1();

const wrappedSteps = useWrappedReviewStepsV1(toRef(props, "issue"), context);

const currentStep = computed(() => {
  return wrappedSteps.value?.find(
    (step) => step.status === "CURRENT" || step.status === "REJECTED"
  );
});

const currentApprover = computed(() => {
  if (!currentStep.value) return undefined;
  const myself = currentStep.value.candidates.find(
    (user) => user.name === me.value.name
  );
  // Show currentUser if currentUser is one of the validate approver candidates.
  if (myself) return myself;
  // Show the first approver candidate otherwise.
  return currentStep.value.candidates[0];
});
</script>
