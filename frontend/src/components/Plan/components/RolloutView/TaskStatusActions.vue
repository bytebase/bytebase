<template>
  <div>
    <div
      v-if="primaryAction || dropdownOptions.length > 0"
      class="flex flex-row justify-end items-center gap-x-2"
    >
      <NButton
        v-if="primaryAction"
        type="primary"
        size="small"
        @click="handlePrimaryAction"
      >
        {{ actionDisplayTitle(primaryAction) }}
      </NButton>
      <NDropdown
        v-if="dropdownOptions.length > 0"
        trigger="hover"
        :options="dropdownOptions"
        @select="handleDropdownSelect"
      >
        <NButton size="small">
          <template #icon>
            <EllipsisVerticalIcon class="w-4 h-4" />
          </template>
        </NButton>
      </NDropdown>
    </div>

    <!-- Task Rollout Action Panel -->
    <TaskRolloutActionPanel
      v-if="currentPanelAction && showActionPanel && actionTarget"
      :action="currentPanelAction"
      :target="actionTarget"
      @close="handleActionPanelClose"
    />
  </div>
</template>

<script setup lang="ts">
import { EllipsisVerticalIcon } from "lucide-vue-next";
import { NButton, NDropdown } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { Task_Status } from "@/types/proto/v1/rollout_service";
import type {
  Task,
  TaskRun,
  Rollout,
  Stage,
} from "@/types/proto/v1/rollout_service";
import TaskRolloutActionPanel from "./TaskRolloutActionPanel.vue";

type TaskStatusAction =
  // NOT_STARTED -> PENDING
  | "RUN"
  // FAILED -> PENDING
  | "RETRY"
  // * -> CANCELLED
  | "CANCEL"
  // * -> SKIPPED
  | "SKIP";

const props = defineProps<{
  task: Task;
  taskRuns: TaskRun[];
  rollout?: Rollout;
}>();

const emit = defineEmits<{
  "task-action-completed": [];
}>();

const { t } = useI18n();
const showActionPanel = ref(false);
const selectedAction = ref<TaskStatusAction>();

type PanelAction = "RUN" | "SKIP" | "CANCEL" | undefined;

const currentPanelAction = computed((): PanelAction => {
  if (!showActionPanel.value || !selectedAction.value) return undefined;

  switch (selectedAction.value) {
    case "RUN":
    case "RETRY":
      return "RUN";
    case "SKIP":
      return "SKIP";
    case "CANCEL":
      return "CANCEL";
    default:
      return undefined;
  }
});

// Determine target based on action type
const actionTarget = computed(() => {
  if (!currentPanelAction.value) return undefined;

  if (currentPanelAction.value === "CANCEL") {
    // For cancel actions, we need task runs
    return {
      type: "taskRuns" as const,
      taskRuns: props.taskRuns,
      stage: stage.value,
    };
  } else {
    // For run and skip actions, we use specific tasks
    return {
      type: "tasks" as const,
      tasks: [props.task],
      stage: stage.value,
    };
  }
});

// Get the stage for the current task
const stage = computed((): Stage => {
  // Find the actual stage if rollout is provided
  if (props.rollout) {
    for (const stage of props.rollout.stages) {
      for (const stageTask of stage.tasks) {
        if (stageTask.name === props.task.name) {
          return stage;
        }
      }
    }
  }

  // Should not reach here.
  return {
    id: "",
    name: "",
    environment: "",
    tasks: [],
  } as Stage;
});

const primaryAction = computed((): TaskStatusAction | null => {
  if (props.task.status === Task_Status.NOT_STARTED) {
    return "RUN";
  } else if (props.task.status === Task_Status.FAILED) {
    return "RETRY";
  } else {
    return null;
  }
});

const dropdownActions = computed((): TaskStatusAction[] => {
  if (
    [
      Task_Status.NOT_STARTED,
      Task_Status.FAILED,
      Task_Status.CANCELED,
    ].includes(props.task.status)
  ) {
    return ["SKIP"];
  } else if (
    [Task_Status.PENDING, Task_Status.RUNNING].includes(props.task.status)
  ) {
    return ["CANCEL"];
  } else {
    return [];
  }
});

const dropdownOptions = computed(() => {
  return dropdownActions.value.map((action) => {
    return {
      key: action,
      label: actionDisplayTitle(action),
    };
  });
});

const actionDisplayTitle = (action: TaskStatusAction) => {
  if (action === "RUN") {
    return t("common.run");
  } else if (action === "RETRY") {
    return t("common.retry");
  } else if (action === "CANCEL") {
    return t("common.cancel");
  } else if (action === "SKIP") {
    return t("common.skip");
  }
};

const handlePrimaryAction = () => {
  if (primaryAction.value) {
    selectedAction.value = primaryAction.value;
    showActionPanel.value = true;
  }
};

const handleDropdownSelect = (action: TaskStatusAction) => {
  selectedAction.value = action;
  showActionPanel.value = true;
};

const handleActionPanelClose = () => {
  showActionPanel.value = false;
  selectedAction.value = undefined;
  emit("task-action-completed");
};
</script>
