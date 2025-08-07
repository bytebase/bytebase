<template>
  <div v-if="task" class="w-full h-full flex flex-col gap-y-4 p-4">
    <!-- Task Basic Info -->
    <div class="w-full flex flex-col gap-y-3">
      <div class="flex flex-row items-center justify-between">
        <div class="flex flex-row items-center gap-x-3">
          <TaskStatus :size="'large'" :status="task.status" />
          <div class="flex flex-row items-center text-xl">
            <DatabaseDisplay :database="database.name" :size="'large'" />
          </div>
          <div class="flex flex-row gap-x-2">
            <NTag round>{{ semanticTaskType(task.type) }}</NTag>
            <NTooltip v-if="schemaVersion">
              <template #trigger>
                <NTag round>{{ schemaVersion }}</NTag>
              </template>
              {{ $t("common.version") }}
            </NTooltip>
            <NTooltip v-if="task.runTime">
              <template #trigger>
                <NTag round>
                  <div class="flex items-center gap-1">
                    <CalendarClockIcon class="w-4 h-4 opacity-80" />
                    <span>{{
                      humanizeTs(
                        getTimeForPbTimestampProtoEs(task.runTime, 0) / 1000
                      )
                    }}</span>
                  </div>
                </NTag>
              </template>
              <div class="space-y-1">
                <div class="text-sm opacity-80">
                  {{ $t("task.scheduled-time") }}
                </div>
                <div class="text-sm whitespace-nowrap">
                  {{ formatFullDateTime(task.runTime) }}
                </div>
              </div>
            </NTooltip>
          </div>
        </div>

        <div class="flex justify-end">
          <!-- Task Status Actions -->
          <TaskStatusActions
            :task="task"
            :task-runs="taskRuns"
            :rollout="rollout"
            :readonly="readonly"
            @action-confirmed="handleTaskActionConfirmed"
          />
          <!-- Rollback entry -->
          <TaskRollbackButton
            v-if="rollbackableTaskRun"
            :task="task"
            :task-run="rollbackableTaskRun"
            :rollout="rollout"
          />
        </div>
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
import dayjs from "dayjs";
import { last } from "lodash-es";
import { CalendarClockIcon } from "lucide-vue-next";
import { NTag, NTooltip } from "naive-ui";
import { computed, watchEffect } from "vue";
import { semanticTaskType } from "@/components/IssueV1";
import TaskRunDetail from "@/components/IssueV1/components/TaskRunSection/TaskRunDetail.vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import { CopyButton } from "@/components/v2";
import {
  taskRunNamePrefix,
  useCurrentProjectV1,
  useSheetV1Store,
} from "@/store";
import { getTimeForPbTimestampProtoEs, unknownTask } from "@/types";
import { TaskRun_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { databaseForTask } from "@/utils";
import {
  extractSchemaVersionFromTask,
  getSheetStatement,
  sheetNameOfTaskV1,
} from "@/utils";
import { usePlanContextWithRollout } from "../../logic";
import DatabaseDisplay from "../common/DatabaseDisplay.vue";
import TaskRollbackButton from "./TaskRollbackButton.vue";
import TaskRunTable from "./TaskRunTable.vue";
import TaskStatusActions from "./TaskStatusActions.vue";

const props = defineProps<{
  rolloutId: string;
  stageId: string;
  taskId: string;
}>();

const { project } = useCurrentProjectV1();
const {
  rollout,
  taskRuns: allTaskRuns,
  readonly,
  events,
} = usePlanContextWithRollout();
const sheetStore = useSheetV1Store();

const task = computed(() => {
  return (
    rollout.value.stages
      .find((s) => s.id === props.stageId)
      ?.tasks.find((t) => t.name.endsWith(`/${props.taskId}`)) || unknownTask()
  );
});

const taskRuns = computed(() => {
  return allTaskRuns.value.filter((run) =>
    run.name.startsWith(`${task.value.name}/${taskRunNamePrefix}`)
  );
});

const latestTaskRun = computed(() => last(taskRuns.value));

const rollbackableTaskRun = computed(() => {
  if (
    latestTaskRun.value &&
    latestTaskRun.value.status === TaskRun_Status.DONE &&
    latestTaskRun.value.priorBackupDetail !== undefined
  ) {
    return latestTaskRun.value;
  }
  return undefined;
});

// Task basic info
const database = computed(() => databaseForTask(project.value, task.value));
const schemaVersion = computed(() => extractSchemaVersionFromTask(task.value));

const formatFullDateTime = (timestamp: any) => {
  const timestampInMilliseconds = getTimeForPbTimestampProtoEs(timestamp, 0);
  return dayjs(timestampInMilliseconds).local().format();
};

// Sheet statement
const statement = computed(() => {
  const sheet = sheetStore.getSheetByName(sheetNameOfTaskV1(task.value));
  if (sheet) {
    return getSheetStatement(sheet);
  }
  return "";
});

// Prepare sheet of the task.
watchEffect(async () => {
  const sheetName = sheetNameOfTaskV1(task.value);
  if (sheetName) {
    await sheetStore.getOrFetchSheetByName(sheetName);
  }
});

// Handle task action completion to refresh data
const handleTaskActionConfirmed = async () => {
  events.emit("status-changed", { eager: true });
};
</script>
