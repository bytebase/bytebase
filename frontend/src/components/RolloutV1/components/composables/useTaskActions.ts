import type { ComputedRef, Ref } from "vue";
import { computed, ref, watch } from "vue";
import type { Stage, Task } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";
import { canRolloutTasks } from "../taskPermissions";

export type TaskAction = "RUN" | "SKIP" | "CANCEL";

export interface TaskActionTarget {
  type: "tasks";
  tasks: Task[];
  stage: Stage;
}

export interface UseTaskActionsReturn {
  canRun: ComputedRef<boolean>;
  canSkip: ComputedRef<boolean>;
  canCancel: ComputedRef<boolean>;
  hasActions: ComputedRef<boolean>;
  showActionPanel: Ref<boolean>;
  currentAction: ComputedRef<TaskAction | undefined>;
  actionTarget: ComputedRef<TaskActionTarget | undefined>;
  runTask: () => void;
  skipTask: () => void;
  cancelTask: () => void;
  closeActionPanel: () => void;
}

// Statuses that allow run/skip actions
const RUNNABLE_STATUSES = [
  Task_Status.NOT_STARTED,
  Task_Status.CANCELED,
  Task_Status.FAILED,
];

// Statuses that allow cancel action
const CANCELABLE_STATUSES = [Task_Status.PENDING, Task_Status.RUNNING];

export const useTaskActions = (
  task: () => Task,
  stage: () => Stage | undefined
): UseTaskActionsReturn => {
  const showActionPanel = ref(false);
  const selectedAction = ref<TaskAction>();

  // Cache permission check - only re-evaluate when task name changes
  // This prevents flickering during poller refetches
  const canPerformActions = ref(false);
  watch(
    () => task().name,
    () => {
      canPerformActions.value = canRolloutTasks([task()]);
    },
    { immediate: true }
  );

  // Use stable primitive for status to prevent re-computation on object reference change
  const taskStatus = computed(() => task().status);

  const canRun = computed(
    () =>
      canPerformActions.value && RUNNABLE_STATUSES.includes(taskStatus.value)
  );

  const canSkip = computed(
    () =>
      canPerformActions.value && RUNNABLE_STATUSES.includes(taskStatus.value)
  );

  const canCancel = computed(
    () =>
      canPerformActions.value && CANCELABLE_STATUSES.includes(taskStatus.value)
  );

  const hasActions = computed(
    () => canRun.value || canSkip.value || canCancel.value
  );

  const currentAction = computed(() =>
    showActionPanel.value && selectedAction.value
      ? selectedAction.value
      : undefined
  );

  const actionTarget = computed((): TaskActionTarget | undefined => {
    const s = stage();
    if (!currentAction.value || !s) return undefined;
    return { type: "tasks", tasks: [task()], stage: s };
  });

  const runTask = () => {
    selectedAction.value = "RUN";
    showActionPanel.value = true;
  };

  const skipTask = () => {
    selectedAction.value = "SKIP";
    showActionPanel.value = true;
  };

  const cancelTask = () => {
    selectedAction.value = "CANCEL";
    showActionPanel.value = true;
  };

  const closeActionPanel = () => {
    showActionPanel.value = false;
  };

  return {
    canRun,
    canSkip,
    canCancel,
    hasActions,
    showActionPanel,
    currentAction,
    actionTarget,
    runTask,
    skipTask,
    cancelTask,
    closeActionPanel,
  };
};
