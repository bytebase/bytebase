import type { ComputedRef, Ref } from "vue";
import { computed, ref } from "vue";
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
  // Action availability
  canRun: ComputedRef<boolean>;
  canSkip: ComputedRef<boolean>;
  canCancel: ComputedRef<boolean>;
  hasActions: ComputedRef<boolean>;

  // Panel state
  showActionPanel: Ref<boolean>;
  currentAction: ComputedRef<TaskAction | undefined>;
  actionTarget: ComputedRef<TaskActionTarget | undefined>;

  // Action handlers
  runTask: () => void;
  skipTask: () => void;
  cancelTask: () => void;
  closeActionPanel: () => void;
}

export const useTaskActions = (
  task: () => Task,
  stage: () => Stage | undefined,
  options: {
    readonly?: () => boolean;
  } = {}
): UseTaskActionsReturn => {
  const showActionPanel = ref(false);
  const selectedAction = ref<TaskAction>();

  const canPerformActions = computed(() => {
    if (options.readonly?.()) return false;
    return canRolloutTasks([task()]);
  });

  const canRun = computed(() => {
    if (!canPerformActions.value) return false;
    return [
      Task_Status.NOT_STARTED,
      Task_Status.CANCELED,
      Task_Status.FAILED,
    ].includes(task().status);
  });

  const canSkip = computed(() => {
    if (!canPerformActions.value) return false;
    return [
      Task_Status.NOT_STARTED,
      Task_Status.FAILED,
      Task_Status.CANCELED,
    ].includes(task().status);
  });

  const canCancel = computed(() => {
    if (!canPerformActions.value) return false;
    return [Task_Status.PENDING, Task_Status.RUNNING].includes(task().status);
  });

  const hasActions = computed(() => {
    return canRun.value || canSkip.value || canCancel.value;
  });

  const currentAction = computed((): TaskAction | undefined => {
    if (!showActionPanel.value || !selectedAction.value) return undefined;
    return selectedAction.value;
  });

  const actionTarget = computed((): TaskActionTarget | undefined => {
    const s = stage();
    if (!currentAction.value || !s) return undefined;
    return {
      type: "tasks",
      tasks: [task()],
      stage: s,
    };
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
