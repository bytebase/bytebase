<template>
  <router-link
    :to="link"
    exact-active-class=""
    class="font-medium text-main hover:border-b hover:border-b-main"
    @click="toTop"
  >
    <span>{{ task.title }}</span>
    <template v-if="schemaVersion">
      <span class="ml-1 text-control-placeholder">(</span>
      <span class="lowercase text-control-placeholder">{{
        $t("common.schema-version")
      }}</span>
      <span class="ml-1 text-control-placeholder">{{ schemaVersion }}</span>
      <span class="text-control-placeholder">)</span>
    </template>
  </router-link>
</template>

<script lang="ts" setup>
import scrollIntoView from "scroll-into-view-if-needed";
import { computed } from "vue";
import { stageForTask } from "@/components/IssueV1/logic";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/router/dashboard/projectV1";
import type { ComposedIssue } from "@/types";
import type { Task } from "@/types/proto/v1/rollout_service";
import {
  extractProjectResourceName,
  extractSchemaVersionFromTask,
  extractStageUID,
  extractTaskUID,
  issueV1Slug,
} from "@/utils";

const props = defineProps<{
  issue: ComposedIssue;
  task: Task;
}>();

const schemaVersion = computed(() => {
  return extractSchemaVersionFromTask(props.task);
});

const link = computed(() => {
  const { issue, task } = props;

  const query: Record<string, any> = {
    task: extractTaskUID(task.name),
  };

  const stage = stageForTask(issue, task);
  if (stage) {
    query.stage = extractStageUID(stage.name);
  }

  return {
    name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
    params: {
      projectId: extractProjectResourceName(issue.project),
      issueSlug: issueV1Slug(issue),
    },
    query,
  };
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
