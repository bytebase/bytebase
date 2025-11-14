<template>
  <div
    v-if="showPendingReview"
    class="h-8 w-full text-base font-medium bg-accent text-white flex justify-center items-center"
  >
    {{ $t("issue.waiting-for-review") }}
  </div>
  <div
    v-else-if="showRejectedReview"
    class="h-8 w-full text-base font-medium bg-warning text-white flex justify-center items-center"
  >
    {{ $t("issue.review-sent-back") }}
  </div>
  <div
    v-else-if="showClosedBanner"
    class="h-8 w-full text-base font-medium bg-gray-400 text-white flex justify-center items-center"
  >
    {{ $t("common.closed") }}
  </div>
  <div
    v-else-if="showSuccessBanner"
    class="h-8 w-full text-base font-medium text-white flex justify-center items-center"
    :class="isUnfinishedResolvedIssue ? 'bg-warning' : 'bg-success'"
  >
    {{ $t("common.done") }}
    <span v-if="isUnfinishedResolvedIssue" class="text-sm ml-2">
      {{ $t("issue.some-tasks-are-not-executed-successfully") }}
    </span>
  </div>
  <div
    v-else-if="showPendingRollout"
    class="h-8 w-full text-base font-medium bg-accent text-white flex justify-center items-center"
  >
    {{ $t("issue.awaiting-rollout") }}
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import {
  Issue_ApprovalStatus,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { activeTaskInRollout, isDatabaseChangeRelatedIssue } from "@/utils";
import {
  isUnfinishedResolvedTask as checkUnfinishedResolvedTask,
  useIssueContext,
} from "../logic";

const { issue } = useIssueContext();

const showPendingReview = computed(() => {
  return (
    issue.value.status === IssueStatus.OPEN &&
    issue.value.approvalStatus === Issue_ApprovalStatus.PENDING
  );
});

const showRejectedReview = computed(() => {
  return (
    issue.value.status === IssueStatus.OPEN &&
    issue.value.approvalStatus === Issue_ApprovalStatus.REJECTED
  );
});

const showClosedBanner = computed(() => {
  return issue.value.status === IssueStatus.CANCELED;
});

const showSuccessBanner = computed(() => {
  return issue.value.status === IssueStatus.DONE;
});

const showPendingRollout = computed(() => {
  if (issue.value.status !== IssueStatus.OPEN) return false;
  if (!isDatabaseChangeRelatedIssue(issue.value)) return false;

  const task = activeTaskInRollout(issue.value.rolloutEntity);
  return task.status === Task_Status.NOT_STARTED;
});

const isUnfinishedResolvedIssue = computed(() => {
  return checkUnfinishedResolvedTask(issue.value);
});
</script>
