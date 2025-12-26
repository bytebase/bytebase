<template>
  <div
    class="task-toolbar sticky top-0 z-10 px-4"
  >
    <div class="flex items-center justify-between border px-3 py-1 rounded-lg border-blue-200 bg-blue-100">
      <!-- Left section -->
      <div class="flex items-center gap-x-3">
        <!-- Select all checkbox -->
        <NTooltip :disabled="!isSelectAllDisabled || !getSelectAllTooltip()">
          <template #trigger>
            <NCheckbox
              v-if="allTasks.length > 0"
              :checked="allSelectableTasksSelected"
              :indeterminate="someTasksSelected && !allSelectableTasksSelected"
              :disabled="isSelectAllDisabled"
              @update:checked="handleSelectAllChange"
            />
          </template>
          <span class="w-56 text-sm">{{ getSelectAllTooltip() }}</span>
        </NTooltip>

        <!-- Selection state -->
          <span class="text-sm text-blue-900">
            {{ selectionCountText }}
          </span>

          <!-- Bulk actions (always shown) -->
          <div class="flex items-center ">

          <NTooltip :disabled="!isRunDisabled || !getRunTooltip()">
            <template #trigger>
              <NButton
                quaternary
                  size="small"
                  type="primary"
                :disabled="isRunDisabled"
                @click="handleAction('RUN')"
              >
              <template #icon>
                <PlayIcon :size="14" />
              </template>
                {{ $t("common.run") }}
              </NButton>
            </template>
            <span class="w-56 text-sm">{{ getRunTooltip() }}</span>
          </NTooltip>
  
          <NTooltip :disabled="!isSkipDisabled || !getSkipTooltip()">
            <template #trigger>
              <NButton
                quaternary
                  size="small"
                  type="primary"
                :disabled="isSkipDisabled"
                @click="handleAction('SKIP')"
              >
              <template #icon>
                <SkipForwardIcon :size="14" />
              </template>
                {{ $t("common.skip") }}
              </NButton>
            </template>
            <span class="w-56 text-sm">{{ getSkipTooltip() }}</span>
          </NTooltip>
  
          <NTooltip :disabled="!isCancelDisabled || !getCancelTooltip()">
            <template #trigger>
              <NButton
                quaternary
                  size="small"
                  type="primary"
                :disabled="isCancelDisabled"
                @click="handleAction('CANCEL')"
              >
              <template #icon>
                <XIcon :size="14" />
              </template>
                {{ $t("common.cancel") }}
              </NButton>
            </template>
            <span class="w-56 text-sm">{{ getCancelTooltip() }}</span>
          </NTooltip>

          </div>
      </div>

    </div>

    <!-- Action confirmation panel -->
    <TaskRolloutActionPanel
      v-if="showActionPanel && pendingAction"
      :show="showActionPanel"
      :action="pendingAction"
      :target="{ type: 'tasks', tasks: selectedTasks, stage }"
      @close="showActionPanel = false"
      @confirm="handleActionPerformed"
    />
  </div>
</template>

<script lang="ts" setup>
import { PlayIcon, SkipForwardIcon, XIcon } from "lucide-vue-next";
import { NButton, NCheckbox, NTooltip } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { usePlanContext } from "@/components/Plan/logic";
import { useCurrentUserV1 } from "@/store";
import { userNamePrefix } from "@/store/modules/v1/common";
import { Issue_Type } from "@/types/proto-es/v1/issue_service_pb";
import type { Stage, Task } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import TaskRolloutActionPanel from "./TaskRolloutActionPanel.vue";
import { canRolloutTasks } from "./taskPermissions";
import type { TaskAction } from "./types";

const props = defineProps<{
  selectedTasks: Task[];
  allTasks: Task[];
  isTaskSelectable: (task: Task) => boolean;
  stage: Stage;
}>();

const emit = defineEmits<{
  (event: "select-all"): void;
  (event: "clear-selection"): void;
  (event: "action-complete"): void;
}>();

const { t } = useI18n();
const { issue } = usePlanContext();
const currentUser = useCurrentUserV1();

const showActionPanel = ref(false);
const pendingAction = ref<TaskAction | null>(null);

const selectableTasks = computed(() => {
  return props.allTasks.filter(props.isTaskSelectable);
});

const allSelectableTasksSelected = computed(() => {
  if (selectableTasks.value.length === 0) {
    return false;
  }
  return (
    props.selectedTasks.length === selectableTasks.value.length &&
    selectableTasks.value.every((task) =>
      props.selectedTasks.some((st) => st.name === task.name)
    )
  );
});

const someTasksSelected = computed(() => {
  return props.selectedTasks.length > 0 && !allSelectableTasksSelected.value;
});

const selectionCountText = computed(() => {
  return t("rollout.task.selected-count", {
    count: props.selectedTasks.length,
  });
});

const isSelectAllDisabled = computed(() => {
  return selectableTasks.value.length === 0;
});

const getSelectAllTooltip = () => {
  if (selectableTasks.value.length === 0) {
    return t("task.no-selectable-tasks");
  }
  return "";
};

const canPerformTaskActions = computed(() => {
  return canRolloutTasks(props.selectedTasks, issue.value);
});

const hasRunnableTasks = computed(() => {
  return props.selectedTasks.some(
    (task) =>
      task.status === Task_Status.NOT_STARTED ||
      task.status === Task_Status.FAILED ||
      task.status === Task_Status.CANCELED
  );
});

const hasSkippableTasks = computed(() => {
  return props.selectedTasks.some(
    (task) => task.status === Task_Status.NOT_STARTED
  );
});

const hasCancellableTasks = computed(() => {
  return props.selectedTasks.some(
    (task) =>
      task.status === Task_Status.PENDING || task.status === Task_Status.RUNNING
  );
});

/**
 * Get base disabled tooltip for common conditions (no selection, no permission)
 */
const getBaseDisabledTooltip = (): string => {
  if (props.selectedTasks.length === 0) {
    return t("task.no-tasks-selected");
  }
  if (!canPerformTaskActions.value) {
    // Special message for data export issues when user is not the creator
    if (
      issue.value &&
      issue.value.type === Issue_Type.DATABASE_EXPORT &&
      issue.value.creator !== `${userNamePrefix}${currentUser.value.email}`
    ) {
      return t("task.data-export-creator-only");
    }
    return t("task.no-permission");
  }
  return "";
};

/**
 * Get action-specific tooltip by combining base tooltip with action requirements
 */
const getActionTooltip = (
  hasValidTasks: boolean,
  noValidTasksMessage: string
): string => {
  const baseTooltip = getBaseDisabledTooltip();
  if (baseTooltip) {
    return baseTooltip;
  }
  if (!hasValidTasks) {
    return noValidTasksMessage;
  }
  return "";
};

const getRunTooltip = () =>
  getActionTooltip(hasRunnableTasks.value, t("task.no-runnable-tasks"));

const getSkipTooltip = () =>
  getActionTooltip(hasSkippableTasks.value, t("task.no-skippable-tasks"));

const getCancelTooltip = () =>
  getActionTooltip(hasCancellableTasks.value, t("task.no-cancellable-tasks"));

const isRunDisabled = computed(() => {
  return (
    props.selectedTasks.length === 0 ||
    !hasRunnableTasks.value ||
    !canPerformTaskActions.value
  );
});

const isSkipDisabled = computed(() => {
  return (
    props.selectedTasks.length === 0 ||
    !hasSkippableTasks.value ||
    !canPerformTaskActions.value
  );
});

const isCancelDisabled = computed(() => {
  return (
    props.selectedTasks.length === 0 ||
    !hasCancellableTasks.value ||
    !canPerformTaskActions.value
  );
});

const handleSelectAllChange = (checked: boolean) => {
  if (checked) {
    emit("select-all");
  } else {
    emit("clear-selection");
  }
};

const handleAction = (action: TaskAction) => {
  pendingAction.value = action;
  showActionPanel.value = true;
};

const handleActionPerformed = () => {
  showActionPanel.value = false;
  pendingAction.value = null;
  emit("action-complete");
};
</script>
