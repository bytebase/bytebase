<template>
  <div class="flex flex-col gap-y-4 p-4">
    <!-- Header: Stage + Status + Database + Scheduled time -->
    <div class="flex flex-col gap-y-3">
      <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
        <div class="flex flex-row items-center gap-x-3 flex-wrap">
          <TaskStatus :status="task.status" />
          <NTag v-if="stageTitle" round>
            {{ stageTitle }}
          </NTag>
          <DatabaseDisplay :database="database.name" size="large" />
          <NTooltip v-if="task.runTime && task.status === Task_Status.PENDING">
            <template #trigger>
              <NTag round size="small">
                <div class="flex items-center gap-1">
                  <CalendarClockIcon class="w-3.5 h-3.5 opacity-80" />
                  <span class="text-xs">{{ scheduledTimeDisplay }}</span>
                </div>
              </NTag>
            </template>
            {{ $t("task.scheduled-time") }}
          </NTooltip>
        </div>

        <!-- Actions -->
        <div v-if="stage" class="flex items-center gap-x-2">
          <TaskStatusActions
            :task="task"
            :stage="stage"
            @action-confirmed="handleTaskActionConfirmed"
          />
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
    <div v-if="latestTaskRun" class="flex flex-col gap-2">
      <span class="textlabel uppercase">{{ $t("task-run.latest-logs") }}</span>
      <TaskRunDetail :task-run="latestTaskRun" :database="database" />
    </div>

    <!-- Task Runs History -->
    <div v-if="taskRunsForTask.length > 0" class="flex flex-col gap-2">
      <span class="textlabel uppercase">{{ $t("task-run.history") }}</span>
      <TaskRunTable :task-runs="taskRunsForTask" />
    </div>

    <!-- SQL Statement -->
    <div v-if="statement" class="flex flex-col gap-2">
      <span class="textlabel uppercase">{{ $t("common.statement") }}</span>
      <MonacoEditor
        class="border rounded-[3px] text-sm overflow-hidden"
        :content="statement"
        :readonly="true"
        :auto-height="{ min: 200, max: 480 }"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { last } from "lodash-es";
import { CalendarClockIcon } from "lucide-vue-next";
import { NTag, NTooltip } from "naive-ui";
import { computed, watch } from "vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import DatabaseDisplay from "@/components/Plan/components/common/DatabaseDisplay.vue";
import TaskRollbackButton from "@/components/RolloutV1/components/Task/TaskRollbackButton.vue";
import TaskStatus from "@/components/RolloutV1/components/Task/TaskStatus.vue";
import TaskStatusActions from "@/components/RolloutV1/components/Task/TaskStatusActions.vue";
import TaskRunDetail from "@/components/RolloutV1/components/TaskRun/TaskRunDetail.vue";
import TaskRunTable from "@/components/RolloutV1/components/TaskRun/TaskRunTable.vue";
import {
  taskRunNamePrefix,
  useCurrentProjectV1,
  useEnvironmentV1Store,
  useSheetV1Store,
} from "@/store";
import { getTimeForPbTimestampProtoEs } from "@/types";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import {
  Task_Status,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import { databaseForTask, getSheetStatement, sheetNameOfTaskV1 } from "@/utils";
import { humanizeTs } from "@/utils/util";
import { emitPlanStatusChanged, usePlanContextWithRollout } from "../../logic";

const props = defineProps<{
  task: Task;
}>();

defineEmits<{
  (event: "close"): void;
}>();

const { project } = useCurrentProjectV1();
const { rollout, taskRuns: allTaskRuns, events } = usePlanContextWithRollout();
const sheetStore = useSheetV1Store();
const environmentStore = useEnvironmentV1Store();

const database = computed(() => databaseForTask(project.value, props.task));

const stage = computed(() => {
  return rollout.value.stages.find((s) =>
    s.tasks.some((t) => t.name === props.task.name)
  );
});

const stageTitle = computed(() => {
  if (!stage.value?.environment) return "";
  return (
    environmentStore.getEnvironmentByName(stage.value.environment)?.title ?? ""
  );
});

const scheduledTimeDisplay = computed(() => {
  const ts = getTimeForPbTimestampProtoEs(props.task.runTime, 0);
  if (!ts) return "";
  return humanizeTs(ts / 1000);
});

const taskRunsForTask = computed(() => {
  return allTaskRuns.value.filter((run) =>
    run.name.startsWith(`${props.task.name}/${taskRunNamePrefix}`)
  );
});

const latestTaskRun = computed(() => last(taskRunsForTask.value));

const rollbackableTaskRun = computed(() => {
  if (
    latestTaskRun.value &&
    latestTaskRun.value.status === TaskRun_Status.DONE &&
    latestTaskRun.value.hasPriorBackup
  ) {
    return latestTaskRun.value;
  }
  return undefined;
});

const statement = computed(() => {
  const sheet = sheetStore.getSheetByName(sheetNameOfTaskV1(props.task));
  if (sheet) return getSheetStatement(sheet);
  return "";
});

watch(
  () => sheetNameOfTaskV1(props.task),
  async (sheetName) => {
    if (!sheetName) return;
    try {
      await sheetStore.getOrFetchSheetByName(sheetName);
    } catch {
      // Ignore — sheet fetch is non-critical for panel display
    }
  },
  { immediate: true }
);

const handleTaskActionConfirmed = () => {
  emitPlanStatusChanged(events, { refreshMode: "fast-follow" });
};
</script>
