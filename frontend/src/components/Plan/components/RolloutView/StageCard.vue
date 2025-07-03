<template>
  <div
    class="!w-80 bg-white z-[1] rounded-lg p-1"
    :class="
      twMerge(
        isCreated ? 'bg-white shadow' : 'bg-zinc-50 border-2 border-dashed'
      )
    "
  >
    <div
      class="w-full flex flex-row justify-between items-center gap-2 px-2 pt-2 pb-1"
    >
      <div class="flex items-center space-x-1">
        <span class="text-base font-medium">
          {{ environmentStore.getEnvironmentByName(stage.environment).title }}
        </span>
        <NTag v-if="!isCreated" round size="tiny">{{
          $t("common.preview")
        }}</NTag>
      </div>
      <div class="flex justify-end items-center">
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
          :positive-button-props="{ size: 'tiny' }"
          @positive-click="createRolloutToStage"
        >
          <template #trigger>
            <NTooltip>
              <template #trigger>
                <NButton text size="small">
                  <template #icon>
                    <CircleFadingPlusIcon class="w-4 h-4" />
                  </template>
                </NButton>
              </template>
              {{ $t("common.create") }}
            </NTooltip>
          </template>
          {{ $t("common.confirm-and-add") }}
        </NPopconfirm>
      </div>
    </div>
    <NVirtualList
      v-if="filteredTasks.length > 0"
      style="max-height: 80vh"
      :items="filteredTasks"
      :item-size="40"
      item-resizable
    >
      <template #default="{ item: task }: { item: Task }">
        <div
          :key="task.name"
          class="w-full border-t border-zinc-50 flex items-center justify-start truncate px-2 py-2 h-10 cursor-pointer hover:bg-zinc-50 transition-colors"
          @click="handleTaskClick(task)"
        >
          <TaskStatus :status="task.status" size="small" class="shrink-0" />
          <TaskDatabaseName :task="task" class="ml-2 flex-1" />
          <div class="ml-auto flex items-center space-x-1 shrink-0">
            <NTag round size="tiny">{{ semanticTaskType(task.type) }}</NTag>

            <NTooltip v-if="extractSchemaVersionFromTask(task)">
              <template #trigger>
                <NTag round size="tiny">
                  {{ extractSchemaVersionFromTask(task) }}
                </NTag>
              </template>
              {{ $t("common.version") }}
            </NTooltip>
          </div>
        </div>
      </template>
    </NVirtualList>
    <div v-else class="text-center text-zinc-500 py-2 text-sm leading-6">
      {{ $t("task.no-tasks") }}
    </div>

    <!-- Task Rollout Action Panel -->
    <TaskRolloutActionPanel
      :show="showRunTasksPanel"
      action="RUN"
      :target="{ type: 'tasks', stage, tasks: runableTasks }"
      @close="handlePanelClose"
    />
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { CircleFadingPlusIcon } from "lucide-vue-next";
import { NTag, NTooltip, NVirtualList, NButton, NPopconfirm } from "naive-ui";
import { twMerge } from "tailwind-merge";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { semanticTaskType } from "@/components/IssueV1";
import TaskStatus from "@/components/Rollout/kits/TaskStatus.vue";
import { rolloutServiceClientConnect } from "@/grpcweb";
import { PROJECT_V1_ROUTE_ROLLOUT_DETAIL_TASK_DETAIL } from "@/router/dashboard/projectV1";
import {
  useCurrentProjectV1,
  useEnvironmentV1Store,
  pushNotification,
} from "@/store";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import type { Stage, Task } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { extractProjectResourceName } from "@/utils";
import { extractSchemaVersionFromTask } from "@/utils";
import { usePlanContextWithRollout } from "../../logic";
import RunTasksButton from "./RunTasksButton.vue";
import TaskDatabaseName from "./TaskDatabaseName.vue";
import TaskRolloutActionPanel from "./TaskRolloutActionPanel.vue";
import { useRolloutViewContext } from "./context";
import { useTaskActionPermissions } from "./taskPermissions";

const props = defineProps<{
  stage: Stage;
  taskStatusFilter: Task_Status[];
}>();

const { t: $t } = useI18n();
const router = useRouter();
const { project } = useCurrentProjectV1();
const environmentStore = useEnvironmentV1Store();
const { events } = usePlanContextWithRollout();
const { rollout } = useRolloutViewContext();
const { canPerformTaskAction } = useTaskActionPermissions();

const showRunTasksPanel = ref(false);

const isCreated = computed(() => {
  return rollout.value.stages.some(
    (stage) => stage.environment === props.stage.environment
  );
});

const filteredTasks = computed(() => {
  if (props.taskStatusFilter.length === 0) {
    return props.stage.tasks;
  }
  return props.stage.tasks.filter((task) =>
    props.taskStatusFilter.includes(task.status)
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
    rollout.value,
    project.value
  );
});

const canCreateRollout = computed(() => {
  return canRunTasks.value;
});

// Helper function to extract IDs from task and stage names
const getTaskRouteParams = (task: Task) => {
  const rolloutId = rollout.value.name.split("/").pop();
  const stageId = props.stage.name.split("/").pop();
  const taskId = task.name.split("/").pop();

  return { rolloutId, stageId, taskId };
};

// Task click handler
const handleTaskClick = (task: Task) => {
  const params = getTaskRouteParams(task);
  if (params.rolloutId && params.stageId && params.taskId) {
    router.push({
      name: PROJECT_V1_ROUTE_ROLLOUT_DETAIL_TASK_DETAIL,
      params: {
        projectId: extractProjectResourceName(project.value.name),
        rolloutId: params.rolloutId,
        stageId: params.stageId,
        taskId: params.taskId,
      },
    });
  }
};

const handleRunAllTasks = () => {
  showRunTasksPanel.value = true;
};

const handlePanelClose = () => {
  showRunTasksPanel.value = false;
};

const createRolloutToStage = async () => {
  try {
    const request = create(CreateRolloutRequestSchema, {
      parent: project.value.name,
      rollout: {
        plan: rollout.value.plan,
      },
      target: props.stage.environment,
    });
    await rolloutServiceClientConnect.createRollout(request);

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: $t("common.success"),
      description: $t("common.created"),
    });

    // Trigger immediate refresh of rollout data
    events.emit("status-changed", { eager: true });
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: $t("common.error"),
      description: String(error),
    });
  }
};
</script>
