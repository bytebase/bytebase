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
      <router-link v-if="rolloutRoute" :to="rolloutRoute">
        <NButton quaternary size="small">
          <template #icon>
            <ExternalLinkIcon class="w-4 h-4" />
          </template>
          {{ $t("common.rollout") }}
        </NButton>
      </router-link>

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
import { ExternalLinkIcon, MenuIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed } from "vue";
import type { RouteLocationRaw } from "vue-router";
import { useSidebarContext } from "@/components/Plan";
import { PROJECT_V1_ROUTE_PLAN_ROLLOUT } from "@/router/dashboard/projectV1";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import {
  activeTaskInRollout,
  extractPlanUID,
  extractProjectResourceName,
  isDatabaseChangeRelatedIssue,
} from "@/utils";
import { useIssueContext } from "../../logic";
import IssueStatusIcon from "../IssueStatusIcon.vue";
import Actions from "./Actions";
import Title from "./Title.vue";

const { isCreating, issue } = useIssueContext();

const rolloutRoute = computed((): RouteLocationRaw | undefined => {
  const plan = issue.value.planEntity;
  if (!plan || !plan.hasRollout) return undefined;

  return {
    name: PROJECT_V1_ROUTE_PLAN_ROLLOUT,
    params: {
      projectId: extractProjectResourceName(plan.name),
      planId: extractPlanUID(plan.name),
    },
  };
});

const issueTaskStatus = computed(() => {
  // For grant request issue, we always show the status as "NOT_STARTED" as task status.
  if (!isDatabaseChangeRelatedIssue(issue.value)) {
    return Task_Status.NOT_STARTED;
  }

  return activeTaskInRollout(issue.value.rolloutEntity).status;
});

const { mode: sidebarMode, mobileSidebarOpen } = useSidebarContext();
</script>
