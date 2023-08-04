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
      v-if="showCancelBanner"
      class="h-8 w-full text-base font-medium bg-gray-400 text-white flex justify-center items-center"
    >
      {{ $t("common.canceled") }}
    </div>
    <div
      v-else-if="showSuccessBanner"
      class="h-8 w-full text-base font-medium bg-success text-white flex justify-center items-center"
    >
      {{ $t("common.done") }}
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
import { computed } from "vue";
import dayjs from "dayjs";

import { activeTaskInRollout, isDatabaseRelatedIssue } from "@/utils";
import {
  IssueStatus,
  Issue_Approver_Status,
} from "@/types/proto/v1/issue_service";
import { useIssueContext } from "../logic";
import { Task_Status } from "@/types/proto/v1/rollout_service";

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

const showCancelBanner = computed(() => {
  return issue.value.status === IssueStatus.CANCELED;
});

const showSuccessBanner = computed(() => {
  return issue.value.status === IssueStatus.DONE;
});

const showPendingRollout = computed(() => {
  if (issue.value.status !== IssueStatus.OPEN) return false;
  if (!isDatabaseRelatedIssue(issue.value)) {
    return false;
  }

  const task = activeTaskInRollout(issue.value.rolloutEntity);
  return task.status === Task_Status.NOT_STARTED;
});

const showEarliestAllowedTimeBanner = computed(() => {
  if (issue.value.status !== IssueStatus.OPEN) return false;
  if (!isDatabaseRelatedIssue(issue.value)) {
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
</script>
