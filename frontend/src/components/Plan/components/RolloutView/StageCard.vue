<template>
  <div class="relative w-full flex items-center gap-4">
    <!-- Stage card -->
    <div
      class="flex-1 bg-white rounded-lg border p-4 shadow-sm"
      :class="
        twMerge(
          isCreated ? 'border-gray-200' : 'border-gray-300 border-dashed',
          !readonly && 'cursor-pointer hover:shadow-md transition-shadow'
        )
      "
      @click="!readonly && isCreated && handleClickStageTitle()"
    >
      <div class="flex items-center justify-between gap-4">
        <!-- Left side: Stage title and status counts -->
        <div class="flex items-center gap-2">
          <div class="flex items-center gap-2">
            <TaskStatus :status="stageStatus" size="medium" />
            <h3
              class="text-base font-medium text-gray-900 whitespace-nowrap w-24 truncate"
            >
              {{
                environmentStore.getEnvironmentByName(stage.environment).title
              }}
            </h3>
          </div>

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
                      : status === Task_Status.PENDING
                        ? 'warning'
                        : 'default'
                "
                class="cursor-pointer hover:opacity-80 transition-opacity"
                @click.stop="handleTaskStatusClick(status)"
              >
                <template #avatar>
                  <TaskStatus :status="status" size="small" />
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
          </div>
        </div>

        <!-- Right side: Actions -->
        <div v-if="!readonly" class="flex justify-end items-center">
          <RunTasksButton
            v-if="isCreated"
            :stage="stage"
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
                  <NButton size="medium">
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
import { CircleFadingPlusIcon } from "lucide-vue-next";
import { NTooltip, NButton, NPopconfirm, NTag } from "naive-ui";
import { twMerge } from "tailwind-merge";
import { computed } from "vue";
import { useRouter } from "vue-router";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import { PROJECT_V1_ROUTE_ROLLOUT_DETAIL_STAGE_DETAIL } from "@/router/dashboard/projectV1";
import { useCurrentProjectV1, useEnvironmentV1Store } from "@/store";
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
} from "@/utils";
import RunTasksButton from "./RunTasksButton.vue";
import { useTaskActionPermissions } from "./taskPermissions";

const props = defineProps<{
  rollout: Rollout;
  stage: Stage;
  taskStatusFilter?: Task_Status[];
  readonly?: boolean;
}>();

const router = useRouter();
const { project } = useCurrentProjectV1();
const environmentStore = useEnvironmentV1Store();
const emit = defineEmits<{
  (event: "run-tasks", stage: Stage, tasks: Task[]): void;
  (event: "create-rollout-to-stage", stage: Stage): void;
}>();

const { canPerformTaskAction } = useTaskActionPermissions();

const TASK_STATUS_FILTERS: Task_Status[] = [
  Task_Status.DONE,
  Task_Status.RUNNING,
  Task_Status.PENDING,
  Task_Status.FAILED,
  Task_Status.CANCELED,
  Task_Status.NOT_STARTED,
  Task_Status.SKIPPED,
];

const isCreated = computed(() => {
  return props.rollout.stages.some(
    (stage) => stage.environment === props.stage.environment
  );
});

const filteredTasks = computed(() => {
  if (!props.taskStatusFilter || props.taskStatusFilter.length === 0) {
    return props.stage.tasks;
  }
  return props.stage.tasks.filter((task) =>
    props.taskStatusFilter!.includes(task.status)
  );
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
</script>
