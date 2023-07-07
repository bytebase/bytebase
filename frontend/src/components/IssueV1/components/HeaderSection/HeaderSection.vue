<template>
  <div
    class="flex flex-col gap-2 md:flex-row md:items-center md:justify-between px-4 py-2"
  >
    <div class="flex-1">
      <div class="flex flex-col gap-y-1">
        <div class="flex items-center gap-x-2">
          <div v-if="!isCreating">
            <IssueStatusIcon
              :issue-status="issue.status"
              :task-status="issueTaskStatus"
            />
          </div>

          <IssueTitle />
        </div>

        <IssueDescription />

        <IssueVCSInfo />

        <slot name="tips"></slot>
      </div>
    </div>
    <div>
      <IssueActions />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";

import IssueStatusIcon from "../IssueStatusIcon.vue";
import { activeTaskInRollout, isDatabaseRelatedIssue } from "@/utils";
import { useIssueContext } from "../../logic";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import IssueTitle from "./IssueTitle.vue";
import IssueDescription from "./IssueDescription.vue";
import IssueVCSInfo from "./IssueVCSInfo.vue";
import IssueActions from "./IssueActions.vue";

const { isCreating, issue } = useIssueContext();

const issueTaskStatus = computed(() => {
  // For grant request issue, we always show the status as "PENDING_APPROVAL" as task status.
  if (!isDatabaseRelatedIssue(issue.value)) {
    return Task_Status.PENDING_APPROVAL;
  }

  return activeTaskInRollout(issue.value.rolloutEntity).status;
});
</script>
