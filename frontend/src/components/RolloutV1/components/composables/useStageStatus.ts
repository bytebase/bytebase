import { type ComputedRef, computed } from "vue";
import { TASK_STATUS_FILTERS } from "@/components/RolloutV1/constants/task";
import type { Stage } from "@/types/proto-es/v1/rollout_service_pb";
import { Task_Status } from "@/types/proto-es/v1/rollout_service_pb";

export interface UseStageStatusReturn {
  stageStatus: ComputedRef<Task_Status>;
}

/**
 * Composable for computing stage status based on its tasks
 * Priority order follows TASK_STATUS_FILTERS: RUNNING > FAILED > PENDING > NOT_STARTED > CANCELED > DONE > SKIPPED
 */
export const useStageStatus = (
  stage: ComputedRef<Stage> | Stage,
  isCreated: boolean
): UseStageStatusReturn => {
  const stageStatus = computed(() => {
    const stageValue = "value" in stage ? stage.value : stage;

    if (!isCreated || stageValue.tasks.length === 0) {
      return Task_Status.STATUS_UNSPECIFIED;
    }

    const tasks = stageValue.tasks;

    // Check statuses in priority order using TASK_STATUS_FILTERS
    for (const status of TASK_STATUS_FILTERS) {
      if (tasks.some((t) => t.status === status)) {
        return status;
      }
    }

    // Fallback to DONE if no matching status found
    return Task_Status.DONE;
  });

  return {
    stageStatus,
  };
};
