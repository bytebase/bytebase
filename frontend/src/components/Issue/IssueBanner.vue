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
import { computed, Ref } from "vue";
import dayjs from "dayjs";

import { Issue } from "@/types";
import { activeTask, isDatabaseRelatedIssueType } from "@/utils";
import { useIssueLogic } from "./logic";
import { useIssueReviewContext } from "@/plugins/issue/logic/review/context";
import { Issue_Approver_Status } from "@/types/proto/v1/issue_service";

const issueContext = useIssueLogic();
const issue = issueContext.issue as Ref<Issue>;
const reviewContext = useIssueReviewContext();

const showPendingReview = computed(() => {
  if (issueContext.create.value) return false;
  if (issue.value.status !== "OPEN") return false;
  return reviewContext.status.value === Issue_Approver_Status.PENDING;
});

const showRejectedReview = computed(() => {
  if (issueContext.create.value) return false;
  if (issue.value.status !== "OPEN") return false;
  return reviewContext.status.value === Issue_Approver_Status.REJECTED;
});

const showCancelBanner = computed(() => {
  return issue.value.status === "CANCELED";
});

const showSuccessBanner = computed(() => {
  return issue.value.status === "DONE";
});

const showPendingRollout = computed(() => {
  if (issue.value.status !== "OPEN") return false;
  if (!isDatabaseRelatedIssueType(issue.value.type)) {
    return false;
  }

  const task = activeTask(issue.value.pipeline!);
  return task.status == "PENDING_APPROVAL";
});

const showEarliestAllowedTimeBanner = computed(() => {
  if (issue.value.status !== "OPEN") return false;
  if (!isDatabaseRelatedIssueType(issue.value.type)) {
    return false;
  }

  const task = activeTask(issue.value.pipeline!);

  if (task.status !== "PENDING") {
    return false;
  }

  const now = Math.floor(Date.now() / 1000);
  return task.earliestAllowedTs > now;
});

const earliestAllowedTime = computed(() => {
  const task = activeTask(issue.value.pipeline!);
  const tz = "UTC" + dayjs().format("ZZ");
  return dayjs(task.earliestAllowedTs * 1000).format(
    `YYYY-MM-DD HH:mm:ss ${tz}`
  );
});
</script>
