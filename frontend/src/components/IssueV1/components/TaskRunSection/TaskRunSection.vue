<template>
  <div class="issue-debug">
    <div>task run section</div>
    <div>mode: {{ mode }}</div>
    <div>flattenTaskRunList.length: {{ flattenTaskRunList.length }}</div>
  </div>
  <div v-if="flattenTaskRunList.length > 0" class="px-4 py-2 space-y-4">
    <TaskRunTable :task-run-list="flattenTaskRunList" />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";

import { useIssueContext } from "../../logic";
import { extractTaskUID } from "@/utils";
import TaskRunTable from "./TaskRunTable.vue";

type ViewMode = "SINGLE" | "MERGED";

const { issue, selectedTask, isGhostMode, isPITRMode, isTenantMode } =
  useIssueContext();

/**
 * MERGED mode: merge all tasks' activities into one table
 * SINGLE mode: show only selected task's activities
 */
const mode = computed((): ViewMode => {
  if (isGhostMode.value) return "MERGED";
  if (isPITRMode.value) return "MERGED";
  if (isTenantMode.value) return "SINGLE";
  return "SINGLE";
});

const flattenTaskRunList = computed(() => {
  if (mode.value === "SINGLE") {
    return issue.value.rolloutTaskRunList.filter(
      (taskRun) => extractTaskUID(taskRun.name) === selectedTask.value.uid
    );
  }
  if (mode.value === "MERGED") {
    return issue.value.rolloutTaskRunList;
  }
  return [];
});
</script>
