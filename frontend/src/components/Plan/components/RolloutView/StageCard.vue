<template>
  <div class="relative w-full flex items-center gap-4">
    <!-- Stage card -->
    <div
      class="flex-1 bg-white rounded-lg border p-4 shadow-sm"
      :class="
        twMerge(
          isCreated
            ? 'border-gray-200 cursor-pointer hover:shadow-md transition-shadow'
            : 'border-gray-300 border-dashed'
        )
      "
      @click="isCreated && handleClickStageTitle()"
    >
      <div class="flex items-start justify-between gap-4">
        <!-- Left side: Stage title and status counts -->
        <div class="flex-1 flex items-start gap-2">
          <div class="flex items-start gap-2">
            <TaskStatus :status="stageStatus" size="medium" />
            <div class="flex flex-col gap-1">
              <h3
                class="text-base font-medium text-gray-900 whitespace-nowrap truncate"
              >
                <EnvironmentV1Name
                  :environment="
                    environmentStore.getEnvironmentByName(stage.environment)
                  "
                />
              </h3>
              <Timestamp
                v-if="latestUpdateTimestamp"
                :timestamp="latestUpdateTimestamp"
              />
            </div>
          </div>

          <!-- Tasks and task status counts -->
          <div v-if="isCreated" class="flex-1 flex flex-col">
            <!-- Task status counts -->
            <div class="flex items-center gap-2 flex-wrap">
              <template v-for="status in TASK_STATUS_FILTERS" :key="status">
                <NTag
                  v-if="getTaskCount(status) > 0"
                  round
                  size="medium"
                  :type="
                    status === Task_Status.RUNNING
                      ? 'info'
                      : status === Task_Status.FAILED
                        ? 'error'
                        : 'default'
                  "
                  class="cursor-pointer hover:opacity-80 transition-opacity"
                  @click.stop="handleTaskStatusClick(status)"
                >
                  <template #avatar>
                    <TaskStatus :status="status" size="small" disabled />
                  </template>
                  <div class="flex flex-row items-center gap-2">
                    <span class="select-none text-base">{{
                      stringifyTaskStatus(status)
                    }}</span>
                    <span class="select-none text-base font-medium">{{
                      getTaskCount(status)
                    }}</span>
                  </div>
                </NTag>
              </template>
              <!-- Toggle button for tasks -->
              <NButton
                v-if="filteredTasks.length > 0"
                quaternary
                round
                size="small"
                class="!px-2"
                @click.stop="showTasks = !showTasks"
              >
                <template #icon>
                  <ChevronDownIcon v-if="!showTasks" class="text-gray-500" />
                  <ChevronUpIcon v-else class="text-gray-500" />
                </template>
              </NButton>
            </div>
            <!-- Tasks -->
            <div
              v-if="filteredTasks.length > 0 && showTasks"
              class="mt-2 flex flex-row gap-2 flex-wrap"
            >
              <NTag
                v-for="task in displayedTasks"
                :key="task.name"
                round
                size="small"
                :bordered="false"
                @click.stop="handleTaskClick(task)"
              >
                <template #avatar>
                  <TaskStatus :status="task.status" size="tiny" disabled />
                </template>
                <div class="flex items-center flex-nowrap gap-2">
                  <DatabaseDisplay :database="task.target" />
                  <template v-if="task.runTime">
                    <span class="font-mono text-control-light opacity-80"
                      >/</span
                    >
                    <NTooltip>
                      <template #trigger>
                        <div class="flex items-center gap-1">
                          <CalendarClockIcon
                            :size="14"
                            class="text-control-light"
                          />
                          <span class="text-control">{{
                            humanizeTs(
                              getTimeForPbTimestampProtoEs(task.runTime, 0) /
                                1000
                            )
                          }}</span>
                        </div>
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
                  </template>
                </div>
              </NTag>
              <NTag
                v-if="remainingTaskCount > 0"
                round
                size="small"
                type="default"
                :bordered="false"
                class="opacity-80"
                :class="isCreated && 'cursor-pointer hover:opacity-100'"
                @click.stop="handleClickStageTitle()"
              >
                +{{ remainingTaskCount }} more
              </NTag>
            </div>
          </div>
        </div>

        <!-- Right side: Actions -->
        <div v-if="!readonly" class="flex justify-end items-center">
          <RunTasksButton
            v-if="isCreated"
            :stage="stage"
            :size="'small'"
            :disabled="!canRunTasks || runableTasks.length === 0"
            @run-tasks="handleRunAllTasks"
          />
          <NPopconfirm
            v-else-if="!isCreated && canCreateRollout"
            :negative-text="null"
            :positive-text="$t('common.confirm')"
            @positive-click="createRolloutToStage"
          >
            <template #trigger>
              <NTooltip>
                <template #trigger>
                  <NButton :size="'small'">
                    <template #icon>
                      <CircleFadingPlusIcon class="w-5 h-5" />
                    </template>
                    {{ $t("common.start") }}
                  </NButton>
                </template>
                {{ $t("rollout.stage.start-stage") }}
              </NTooltip>
            </template>
            {{ $t("common.confirm-and-add") }}
          </NPopconfirm>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Timestamp as TimestampPb } from "@bufbuild/protobuf/wkt";
import dayjs from "dayjs";
import {
  CircleFadingPlusIcon,
  ChevronDownIcon,
  ChevronUpIcon,
  CalendarClockIcon,
} from "lucide-vue-next";
import { NTooltip, NButton, NPopconfirm, NTag } from "naive-ui";
import { twMerge } from "tailwind-merge";
import { computed, ref, watch } from "vue";
import { useRouter } from "vue-router";
import DatabaseDisplay from "@/components/Plan/components/common/DatabaseDisplay.vue";
import { TASK_STATUS_FILTERS } from "@/components/Plan/constants/task";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import Timestamp from "@/components/misc/Timestamp.vue";
import { EnvironmentV1Name } from "@/components/v2";
import {
  PROJECT_V1_ROUTE_ROLLOUT_DETAIL_STAGE_DETAIL,
  PROJECT_V1_ROUTE_ROLLOUT_DETAIL_TASK_DETAIL,
} from "@/router/dashboard/projectV1";
import {
  useCurrentProjectV1,
  useEnvironmentV1Store,
  batchGetOrFetchDatabases,
} from "@/store";
import { getTimeForPbTimestampProtoEs } from "@/types";
import type {
  Stage,
  Task,
  Rollout,
} from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractProjectResourceName,
  stringifyTaskStatus,
  getStageStatus,
  humanizeTs,
} from "@/utils";
import RunTasksButton from "./RunTasksButton.vue";
import { useTaskActionPermissions } from "./taskPermissions";

const props = defineProps<{
  rollout: Rollout;
  stage: Stage;
  taskStatusFilter?: Task_Status[];
  readonly?: boolean;
  defaultShowTasks?: boolean;
}>();

const router = useRouter();
const { project } = useCurrentProjectV1();
const environmentStore = useEnvironmentV1Store();
const emit = defineEmits<{
  (event: "run-tasks", stage: Stage, tasks: Task[]): void;
  (event: "create-rollout-to-stage", stage: Stage): void;
}>();

const { canPerformTaskAction } = useTaskActionPermissions();

const isCreated = computed(() => {
  return props.rollout.stages.some(
    (stage) => stage.environment === props.stage.environment
  );
});

// Toggle state for showing/hiding tasks - default based on parent coordination
const showTasks = ref(props.defaultShowTasks || false);

const filteredTasks = computed(() => {
  let tasks = props.stage.tasks;

  // Apply status filter if provided
  if (props.taskStatusFilter && props.taskStatusFilter.length > 0) {
    tasks = tasks.filter((task) =>
      props.taskStatusFilter!.includes(task.status)
    );
  }

  // Sort tasks by status order defined in TASK_STATUS_FILTERS
  const statusOrder = new Map<Task_Status, number>();
  TASK_STATUS_FILTERS.forEach((status, index) => {
    statusOrder.set(status, index);
  });

  return tasks.slice().sort((a, b) => {
    const aOrder = statusOrder.get(a.status) ?? Number.MAX_SAFE_INTEGER;
    const bOrder = statusOrder.get(b.status) ?? Number.MAX_SAFE_INTEGER;
    return aOrder - bOrder;
  });
});

// The maximum number of tasks to display before showing "more".
const MAX_DISPLAYED_TASKS = 10;

const displayedTasks = computed(() => {
  return filteredTasks.value.slice(0, MAX_DISPLAYED_TASKS);
});

const remainingTaskCount = computed(() => {
  return Math.max(0, filteredTasks.value.length - MAX_DISPLAYED_TASKS);
});

const runableTasks = computed(() => {
  return filteredTasks.value.filter(
    (task) =>
      task.status === Task_Status.NOT_STARTED ||
      task.status === Task_Status.PENDING ||
      task.status === Task_Status.FAILED ||
      task.status === Task_Status.CANCELED
  );
});

const canRunTasks = computed(() => {
  return canPerformTaskAction(
    filteredTasks.value,
    props.rollout,
    project.value
  );
});

const canCreateRollout = computed(() => {
  return canRunTasks.value;
});

const stageStatus = computed(() => {
  // Create a temporary stage object with filtered tasks for status calculation
  const stageWithFilteredTasks = {
    ...props.stage,
    tasks: filteredTasks.value,
  };
  return getStageStatus(stageWithFilteredTasks);
});

const latestUpdateTimestamp = computed(() => {
  let latestTimestamp: TimestampPb | undefined;

  for (const task of props.stage.tasks) {
    if (task.updateTime) {
      if (
        !latestTimestamp ||
        task.updateTime.seconds > latestTimestamp.seconds
      ) {
        latestTimestamp = task.updateTime;
      }
    }
  }

  return latestTimestamp;
});

const formatFullDateTime = (timestamp: TimestampPb) => {
  const timestampInMilliseconds = getTimeForPbTimestampProtoEs(timestamp, 0);
  return dayjs(timestampInMilliseconds).local().format();
};

const getTaskCount = (status: Task_Status) => {
  return filteredTasks.value.filter((task) => task.status === status).length;
};

const handleRunAllTasks = () => {
  emit("run-tasks", props.stage, runableTasks.value);
};

const createRolloutToStage = () => {
  emit("create-rollout-to-stage", props.stage);
};

const handleClickStageTitle = () => {
  // Only navigate if the stage is created
  if (!isCreated.value) return;

  const rolloutId = props.rollout.name.split("/").pop();
  const stageId = props.stage.name.split("/").pop();

  if (!rolloutId || !stageId) return;

  // Navigate to the stage detail route
  router.push({
    name: PROJECT_V1_ROUTE_ROLLOUT_DETAIL_STAGE_DETAIL,
    params: {
      projectId: extractProjectResourceName(project.value.name),
      rolloutId,
      stageId,
    },
  });
};

const handleTaskStatusClick = (status: Task_Status) => {
  // Only navigate if the stage is created
  if (!isCreated.value) return;

  const rolloutId = props.rollout.name.split("/").pop();
  const stageId = props.stage.name.split("/").pop();

  if (!rolloutId || !stageId) return;

  // Navigate to the stage detail route with task status filter
  router.push({
    name: PROJECT_V1_ROUTE_ROLLOUT_DETAIL_STAGE_DETAIL,
    params: {
      projectId: extractProjectResourceName(project.value.name),
      rolloutId,
      stageId,
    },
    query: {
      taskStatus: Task_Status[status],
    },
  });
};

const handleTaskClick = (task: Task) => {
  // Only navigate if the stage is created
  if (!isCreated.value) return;

  const rolloutId = props.rollout.name.split("/").pop();
  const stageId = props.stage.name.split("/").pop();
  const taskId = task.name.split("/").pop();

  if (!rolloutId || !stageId || !taskId) return;

  // Navigate to the task detail route
  router.push({
    name: PROJECT_V1_ROUTE_ROLLOUT_DETAIL_TASK_DETAIL,
    params: {
      projectId: extractProjectResourceName(project.value.name),
      rolloutId,
      stageId,
      taskId,
    },
  });
};

const prepareDatabases = async () => {
  // Only load data when tasks are shown
  if (!showTasks.value) return;

  if (displayedTasks.value.length > 0) {
    try {
      await batchGetOrFetchDatabases(
        displayedTasks.value.map((task) => task.target)
      );
    } catch {
      // Ignore errors - this is just for pre-loading data
    }
  }
};

watch(
  showTasks,
  () => {
    prepareDatabases();
  },
  { immediate: true }
);
</script>
