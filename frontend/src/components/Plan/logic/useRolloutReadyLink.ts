import { computed } from "vue";
import { useRoute } from "vue-router";
import {
  PROJECT_V1_ROUTE_PLAN_ROLLOUT,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
} from "@/router/dashboard/projectV1";
import {
  Issue_ApprovalStatus,
  Issue_Type,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import { Task_Status, Task_Type } from "@/types/proto-es/v1/rollout_service_pb";
import { usePlanContext } from "./context";

const ROLLOUT_ROUTES = new Set([
  PROJECT_V1_ROUTE_PLAN_ROLLOUT,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_STAGE,
  PROJECT_V1_ROUTE_PLAN_ROLLOUT_TASK,
]);

const ACTIONABLE_TASK_STATUSES = new Set([
  Task_Status.NOT_STARTED,
  Task_Status.PENDING,
  Task_Status.RUNNING,
  Task_Status.FAILED,
  Task_Status.CANCELED,
]);

/**
 * Composable that determines whether the "Ready for Rollout" link should be shown.
 * This link appears when:
 * - User is not already on the rollout tab
 * - Rollout exists
 * - For plans without issue: always show when rollout exists
 * - For plans with issue: issue is a database change AND
 *   (issue is DONE, OR issue is OPEN and approved with actionable tasks)
 */
export const useRolloutReadyLink = () => {
  const route = useRoute();
  const { plan, issue, rollout } = usePlanContext();

  const isOnRolloutTab = computed(() => {
    return ROLLOUT_ROUTES.has(route.name as string);
  });

  const shouldShow = computed(() => {
    // Hide if on rollout tab
    if (isOnRolloutTab.value) {
      return false;
    }

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

    // For plans without issue but with rollout, show the link
    if (!issue.value && plan.value.hasRollout) {
      return true;
    }

    if (!issue.value) {
      return false;
    }

    // Only show for database change issues
    if (issue.value.type !== Issue_Type.DATABASE_CHANGE) {
      return false;
    }

    // For DONE issues with rollout, always show link for navigation
    if (issue.value.status === IssueStatus.DONE) {
      return true;
    }

    // For OPEN issues, check if approved and has actionable tasks
    if (issue.value.status === IssueStatus.OPEN) {
      // Check if issue is approved
      if (
        issue.value.approvalStatus !== Issue_ApprovalStatus.APPROVED &&
        issue.value.approvalStatus !== Issue_ApprovalStatus.SKIPPED
      ) {
        return false;
      }

      // Check if there's any task that needs action
      const allTasks = rollout.value.stages.flatMap((stage) => stage.tasks);
      return allTasks.some((task) => ACTIONABLE_TASK_STATUSES.has(task.status));
    }

    return false;
  });

  return {
    shouldShow,
  };
};
