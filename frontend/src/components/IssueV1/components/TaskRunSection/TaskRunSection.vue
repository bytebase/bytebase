<template>
  <div v-if="flattenTaskRunList.length > 0" class="px-4 py-2 space-y-4">
    <TaskRunTable :task-run-list="flattenTaskRunList" />
  </div>
</template>

<script lang="ts" setup>
import { computed, watchEffect } from "vue";
import {
  useIssueContext,
  taskRunListForTask,
} from "@/components/IssueV1/logic";
import { rolloutServiceClient } from "@/grpcweb";
import { TaskRun_Status } from "@/types/proto/v1/rollout_service";
import TaskRunTable from "./TaskRunTable.vue";

type ViewMode = "SINGLE" | "MERGED";

const { issue, selectedTask, isGhostMode } = useIssueContext();

/**
 * MERGED mode: merge all tasks' activities into one table
 * SINGLE mode: show only selected task's activities
 */
const mode = computed((): ViewMode => {
  if (isGhostMode.value) return "MERGED";
  return "SINGLE";
});

const flattenTaskRunList = computed(() => {
  if (mode.value === "SINGLE") {
    return taskRunListForTask(issue.value, selectedTask.value);
  }
  if (mode.value === "MERGED") {
    return issue.value.rolloutTaskRunList;
  }
  return [];
});

watchEffect(async () => {
  // Fetching the latest task run log for running task runs of selected task.
  for (const taskRun of flattenTaskRunList.value) {
    if (taskRun.status === TaskRun_Status.RUNNING) {
      const taskRunLog = await rolloutServiceClient.getTaskRunLog({
        parent: taskRun.name,
      });
      taskRun.taskRunLog = taskRunLog;
    }
  }
});
</script>
