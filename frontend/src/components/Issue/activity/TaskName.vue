<template>
  <router-link
    :to="link"
    exact-active-class=""
    class="font-medium text-main hover:border-b hover:border-b-main"
    @click="toTop"
  >
    <span>{{ task.name }}</span>
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
import { Issue, Task } from "@/types";

const props = defineProps<{
  issue: Issue;
  task: Task;
}>();

const schemaVersion = computed(() => {
  const { task } = props;

  const taskPayload = task.payload as { schemaVersion?: string };
  return taskPayload.schemaVersion ?? "";
});

const link = computed(() => {
  const { issue, task } = props;
  const query: Record<string, any> = {
    task: task.id,
  };

  const stageIndex =
    issue.pipeline?.stageList.findIndex((stage) => {
      return stage.taskList.findIndex((t) => t.id === task.id) >= 0;
    }) ?? -1;
  if (stageIndex >= 0) {
    query.stage = stageIndex + 1;
  }

  return {
    path: `/issue/${issue.id}`,
    query,
  };
});

const toTop = (e: Event) => {
  const taskElem = document.querySelector(`[data-task-id="${props.task.id}"]`);
  if (taskElem) {
    scrollIntoView(taskElem, {
      scrollMode: "if-needed",
    });
  }
};
</script>
