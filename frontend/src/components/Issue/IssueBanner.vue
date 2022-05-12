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
</template>

<script lang="ts" setup>
import { computed, Ref } from "vue";
import { Issue } from "@/types";
import { activeTask } from "@/utils";
import { useIssueLogic } from "./logic";

const issue = useIssueLogic().issue as Ref<Issue>;

const showCancelBanner = computed(() => {
  return issue.value.status == "CANCELED";
});

const showSuccessBanner = computed(() => {
  return issue.value.status == "DONE";
});

const showPendingApproval = computed(() => {
  const task = activeTask(issue.value.pipeline);
  return task.status == "PENDING_APPROVAL";
});
</script>
