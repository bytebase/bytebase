<template>
  <div class="flex flex-col gap-2 px-4 py-2">
    <div
      class="flex flex-col md:flex-row md:items-stretch md:justify-between gap-2"
    >
      <div class="flex-1 flex items-center gap-x-2">
        <div v-if="!isCreating">
          <IssueStatusIcon
            :issue-status="issue.status"
            :task-status="issueTaskStatus"
          />
        </div>

        <Title />
      </div>

      <Actions />
    </div>
    <div
      class="flex flex-col md:flex-row md:items-stretch md:justify-between gap-2"
    >
      <div class="flex-1 flex flex-col gap-x-2">
        <Description />

        <VCSInfo />

        <RollbackFromTips />
      </div>

      <div class="flex flex-col items-end gap-y-2">
        <ReviewSection v-if="!isCreating" />
        <Assignee />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";

import IssueStatusIcon from "../IssueStatusIcon.vue";
import { activeTaskInRollout, isDatabaseRelatedIssue } from "@/utils";
import { useIssueContext } from "../../logic";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import Title from "./Title.vue";
import Description from "./Description.vue";
import VCSInfo from "./VCSInfo.vue";
import Actions from "./Actions";
import ReviewSection from "./ReviewSection";
import Assignee from "./Assignee";
import RollbackFromTips from "./RollbackFromTips.vue";

const { isCreating, issue } = useIssueContext();

const issueTaskStatus = computed(() => {
  // For grant request issue, we always show the status as "PENDING_APPROVAL" as task status.
  if (!isDatabaseRelatedIssue(issue.value)) {
    return Task_Status.PENDING_APPROVAL;
  }

  return activeTaskInRollout(issue.value.rolloutEntity).status;
});
</script>
