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
            <template #user>{{ approverInCurrentStep?.title }}</template>
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
    <template v-else>
      <span>-</span>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computedAsync } from "@vueuse/core";
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { toRef } from "vue";
import { BBAvatar } from "@/bbkit";
import {
  extractReviewContext,
  useWrappedReviewStepsV1,
} from "@/components/IssueV1";
import { useCurrentUserV1, useUserStore } from "@/store";
import { userNamePrefix } from "@/store/modules/v1/common";
import { type ComposedIssue } from "@/types";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";

const props = defineProps<{
  issue: ComposedIssue;
}>();

const context = extractReviewContext(toRef(props, "issue"));
const { ready, done } = context;
const me = useCurrentUserV1();
const userStore = useUserStore();

const wrappedSteps = useWrappedReviewStepsV1(toRef(props, "issue"), context);

const currentStep = computed(() => {
  return wrappedSteps.value.find(
    (step) => step.status === "CURRENT" || step.status === "REJECTED"
  );
});

const approverInCurrentStep = computedAsync(() => {
  if (!currentStep.value?.approver) {
    return;
  }
  return userStore.getOrFetchUserByIdentifier(currentStep.value.approver);
});

const currentApprover = computedAsync(() => {
  if (!currentStep.value) return undefined;
  const includeMyself = currentStep.value.candidates.includes(
    `${userNamePrefix}${me.value.email}`
  );
  // Show currentUser if currentUser is one of the validate approver candidates.
  if (includeMyself) return me.value;
  // Show the first approver candidate otherwise.
  return userStore.getOrFetchUserByIdentifier(currentStep.value.candidates[0]);
});
</script>
