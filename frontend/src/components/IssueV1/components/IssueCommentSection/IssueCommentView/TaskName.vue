<template>
  <router-link
    :to="link"
    exact-active-class=""
    class="font-medium text-main hover:border-b hover:border-b-main"
    @click="toTop"
  >
    <span>{{ databaseForTask(projectOfIssue(issue), task).databaseName }}</span>
  </router-link>
</template>

<script lang="ts" setup>
import scrollIntoView from "scroll-into-view-if-needed";
import { computed } from "vue";
import type { LocationQueryRaw } from "vue-router";
import { projectOfIssue } from "@/components/IssueV1/logic";
import { useIssueLayoutVersion } from "@/composables/useIssueLayoutVersion";
import {
  PROJECT_V1_ROUTE_ISSUE_DETAIL,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
} from "@/router/dashboard/projectV1";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import {
  databaseForTask,
  extractPlanUIDFromRolloutName,
  extractProjectResourceName,
  extractStageNameFromTaskName,
  extractStageUID,
  extractTaskUID,
  issueV1Slug,
} from "@/utils";

const props = defineProps<{
  issue: Issue;
  task: Task;
}>();

const { enabledNewLayout } = useIssueLayoutVersion();

const link = computed(() => {
  const { issue, task } = props;

  if (enabledNewLayout.value) {
    return {
      name: PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
      params: {
        projectId: extractProjectResourceName(task.name),
        planId: extractPlanUIDFromRolloutName(task.name),
        stageId: extractStageUID(extractStageNameFromTaskName(task.name)),
        taskId: extractTaskUID(task.name),
      },
    };
  } else {
    const query: LocationQueryRaw = {
      task: extractTaskUID(task.name),
      stage: extractStageUID(task.name),
    };

    return {
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
      params: {
        projectId: extractProjectResourceName(issue.name),
        issueSlug: issueV1Slug(issue.name, issue.title),
      },
      query,
    };
  }
});

const toTop = () => {
  const taskElem = document.querySelector(
    `[data-task-id="${extractTaskUID(props.task.name)}"]`
  );
  if (taskElem) {
    scrollIntoView(taskElem, {
      scrollMode: "if-needed",
    });
  }
};
</script>
