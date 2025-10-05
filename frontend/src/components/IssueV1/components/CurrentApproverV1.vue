<template>
  <div class="flex flex-row items-center gap-x-2 overflow-hidden">
    <template
      v-if="
        issue.status !== IssueStatus.OPEN || rolloutReady || !approvalFlowReady
      "
    >
      <span>-</span>
    </template>
    <template
      v-else-if="issue.approvalStatus === Issue_ApprovalStatus.REJECTED"
    >
      <NTooltip :disabled="!rejectedApprover">
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
            <template #user>{{ rejectedApprover?.title }}</template>
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
import { extractReviewContext } from "@/components/IssueV1";
import {
  candidatesOfApprovalStepV1,
  useCurrentUserV1,
  userNamePrefix,
  useUserStore,
} from "@/store";
import { type ComposedIssue } from "@/types";
import {
  IssueStatus,
  Issue_ApprovalStatus,
  Issue_Approver_Status,
} from "@/types/proto-es/v1/issue_service_pb";

const props = defineProps<{
  issue: ComposedIssue;
}>();

const flow = extractReviewContext(toRef(props, "issue"));
const me = useCurrentUserV1();
const userStore = useUserStore();

const approvalFlowReady = computed(() => {
  const approvalStatus = props.issue.approvalStatus;
  return approvalStatus !== Issue_ApprovalStatus.CHECKING;
});

const rolloutReady = computed(() => {
  const approvalStatus = props.issue.approvalStatus;
  return (
    approvalStatus === Issue_ApprovalStatus.APPROVED ||
    approvalStatus === Issue_ApprovalStatus.SKIPPED
  );
});

const rejectedApprover = computedAsync(() => {
  if (props.issue.approvalStatus !== Issue_ApprovalStatus.REJECTED) {
    return undefined;
  }
  const rejectedApproval = flow.value.approvers.find(
    (ap) => ap.status === Issue_Approver_Status.REJECTED
  );
  if (!rejectedApproval?.principal) return undefined;
  return userStore.getOrFetchUserByIdentifier(rejectedApproval.principal);
});

const currentApprover = computedAsync(() => {
  if (props.issue.approvalStatus !== Issue_ApprovalStatus.PENDING) {
    return undefined;
  }

  const currentStepIndex = flow.value.approvers.length;
  const steps = flow.value.template.flow?.steps || [];
  const step = steps[currentStepIndex];
  if (!step) return undefined;

  const candidates = candidatesOfApprovalStepV1(props.issue, step);
  const currentUserName = `${userNamePrefix}${me.value.email}`;

  // Show currentUser if currentUser is one of the validate approver candidates.
  if (candidates.includes(currentUserName)) return me.value;

  // Show the first approver candidate otherwise.
  if (candidates.length === 0) return undefined;
  return userStore.getOrFetchUserByIdentifier(candidates[0]);
});
</script>
