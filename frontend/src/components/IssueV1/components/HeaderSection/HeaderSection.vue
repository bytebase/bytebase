<template>
  <div
    class="flex flex-col md:flex-row md:items-start md:justify-between gap-2 px-4 py-2"
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
