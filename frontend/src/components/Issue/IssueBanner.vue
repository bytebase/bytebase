<template>
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
    v-else-if="showPendingApproval"
    class="h-8 w-full text-base font-medium bg-accent text-white flex justify-center items-center"
  >
    {{ $t("issue.waiting-approval") }}
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

<script lang="ts" setup>
import { computed, Ref } from "vue";
import dayjs from "dayjs";

import { Issue } from "@/types";
import { activeTask } from "@/utils";
import { useIssueLogic } from "./logic";

const issue = useIssueLogic().issue as Ref<Issue>;

const showCancelBanner = computed(() => {
  if (issue.value.status == "CANCELED") {
    return true;
  }

  const task = activeTask(issue.value.pipeline);
  return task.status == "CANCELED";
});

const showSuccessBanner = computed(() => {
  return issue.value.status == "DONE";
});

const showPendingApproval = computed(() => {
  const task = activeTask(issue.value.pipeline);
  return task.status == "PENDING_APPROVAL";
});

const showEarliestAllowedTimeBanner = computed(() => {
  const task = activeTask(issue.value.pipeline);

  if (task.status !== "PENDING") {
    return false;
  }

  const now = Math.floor(Date.now() / 1000);
  return task.earliestAllowedTs > now;
});

const earliestAllowedTime = computed(() => {
  const task = activeTask(issue.value.pipeline);
  const tz = "UTC" + dayjs().format("ZZ");
  return dayjs(task.earliestAllowedTs * 1000).format(
    `YYYY-MM-DD HH:mm:ss ${tz}`
  );
});
</script>
