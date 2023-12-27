<template>
  <div
    class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-2 px-4 py-2"
  >
    <div class="flex-1 flex items-center gap-x-2">
      <IssueStatusIcon
        v-if="!isCreating"
        :issue-status="issue.status"
        :task-status="issueTaskStatus"
        :issue="issue"
      />

      <Title />
    </div>

    <Actions />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import { activeTaskInRollout, isDatabaseRelatedIssue } from "@/utils";
import { useIssueContext } from "../../logic";
import IssueStatusIcon from "../IssueStatusIcon.vue";
import Actions from "./Actions";
import Title from "./Title.vue";

const { isCreating, issue } = useIssueContext();

const issueTaskStatus = computed(() => {
  // For grant request issue, we always show the status as "NOT_STARTED" as task status.
  if (!isDatabaseRelatedIssue(issue.value)) {
    return Task_Status.NOT_STARTED;
  }

  return activeTaskInRollout(issue.value.rolloutEntity).status;
});
</script>
