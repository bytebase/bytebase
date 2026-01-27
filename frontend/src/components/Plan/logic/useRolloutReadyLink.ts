import { computed } from "vue";
import { useRoute } from "vue-router";
import {
  PROJECT_V1_ROUTE_PLAN_ROLLOUT,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
} from "@/router/dashboard/projectV1";
import { Task_Type } from "@/types/proto-es/v1/rollout_service_pb";
import { usePlanContext } from "./context";

/**
 * Composable that determines whether the "Rollout" link should be shown.
 * This link appears when:
 * - User is not already on the rollout tab
 * - Plan has a rollout (plan.hasRollout is true)
 * - Rollout does not contain database creation/export tasks
 */
export const useRolloutReadyLink = () => {
  const route = useRoute();
  const { plan, rollout } = usePlanContext();

  // Defined inside the function to avoid circular dependency issues
  // when the module is imported before the router is initialized
  const ROLLOUT_ROUTES = new Set([
    PROJECT_V1_ROUTE_PLAN_ROLLOUT,
    PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
    PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
  ]);

  const isOnRolloutTab = computed(() => {
    return ROLLOUT_ROUTES.has(route.name as string);
  });

  const shouldShow = computed(() => {
    // Hide if on rollout tab
    if (isOnRolloutTab.value) {
      return false;
    }

    // Show if plan has rollout
    if (!plan.value.hasRollout || !rollout.value) {
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
