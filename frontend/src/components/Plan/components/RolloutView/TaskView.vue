<template>
  <div v-if="task" class="w-full h-full flex flex-col gap-y-4 p-4">
    <!-- Task Basic Info -->
    <div class="w-full flex flex-col gap-y-3">
      <div class="flex flex-row items-center justify-between">
        <div class="flex flex-row items-center gap-x-3">
          <TaskStatus :size="'large'" :status="task.status" />
          <div class="flex flex-row items-center text-xl">
            <InstanceV1EngineIcon
              class="mr-2"
              :size="'large'"
              :instance="database.instanceResource"
            />
            <span>{{ database.instanceResource.title }}</span>
            <ChevronRightIcon class="inline opacity-40 mx-1 w-5" />
            <span class="font-medium">{{ database.databaseName }}</span>
          </div>
          <div class="flex flex-row gap-x-2">
            <NTag round>{{ semanticTaskType(task.type) }}</NTag>
            <NTooltip v-if="schemaVersion">
              <template #trigger>
                <NTag round>{{ schemaVersion }}</NTag>
              </template>
              {{ $t("common.version") }}
            </NTooltip>
          </div>
        </div>

        <!-- Task Status Actions -->
        <TaskStatusActions
          :task="task"
          :task-runs="taskRuns"
          :rollout="rollout"
          @action-confirmed="handleTaskActionConfirmed"
        />
      </div>
    </div>

    <!-- Latest Task Run Detail -->
    <div v-if="latestTaskRun" class="w-full">
      <div class="flex items-center justify-between">
        <span class="text-base font-medium">{{
          $t("task-run.latest-logs")
        }}</span>
      </div>
      <TaskRunDetail :task-run="latestTaskRun" :database="database" />
    </div>

    <!-- Task Runs Table -->
    <div v-if="taskRuns.length > 0" class="w-full">
      <div class="flex items-center justify-between mb-2">
        <span class="text-base font-medium">{{ $t("task-run.history") }}</span>
      </div>
      <TaskRunTable :task="task" :task-runs="taskRuns" />
    </div>

    <!-- Sheet Statement -->
    <div class="w-full flex-1 min-h-0">
      <div class="flex items-center justify-between mb-2">
        <span class="text-base font-medium">{{ $t("common.statement") }}</span>
        <CopyButton v-if="statement" :size="'medium'" :content="statement" />
      </div>
      <MonacoEditor
        v-if="statement"
        class="h-full min-h-[200px] border rounded-[3px] text-sm overflow-clip"
        :content="statement"
        :readonly="true"
        :auto-height="{ min: 200, max: 480 }"
      />
      <div
        v-else
        class="h-32 border rounded-[3px] flex items-center justify-center text-control"
      >
        {{ $t("common.no-data") }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { isEqual, sortBy } from "lodash-es";
import { ChevronRightIcon } from "lucide-vue-next";
import { NTag, NTooltip } from "naive-ui";
import { computed, ref, watchEffect } from "vue";
import { semanticTaskType } from "@/components/IssueV1";
import TaskRunDetail from "@/components/IssueV1/components/TaskRunSection/TaskRunDetail.vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import { InstanceV1EngineIcon, CopyButton } from "@/components/v2";
import { rolloutServiceClientConnect } from "@/grpcweb";
import { useCurrentProjectV1, useSheetV1Store } from "@/store";
import { getDateForPbTimestampProtoEs, unknownTask } from "@/types";
import type { TaskRun } from "@/types/proto-es/v1/rollout_service_pb";
import { ListTaskRunsRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import { databaseForTask } from "@/utils";
import {
  extractSchemaVersionFromTask,
  getSheetStatement,
  sheetNameOfTaskV1,
  isValidTaskName,
} from "@/utils";
import { usePlanContextWithRollout } from "../../logic";
import TaskRunTable from "./TaskRunTable.vue";
import TaskStatusActions from "./TaskStatusActions.vue";

const props = defineProps<{
  rolloutId: string;
  stageId: string;
  taskId: string;
}>();

const { project } = useCurrentProjectV1();
const { rollout } = usePlanContextWithRollout();
const sheetStore = useSheetV1Store();
const taskRunsRef = ref<TaskRun[]>([]);

// Get the task - either from props or from fetched rollout
const task = computed(() => {
  return (
    rollout.value.stages
      .find((s) => s.id === props.stageId)
      ?.tasks.find((t) => t.name.endsWith(`/${props.taskId}`)) || unknownTask()
  );
});

// Task basic info
const database = computed(() => databaseForTask(project.value, task.value));
const schemaVersion = computed(() => extractSchemaVersionFromTask(task.value));

// Sheet statement
const statement = computed(() => {
  const sheet = sheetStore.getSheetByName(sheetNameOfTaskV1(task.value));
  if (sheet) {
    return getSheetStatement(sheet);
  }
  return "";
});

// Fetch task runs
watchEffect(async () => {
  if (!isValidTaskName(task.value.name)) {
    return;
  }

  try {
    const request = create(ListTaskRunsRequestSchema, {
      parent: task.value.name,
    });
    const response = await rolloutServiceClientConnect.listTaskRuns(request);
    const taskRuns = response.taskRuns;
    const sorted = sortBy(taskRuns, (t) =>
      getDateForPbTimestampProtoEs(t.createTime)
    ).reverse();
    if (!isEqual(sorted, taskRunsRef.value)) {
      taskRunsRef.value = sorted;
    }
  } catch (error) {
    console.error("Failed to fetch task runs:", error);
  }
});

// Fetch sheet when task changes
watchEffect(async () => {
  const sheetName = sheetNameOfTaskV1(task.value);
  if (sheetName) {
    await sheetStore.getOrFetchSheetByName(sheetName);
  }
});

// Task run info
const taskRuns = computed(() => taskRunsRef.value);
const latestTaskRun = computed(() => taskRuns.value[0]);

// Handle task action completion to refresh data
const handleTaskActionConfirmed = async () => {
  // Refresh task runs after action
  if (isValidTaskName(task.value.name)) {
    try {
      const request = create(ListTaskRunsRequestSchema, {
        parent: task.value.name,
      });
      const response = await rolloutServiceClientConnect.listTaskRuns(request);
      const taskRuns = response.taskRuns;
      const sorted = sortBy(taskRuns, (t) =>
        getDateForPbTimestampProtoEs(t.createTime)
      ).reverse();
      taskRunsRef.value = sorted;
    } catch (error) {
      console.error("Failed to refresh task runs:", error);
    }
  }
};
</script>
