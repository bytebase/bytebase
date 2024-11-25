<template>
  <template v-if="showPendingReview">
    <div
      class="h-8 w-full text-base font-medium bg-accent text-white flex justify-center items-center"
    >
      {{ $t("issue.waiting-for-review") }}
    </div>
  </template>
  <template v-else-if="showRejectedReview">
    <div
      class="h-8 w-full text-base font-medium bg-warning text-white flex justify-center items-center"
    >
      {{ $t("issue.review-sent-back") }}
    </div>
  </template>
  <template v-else>
    <div
      v-if="showClosedBanner"
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
      {{ $t("issue.waiting-to-rollout") }}
    </div>
    <div
      v-else-if="showEarliestAllowedTimeBanner"
      class="h-8 w-full text-base font-medium bg-accent text-white flex justify-center items-center"
    >
      {{
        $t("issue.waiting-earliest-allowed-time", { time: earliestAllowedTime })
      }}
    </div>
  </template>
</template>

<script lang="ts" setup>
import dayjs from "dayjs";
import { computed } from "vue";
import {
  IssueStatus,
  Issue_Approver_Status,
} from "@/types/proto/v1/issue_service";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import {
  activeTaskInRollout,
  isDatabaseChangeRelatedIssue,
} from "@/utils";
import {
  useIssueContext,
  isUnfinishedResolvedTask as checkUnfinishedResolvedTask,
} from "../logic";

const { isCreating, issue, reviewContext } = useIssueContext();
const { status: reviewStatus } = reviewContext;

const showPendingReview = computed(() => {
  if (isCreating.value) return false;
  if (issue.value.status !== IssueStatus.OPEN) return false;
  return reviewStatus.value === Issue_Approver_Status.PENDING;
});

const showRejectedReview = computed(() => {
  if (isCreating.value) return false;
  if (issue.value.status !== IssueStatus.OPEN) return false;
  return reviewStatus.value === Issue_Approver_Status.REJECTED;
});

const showClosedBanner = computed(() => {
  return issue.value.status === IssueStatus.CANCELED;
});

const showSuccessBanner = computed(() => {
  return issue.value.status === IssueStatus.DONE;
});

const showPendingRollout = computed(() => {
  if (issue.value.status !== IssueStatus.OPEN) return false;
  if (!isDatabaseChangeRelatedIssue(issue.value)) {
    return false;
  }

  const task = activeTaskInRollout(issue.value.rolloutEntity);
  return task.status === Task_Status.NOT_STARTED;
});

const showEarliestAllowedTimeBanner = computed(() => {
  if (issue.value.status !== IssueStatus.OPEN) return false;
  if (!isDatabaseChangeRelatedIssue(issue.value)) {
    return false;
  }

  const task = activeTaskInRollout(issue.value.rolloutEntity);

  if (task.status !== Task_Status.PENDING) {
    return false;
  }

  // const now = Math.floor(Date.now() / 1000);
  // return task.earliestAllowedTs > now;
  return false; // todo
});

const earliestAllowedTime = computed(() => {
  // const task = activeTaskInRollout(issue.value.rolloutEntity);
  const tz = "UTC" + dayjs().format("ZZ");
  return dayjs().format(`YYYY-MM-DD HH:mm:ss ${tz}`);
  // return dayjs(task.earliestAllowedTs * 1000).format(
  //   `YYYY-MM-DD HH:mm:ss ${tz}`
  // );
});

const isUnfinishedResolvedIssue = computed(() => {
  return checkUnfinishedResolvedTask(issue.value);
});
</script>
