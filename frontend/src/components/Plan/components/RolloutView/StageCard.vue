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
          <div class="flex items-center gap-x-3 gap-y-2 flex-wrap">
            <NTag v-if="taskCounts.done > 0" round size="medium" type="default">
              <template #avatar>
                <TaskStatus :status="Task_Status.DONE" size="small" />
              </template>
              <div class="flex flex-row items-center gap-2">
                <span class="select-none text-base">{{
                  stringifyTaskStatus(Task_Status.DONE)
                }}</span>
                <span class="select-none text-base font-medium">{{
                  taskCounts.done
                }}</span>
              </div>
            </NTag>
            <NTag
              v-if="taskCounts.notStarted > 0"
              round
              size="medium"
              type="default"
            >
              <template #avatar>
                <TaskStatus :status="Task_Status.NOT_STARTED" size="small" />
              </template>
              <div class="flex flex-row items-center gap-2">
                <span class="select-none text-base">{{
                  stringifyTaskStatus(Task_Status.NOT_STARTED)
                }}</span>
                <span class="select-none text-base font-medium">{{
                  taskCounts.notStarted
                }}</span>
              </div>
            </NTag>
            <NTag v-if="taskCounts.running > 0" round size="medium" type="info">
              <template #avatar>
                <TaskStatus :status="Task_Status.RUNNING" size="small" />
              </template>
              <div class="flex flex-row items-center gap-2">
                <span class="select-none text-base">{{
                  stringifyTaskStatus(Task_Status.RUNNING)
                }}</span>
                <span class="select-none text-base font-medium">{{
                  taskCounts.running
                }}</span>
              </div>
            </NTag>
            <NTag v-if="taskCounts.failed > 0" round size="medium" type="error">
              <template #avatar>
                <TaskStatus :status="Task_Status.FAILED" size="small" />
              </template>
              <div class="flex flex-row items-center gap-2">
                <span class="select-none text-base">{{
                  stringifyTaskStatus(Task_Status.FAILED)
                }}</span>
                <span class="select-none text-base font-medium">{{
                  taskCounts.failed
                }}</span>
              </div>
            </NTag>
            <NTag
              v-if="taskCounts.canceled > 0"
              round
              size="medium"
              type="default"
            >
              <template #avatar>
                <TaskStatus :status="Task_Status.CANCELED" size="small" />
              </template>
              <div class="flex flex-row items-center gap-2">
                <span class="select-none text-base">{{
                  stringifyTaskStatus(Task_Status.CANCELED)
                }}</span>
                <span class="select-none text-base font-medium">{{
                  taskCounts.canceled
                }}</span>
              </div>
            </NTag>
            <NTag
              v-if="taskCounts.pending > 0"
              round
              size="medium"
              type="warning"
            >
              <template #avatar>
                <TaskStatus :status="Task_Status.PENDING" size="small" />
              </template>
              <div class="flex flex-row items-center gap-2">
                <span class="select-none text-base">{{
                  stringifyTaskStatus(Task_Status.PENDING)
                }}</span>
                <span class="select-none text-base font-medium">{{
                  taskCounts.pending
                }}</span>
              </div>
            </NTag>
            <NTag
              v-if="taskCounts.skipped > 0"
              round
              size="medium"
              type="default"
            >
              <template #avatar>
                <TaskStatus :status="Task_Status.SKIPPED" size="small" />
              </template>
              <div class="flex flex-row items-center gap-2">
                <span class="select-none text-base">{{
                  stringifyTaskStatus(Task_Status.SKIPPED)
                }}</span>
                <span class="select-none text-base font-medium">{{
                  taskCounts.skipped
                }}</span>
              </div>
            </NTag>
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

const taskCounts = computed(() => {
  const counts = {
    done: 0,
    notStarted: 0,
    running: 0,
    failed: 0,
    canceled: 0,
    pending: 0,
    skipped: 0,
  };

  filteredTasks.value.forEach((task) => {
    switch (task.status) {
      case Task_Status.DONE:
        counts.done++;
        break;
      case Task_Status.NOT_STARTED:
        counts.notStarted++;
        break;
      case Task_Status.RUNNING:
        counts.running++;
        break;
      case Task_Status.FAILED:
        counts.failed++;
        break;
      case Task_Status.CANCELED:
        counts.canceled++;
        break;
      case Task_Status.PENDING:
        counts.pending++;
        break;
      case Task_Status.SKIPPED:
        counts.skipped++;
        break;
    }
  });

  return counts;
});

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
</script>
