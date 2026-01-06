<template>
  <div v-if="task" class="w-full h-full flex flex-col gap-y-4 p-4">
    <!-- Task Basic Info -->
    <div class="w-full flex flex-col gap-y-3">
      <div
        class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3"
      >
        <div class="flex flex-row items-center gap-x-3 flex-wrap">
          <NTag round>
            {{ selectedStageTitle }}
          </NTag>
          <TaskStatus :status="task.status" />
          <div class="flex flex-row items-center text-xl min-w-0">
            <DatabaseDisplay :database="database.name" :size="'large'" />
          </div>
          <div class="flex flex-row gap-x-2 flex-wrap">
            <NTooltip v-if="task.runTime && task.status === Task_Status.PENDING">
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
              <div class="flex flex-col gap-y-1">
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

        <div class="flex justify-start sm:justify-end">
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
      <TaskRunTable :task-runs="taskRuns" />
    </div>

    <!-- Sheet Statement -->
    <div v-if="!isReleaseTask" class="w-full flex-1 min-h-0">
      <div class="flex items-center justify-between mb-2">
        <span class="text-base font-medium">{{ $t("common.statement") }}</span>
        <div>
          <RouterLink
            v-if="relatedSpec"
            :to="{
              name: PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL,
              params: {
                projectId: extractProjectResourceName(plan.name),
                planId: extractPlanUID(plan.name),
                specId: relatedSpec.id,
              },
            }"
          >
            <NButton size="small" text type="primary">
              <template #icon>
                <LinkIcon :size="14" />
              </template>
              <span class="underline">
                {{ $t("plan.spec.change") }}: {{ getSpecTitle(relatedSpec) }}
              </span>
            </NButton>
          </RouterLink>
        </div>
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

    <!-- Release Info for release-based tasks -->
    <ReleaseInfoCard
      v-else
      :release-name="releaseName"
      class="w-full"
    />
  </div>
</template>

<script setup lang="ts">
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import dayjs from "dayjs";
import { last } from "lodash-es";
import { CalendarClockIcon, LinkIcon } from "lucide-vue-next";
import { NButton, NTag, NTooltip } from "naive-ui";
import { computed, watchEffect } from "vue";
import { RouterLink } from "vue-router";
import TaskRunDetail from "@/components/IssueV1/components/TaskRunSection/TaskRunDetail.vue";
import { MonacoEditor } from "@/components/MonacoEditor";
import DatabaseDisplay from "@/components/Plan/components/common/DatabaseDisplay.vue";
import {
  getSpecTitle,
  usePlanContextWithRollout,
} from "@/components/Plan/logic";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import { PROJECT_V1_ROUTE_PLAN_DETAIL_SPEC_DETAIL } from "@/router/dashboard/projectV1";
import {
  taskRunNamePrefix,
  useCurrentProjectV1,
  useEnvironmentV1Store,
  useSheetV1Store,
} from "@/store";
import { getTimeForPbTimestampProtoEs, unknownTask } from "@/types";
import {
  Task_Status,
  TaskRun_Status,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  databaseForTask,
  extractPlanUID,
  extractProjectResourceName,
  getSheetStatement,
  isReleaseBasedTask,
  releaseNameOfTaskV1,
  sheetNameOfTaskV1,
} from "@/utils";
import ReleaseInfoCard from "./ReleaseInfoCard.vue";
import TaskRollbackButton from "./TaskRollbackButton.vue";
import TaskRunTable from "./TaskRunTable.vue";
import TaskStatusActions from "./TaskStatusActions.vue";

const props = defineProps<{
  planId: string;
  stageId: string;
  taskId: string;
}>();

const { project } = useCurrentProjectV1();
const {
  rollout,
  taskRuns: allTaskRuns,
  readonly,
  events,
  plan,
} = usePlanContextWithRollout();
const sheetStore = useSheetV1Store();
const environmentStore = useEnvironmentV1Store();

// Stage-related computed properties
// Stage → Task relationship: Access tasks via stage.tasks array
const selectedStage = computed(() => {
  return rollout.value.stages.find((s) => s.id === props.stageId);
});

const selectedStageEnvironment = computed(() => {
  if (!selectedStage.value?.environment) return undefined;
  return environmentStore.getEnvironmentByName(selectedStage.value.environment);
});

const selectedStageTitle = computed(() => {
  // Prefer environment title, fallback to stage name
  return (
    selectedStageEnvironment.value?.title || selectedStage.value?.name || ""
  );
});

// Task-related computed properties
// Efficient lookup: We already have the stage from URL, so search within that stage's tasks only
const task = computed(() => {
  return (
    selectedStage.value?.tasks.find((t) =>
      t.name.endsWith(`/${props.taskId}`)
    ) || unknownTask()
  );
});

const isReleaseTask = computed(() => isReleaseBasedTask(task.value));

// Release name for release-based tasks
const releaseName = computed(() => releaseNameOfTaskV1(task.value));

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
    latestTaskRun.value.hasPriorBackup
  ) {
    return latestTaskRun.value;
  }
  return undefined;
});

// Task basic info
const database = computed(() => databaseForTask(project.value, task.value));

// Related spec
const relatedSpec = computed(() => {
  if (!plan.value || !task.value.specId) return undefined;
  return plan.value.specs.find((spec) => spec.id === task.value.specId);
});

const formatFullDateTime = (timestamp?: Timestamp) => {
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
