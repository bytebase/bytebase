<template>
  <div class="flex flex-col py-2">
    <div
      class="flex flex-col md:flex-row md:items-start md:justify-between gap-2 px-4"
    >
      <div class="flex-1 flex items-center gap-x-2">
        <div v-if="!isCreating">
          <IssueStatusIcon
            :issue-status="issue.status"
            :task-status="issueTaskStatus"
            :issue="issue"
          />
        </div>

        <Title />
      </div>

      <Actions />
    </div>

    <div
      class="w-full flex flex-col border-t mt-2 pt-6 md:flex-row md:items-stretch md:justify-between gap-2 px-4"
    >
      <div class="flex-1 flex flex-col gap-x-2">
        <VCSInfo />
        <RollbackFromTips />
      </div>

      <div class="flex flex-col items-end gap-y-2">
        <ReviewSection v-if="!isCreating" />
        <Assignee v-if="shouldShowAssignee" />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { usePageMode } from "@/store";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import { activeTaskInRollout, isDatabaseRelatedIssue } from "@/utils";
import { useIssueContext } from "../../logic";
import IssueStatusIcon from "../IssueStatusIcon.vue";
import Actions from "./Actions";
import Assignee from "./Assignee";
import ReviewSection from "./ReviewSection";
import RollbackFromTips from "./RollbackFromTips.vue";
import Title from "./Title.vue";
import VCSInfo from "./VCSInfo.vue";

const { isCreating, issue } = useIssueContext();
const pageMode = usePageMode();

const issueTaskStatus = computed(() => {
  // For grant request issue, we always show the status as "NOT_STARTED" as task status.
  if (!isDatabaseRelatedIssue(issue.value)) {
    return Task_Status.NOT_STARTED;
  }

  return activeTaskInRollout(issue.value.rolloutEntity).status;
});

const shouldShowAssignee = computed(() => {
  return pageMode.value === "BUNDLED" && isDatabaseRelatedIssue(issue.value);
});
</script>
