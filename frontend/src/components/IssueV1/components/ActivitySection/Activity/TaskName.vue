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
import { computed } from "vue";
import scrollIntoView from "scroll-into-view-if-needed";
import { ComposedIssue } from "@/types";
import { Task } from "@/types/proto/v1/rollout_service";
import { extractSchemaVersionFromTask } from "@/utils";
import { stageForTask } from "@/components/IssueV1/logic";

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
    task: task.uid,
  };

  const stage = stageForTask(issue, task);
  if (stage) {
    query.stage = stage.uid;
  }

  return {
    path: `/issue-v1/${issue.uid}`,
    query,
  };
});

const toTop = (e: Event) => {
  const taskElem = document.querySelector(`[data-task-id="${props.task.uid}"]`);
  if (taskElem) {
    scrollIntoView(taskElem, {
      scrollMode: "if-needed",
    });
  }
};
</script>
