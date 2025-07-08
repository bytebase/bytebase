<template>
  <NScrollbar x-scrollable>
    <div
      v-bind="$attrs"
      class="text-sm flex flex-col lg:flex-row items-start lg:items-center bg-blue-100 py-3 px-4 text-main gap-y-2 gap-x-4"
    >
      <span class="whitespace-nowrap">{{
        tasks.length > 0
          ? $t("task.selected-n-tasks", { n: tasks.length })
          : $t("task.no-tasks-selected")
      }}</span>
      <div class="flex items-center">
        <template v-for="action in actions" :key="action.text">
          <NTooltip
            :disabled="!action.disabled || !action.tooltip(action.text)"
          >
            <template #trigger>
              <NButton
                quaternary
                size="small"
                type="primary"
                :disabled="action.disabled"
                @click="action.click"
              >
                <template #icon>
                  <component :is="action.icon" class="h-4 w-4" />
                </template>
                <span class="text-sm">{{ action.text }}</span>
              </NButton>
            </template>
            <span class="w-56 text-sm">
              {{ action.tooltip(action.text.toLowerCase()) }}
            </span>
          </NTooltip>
        </template>
      </div>
    </div>
  </NScrollbar>

  <!-- Task Rollout Action Panel -->
  <template v-if="state.selectedAction && actionTarget">
    <TaskRolloutActionPanel
      :show="state.showActionPanel"
      :action="state.selectedAction"
      :target="actionTarget"
      @close="handleActionPanelClose"
    />
  </template>
</template>

<script lang="ts" setup>
import { PlayIcon, XIcon, SkipForwardIcon } from "lucide-vue-next";
import { NButton, NScrollbar, NTooltip } from "naive-ui";
import type { VNode } from "vue";
import { computed, h, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useCurrentProjectV1 } from "@/store";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import type {
  Task,
  Rollout,
  Stage,
} from "@/types/proto-es/v1/rollout_service_pb";
import { usePlanContext } from "../../logic";
import TaskRolloutActionPanel from "./TaskRolloutActionPanel.vue";
import { useTaskActionPermissions } from "./taskPermissions";

interface TaskAction {
  icon: VNode;
  text: string;
  disabled: boolean;
  click: () => void;
  tooltip: (action: string) => string;
}

interface LocalState {
  showActionPanel: boolean;
  selectedAction?: "RUN" | "SKIP" | "CANCEL";
}

const props = defineProps<{
  tasks: Task[];
  rollout: Rollout;
}>();

const emit = defineEmits<{
  (event: "refresh"): void;
  (event: "task-action-completed"): void;
}>();

const state = reactive<LocalState>({
  showActionPanel: false,
  selectedAction: undefined,
});

const { t } = useI18n();
const { project } = useCurrentProjectV1();
const planContext = usePlanContext();
const { canPerformTaskAction } = useTaskActionPermissions();

// Get stages for selected tasks
const selectedTaskStages = computed(() => {
  const stageMap = new Map<string, Stage>();
  props.rollout.stages.forEach((stage) => {
    stage.tasks.forEach((task) => {
      if (props.tasks.some((selectedTask) => selectedTask.name === task.name)) {
        stageMap.set(stage.name, stage);
      }
    });
  });
  return Array.from(stageMap.values());
});

// Check if selected tasks are from multiple stages (cross-stage)
const isCrossStage = computed(() => {
  return selectedTaskStages.value.length > 1;
});

// Permission checks for each action
const canRunTasks = computed(() => {
  if (props.tasks.length === 0) return false;
  return canPerformTaskAction(
    props.tasks,
    props.rollout,
    project.value,
    planContext.issue?.value
  );
});

const canSkipTasks = computed(() => {
  if (props.tasks.length === 0) return false;
  return canPerformTaskAction(
    props.tasks,
    props.rollout,
    project.value,
    planContext.issue?.value
  );
});

const canCancelTasks = computed(() => {
  if (props.tasks.length === 0) return false;
  return canPerformTaskAction(
    props.tasks,
    props.rollout,
    project.value,
    planContext.issue?.value
  );
});

// Action target for the panel
const actionTarget = computed(() => {
  if (!state.selectedAction || selectedTaskStages.value.length === 0) {
    return undefined;
  }

  if (state.selectedAction === "CANCEL") {
    // For cancel actions, we would need task runs, but for now we'll use tasks
    return {
      type: "tasks" as const,
      tasks: props.tasks,
      stage: selectedTaskStages.value[0], // Use first stage for simplicity
    };
  } else {
    return {
      type: "tasks" as const,
      tasks: props.tasks,
      stage: selectedTaskStages.value[0], // Use first stage for simplicity
    };
  }
});

// Check if tasks have the right status for each action
const hasRunnableTasks = computed(() => {
  return props.tasks.some((task) =>
    [Task_Status.NOT_STARTED, Task_Status.FAILED].includes(task.status)
  );
});

const hasCancellableTasks = computed(() => {
  return props.tasks.some((task) =>
    [Task_Status.PENDING, Task_Status.RUNNING].includes(task.status)
  );
});

const hasSkippableTasks = computed(() => {
  return props.tasks.some((task) =>
    [
      Task_Status.NOT_STARTED,
      Task_Status.FAILED,
      Task_Status.CANCELED,
    ].includes(task.status)
  );
});

const getDisabledTooltip = (_action: string) => {
  if (props.tasks.length === 0) {
    return t("task.no-tasks-selected");
  }
  if (isCrossStage.value) {
    return t("task.cross-stage-not-supported");
  }
  return "";
};

const actions = computed((): TaskAction[] => {
  const resp: TaskAction[] = [];

  // Run action - always show
  resp.push({
    icon: h(PlayIcon),
    text: t("common.run"),
    disabled:
      props.tasks.length === 0 ||
      isCrossStage.value ||
      !canRunTasks.value ||
      !hasRunnableTasks.value,
    click: () => {
      state.selectedAction = "RUN";
      state.showActionPanel = true;
    },
    tooltip: (action) => {
      if (props.tasks.length === 0) {
        return getDisabledTooltip(action);
      }
      if (isCrossStage.value) {
        return getDisabledTooltip(action);
      }
      if (!canRunTasks.value) {
        return t("task.no-permission");
      }
      if (!hasRunnableTasks.value) {
        return t("task.no-runnable-tasks");
      }
      return "";
    },
  });

  // Skip action - always show
  resp.push({
    icon: h(SkipForwardIcon),
    text: t("common.skip"),
    disabled:
      props.tasks.length === 0 ||
      isCrossStage.value ||
      !canSkipTasks.value ||
      !hasSkippableTasks.value,
    click: () => {
      state.selectedAction = "SKIP";
      state.showActionPanel = true;
    },
    tooltip: (action) => {
      if (props.tasks.length === 0) {
        return getDisabledTooltip(action);
      }
      if (isCrossStage.value) {
        return getDisabledTooltip(action);
      }
      if (!canSkipTasks.value) {
        return t("task.no-permission");
      }
      if (!hasSkippableTasks.value) {
        return t("task.no-skippable-tasks");
      }
      return "";
    },
  });

  // Cancel action - always show
  resp.push({
    icon: h(XIcon),
    text: t("common.cancel"),
    disabled:
      props.tasks.length === 0 ||
      isCrossStage.value ||
      !canCancelTasks.value ||
      !hasCancellableTasks.value,
    click: () => {
      state.selectedAction = "CANCEL";
      state.showActionPanel = true;
    },
    tooltip: (action) => {
      if (props.tasks.length === 0) {
        return getDisabledTooltip(action);
      }
      if (isCrossStage.value) {
        return getDisabledTooltip(action);
      }
      if (!canCancelTasks.value) {
        return t("task.no-permission");
      }
      if (!hasCancellableTasks.value) {
        return t("task.no-cancellable-tasks");
      }
      return "";
    },
  });

  return resp;
});

const handleActionPanelClose = () => {
  state.showActionPanel = false;
  state.selectedAction = undefined;
  emit("task-action-completed");
  emit("refresh");
};
</script>
