<template>
  <div
    class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3 py-4"
    :class="sidebarMode === 'MOBILE' ? 'pl-6 pr-4' : 'px-6'"
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
import { useSidebarContext } from "@/components/Plan";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { activeTaskInRollout, isDatabaseChangeRelatedIssue } from "@/utils";
import { useIssueContext } from "../../logic";
import IssueStatusIcon from "../IssueStatusIcon.vue";
import Actions from "./Actions";
import Title from "./Title.vue";

const { isCreating, issue } = useIssueContext();

const issueTaskStatus = computed(() => {
  // For grant request issue, we always show the status as "NOT_STARTED" as task status.
  if (!isDatabaseChangeRelatedIssue(issue.value)) {
    return Task_Status.NOT_STARTED;
  }

  return activeTaskInRollout(issue.value.rolloutEntity).status;
});

const { mode: sidebarMode, mobileSidebarOpen } = useSidebarContext();
</script>
