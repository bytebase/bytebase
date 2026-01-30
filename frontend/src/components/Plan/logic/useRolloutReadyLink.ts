import { computed } from "vue";
import { Task_Type } from "@/types/proto-es/v1/rollout_service_pb";
import { usePlanContext } from "./context";

/**
 * Composable that determines whether the "Rollout" link should be shown.
 * This link appears when:
 * - Rollout has been fetched (rollout ref is defined)
 * - Rollout does not contain database creation/export tasks
 */
export const useRolloutReadyLink = () => {
  const { rollout } = usePlanContext();

  const shouldShow = computed(() => {
    if (!rollout.value) {
      return false;
    }

    // Don't show for rollouts with database creation/export tasks
    const hasDatabaseCreateOrExportTasks = rollout.value.stages.some((stage) =>
      stage.tasks.some(
        (task) =>
          task.type === Task_Type.DATABASE_CREATE ||
          task.type === Task_Type.DATABASE_EXPORT
      )
    );
    if (hasDatabaseCreateOrExportTasks) {
      return false;
    }

    return true;
  });

  return {
    shouldShow,
  };
};
