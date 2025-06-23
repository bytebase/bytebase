<template>
  <div v-if="flattenTaskRunList.length > 0" class="px-4 py-2 space-y-4">
    <TaskRunTable :task-run-list="flattenTaskRunList" />
  </div>
</template>

<script lang="ts" setup>
import { computed, watchEffect } from "vue";
import { create } from "@bufbuild/protobuf";
import { GetTaskRunLogRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import { convertNewTaskRunLogToOld } from "@/utils/v1/rollout-conversions";
import {
  useIssueContext,
  taskRunListForTask,
} from "@/components/IssueV1/logic";
import { rolloutServiceClientConnect } from "@/grpcweb";
import { TaskRun_Status } from "@/types/proto/v1/rollout_service";
import TaskRunTable from "./TaskRunTable.vue";

const { issue, selectedTask } = useIssueContext();

const flattenTaskRunList = computed(() => {
  return taskRunListForTask(issue.value, selectedTask.value);
});

watchEffect(async () => {
  // Fetching the latest task run log for running task runs of selected task.
  for (const taskRun of flattenTaskRunList.value) {
    if (taskRun.status === TaskRun_Status.RUNNING) {
      const request = create(GetTaskRunLogRequestSchema, {
        parent: taskRun.name,
      });
      const response = await rolloutServiceClientConnect.getTaskRunLog(request);
      const taskRunLog = convertNewTaskRunLogToOld(response);
      taskRun.taskRunLog = taskRunLog;
    }
  }
});
</script>
