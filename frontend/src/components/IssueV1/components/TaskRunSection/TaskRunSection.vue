<template>
  <div
    v-if="flattenTaskRunList.length > 0"
    class="px-4 py-2 flex flex-col gap-y-4"
  >
    <TaskRunTable :task-run-list="flattenTaskRunList" />
  </div>
</template>

<script lang="ts" setup>
import { computed, watchEffect } from "vue";
import {
  taskRunListForTask,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { useTaskRunLogStore } from "@/store/modules/v1/taskRunLog";
import { TaskRun_Status } from "@/types/proto-es/v1/rollout_service_pb";
import TaskRunTable from "./TaskRunTable.vue";

const { issue, selectedTask } = useIssueContext();
const taskRunLogStore = useTaskRunLogStore();

const flattenTaskRunList = computed(() => {
  return taskRunListForTask(issue.value, selectedTask.value);
});

watchEffect(async () => {
  // Fetching the latest task run log for running task runs of selected task.
  for (const taskRun of flattenTaskRunList.value) {
    if (taskRun.status === TaskRun_Status.RUNNING) {
      await taskRunLogStore.fetchTaskRunLog(taskRun.name);
    }
  }
});
</script>
