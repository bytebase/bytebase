<template>
  <div
    class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-2 py-2"
    :class="sidebarMode === 'MOBILE' ? 'pl-4 pr-2' : 'px-4'"
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

    <div class="flex flex-row items-center justify-end">
      <Actions />

      <NButton
        v-if="sidebarMode === 'MOBILE'"
        :quaternary="true"
        size="medium"
        style="--n-padding: 0 4px"
        @click="mobileSidebarOpen = true"
      >
        <MenuIcon class="w-6 h-6" />
      </NButton>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { MenuIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed } from "vue";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import { activeTaskInRollout, isDatabaseRelatedIssue } from "@/utils";
import { useIssueContext, useIssueSidebarContext } from "../../logic";
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

const { mode: sidebarMode, mobileSidebarOpen } = useIssueSidebarContext();
</script>
